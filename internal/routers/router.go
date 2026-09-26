package routers

import (
	"log"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/filestore"
	"goblog/internal/handler/admin"
	"goblog/internal/handler/front"
	"goblog/internal/pkg/github"
	"goblog/internal/pkg/view"
	"goblog/internal/repository"
	"goblog/internal/routers/middleware"
)

func faviconHandler(ctx *gin.Context) {
	http.ServeFile(ctx.Writer, ctx.Request, "static/favicon.ico")
}

type Server struct {
	config *config.Config
}

func NewServer(config *config.Config) *Server {
	return &Server{
		config: config,
	}
}

// localProxies are trusted when server.trusted_proxies is empty: nginx or
// cloudflared running on the same machine.
var localProxies = []string{"127.0.0.1", "::1"}

func (server *Server) InitRouter(router *gin.Engine) (cleanup func()) {
	// Only proxies we know may tell us the client's address. gin believes any
	// X-Forwarded-For by default, and its first entry is whatever the client
	// wrote there (Cloudflare appends the real address, it does not replace).
	proxies := localProxies
	if server.config.Server != nil && len(server.config.Server.TrustedProxies) > 0 {
		proxies = server.config.Server.TrustedProxies
	}
	if err := router.SetTrustedProxies(proxies); err != nil {
		log.Fatal("server.trusted_proxies: ", err)
	}
	router.Use(warnUntrustedProxy(), middleware.SecurityHeaders)

	secret := server.config.App.SessionSecret
	if secret == "" {
		secret = "goblog-default-secret-change-me"
	}
	store := cookie.NewStore([]byte(secret))
	isHTTPS := strings.HasPrefix(server.config.App.Host, "https://")
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   isHTTPS,
		SameSite: http.SameSiteLaxMode,
	})
	router.Use(sessions.Sessions("goblog_session", store))

	repo, err := filestore.NewFileRepository(server.config.App.DataDir, server.config.App.GitRepo, server.config.App.GitToken)
	if err != nil {
		log.Fatal("init file repository failed: ", err)
	}

	feedHandler := front.NewFeedHandler(repo, server.config.App.Host, server.config.App.Name)
	feedHandler.Description = server.config.App.Description
	feedHandler.GenerateFeedXml()
	sitemapHandler := front.NewSitemapHandler(repo, server.config.App.Host)
	sitemapHandler.Pages = repo
	sitemapHandler.Projects = repo
	sitemapHandler.GenerateSitemap()
	heatmapHandler := admin.NewHeatMapHandler(repo)
	heatmapHandler.RunTask(repo.Done())
	postHandler := admin.NewPostHandler(repo, repo, repo, repo, feedHandler, sitemapHandler, server.config)
	postHandler.Sync = repo
	githubClient := github.NewClient(server.config.App.GitHubAPI, server.config.App.GitHubToken)
	var githubStats *github.Stats // stays nil (no numbers, no requests) when app.github_stats is off
	if server.config.App.GitHubStatsEnabled() {
		githubStats = github.NewStats(githubClient)
		githubStats.Run(repo.Done(), projectRepos(repo), githubStatsInterval)
	}
	if projects, err := repo.GetProjects(); err == nil {
		view.SetProjectCount(len(projects))
	}
	projectCatalog := &front.ProjectCatalog{Projects: repo, Posts: repo, Stats: githubStats}
	frontPostHandler := front.NewPostHandler(repo, repo, repo, server.config)
	frontPostHandler.Projects = projectCatalog
	frontProjectHandler := front.NewProjectHandler(projectCatalog, server.config)
	projectHandler := admin.NewProjectHandler(repo, repo, githubClient, githubStats, sitemapHandler, server.config)
	searchHandler := front.NewSearchHandler(repo, repo, repo)
	statsHandler := front.NewStatsHandler(repo, repo, repo, repo, server.config)
	pwaHandler := front.NewPWAHandler(server.config)
	if archive, err := repo.GetPostsArchive(); err == nil && len(archive) > 0 {
		// footer "running for N days" counts from the very first post
		view.SetSiteSince(archive[len(archive)-1].CreatedAt)
	}
	authHandler := admin.NewAuthHandler(repo, server.config)
	categoryHandler := admin.NewCategoryHandler(repo, server.config)
	archiveHandler := front.NewArchiveHandler(repo, server.config)

	pageHandler := admin.NewPageHandler(repo, server.config)
	pageHandler.Sitemap = sitemapHandler
	frontPageHandler := front.NewPageHandler(repo, server.config)
	tagHandler := admin.NewTagHandler(repo, server.config)
	frontTagHandler := front.NewTagHandler(repo, server.config)
	frontTagHandler.Heatmap = heatmapHandler.JSON
	bookHandler := admin.NewBookHandler(repo, server.config)
	frontBookHandler := front.NewBookHandler(repo, server.config)

	// Content pushed to the repository from elsewhere shows up without a restart: everything derived
	// from it is rebuilt after each sync that brought something new.
	repo.OnReload(func() {
		feedHandler.GenerateFeedXml()
		sitemapHandler.GenerateSitemap()
		heatmapHandler.NewJob().Run()
		if projects, err := repo.GetProjects(); err == nil {
			view.SetProjectCount(len(projects))
		}
		if archive, err := repo.GetPostsArchive(); err == nil && len(archive) > 0 {
			view.SetSiteSince(archive[len(archive)-1].CreatedAt)
		}
		githubStats.Kick()
	})
	syncEvery, err := server.config.App.GitSyncInterval()
	if err != nil {
		slog.Error("invalid app.git_sync, using the default", "err", err, "default", syncEvery)
	}
	repo.StartSync(syncEvery)
	gitWebhook := &front.GitWebhook{Secret: server.config.App.GitWebhookSecret, RequestSync: repo.RequestSync}

	loginLimiter := middleware.NewRateLimiter(5, 15*time.Minute)
	hookLimiter := middleware.NewRateLimiter(30, time.Minute)
	searchLimiter := middleware.NewRateLimiter(90, time.Minute)
	likeLimiter := middleware.NewRateLimiter(20, time.Minute)

	router.Use(middleware.StaticCache("/static/", "/covers/", "/favicon.ico"))
	router.StaticFS("/static/", http.Dir("static"))
	// Book covers live in the content repo (data_dir/covers) and are referenced
	// as /covers/<file> from books.json.
	router.Static("/covers", filepath.Join(server.config.App.DataDir, "covers"))
	router.NoRoute(front.NotFound(server.config.App))

	manage := router.Group("admin")
	{
		manage.GET("/login", authHandler.Login)
		manage.POST("/register", authHandler.Register)
		manage.POST("/sign-in", loginLimiter.Limit(), authHandler.Signin)
		manage.GET("/signup", authHandler.Signup)
		manage.Use(middleware.AuthWithSession)
		manage.Use(middleware.CSRFProtect)
		manage.GET("/logout", authHandler.Logout)
		manage.GET("/", postHandler.PostList)
		manage.POST("/sync", postHandler.SyncNow)
		manage.GET("/posts/add", postHandler.PostAdd)
		manage.POST("/posts/save", postHandler.PostSave)
		manage.POST("/posts/delete/:id", postHandler.PostDelete)
		manage.GET("/posts/preview", postHandler.DraftPreview)
		manage.POST("/drafts/delete/:slug", postHandler.DraftDelete)
		manage.GET("/pages", pageHandler.PageList)
		manage.GET("/pages/add", pageHandler.PageAdd)
		manage.POST("/pages/save", pageHandler.PageSave)
		manage.POST("/pages/delete/:id", pageHandler.PageDelete)
		manage.GET("/categories", categoryHandler.CategoryList)
		manage.GET("/categories/add", categoryHandler.CategoryAdd)
		manage.POST("/categories/save", categoryHandler.CategorySave)
		manage.POST("/categories/delete", categoryHandler.CategoryDelete)
		manage.GET("/tags", tagHandler.TagList)
		manage.GET("/tags/edit", tagHandler.TagEdit)
		manage.POST("/tags/save", tagHandler.TagSave)
		manage.POST("/tags/delete", tagHandler.TagDelete)
		manage.GET("/projects", projectHandler.ProjectList)
		manage.GET("/projects/add", projectHandler.ProjectAdd)
		manage.POST("/projects/save", projectHandler.ProjectSave)
		manage.POST("/projects/delete/:id", projectHandler.ProjectDelete)
		manage.GET("/projects/github", projectHandler.GitHubRepo)
		manage.GET("/projects/github/suggestions", projectHandler.GitHubSuggestions)
		manage.GET("/books", bookHandler.BookList)
		manage.GET("/books/add", bookHandler.BookAdd)
		manage.POST("/books/save", bookHandler.BookSave)
		manage.POST("/books/delete/:id", bookHandler.BookDelete)
	}
	client := router.Group("")
	{
		client.GET("/", frontPostHandler.Index)
		client.GET("/favicon.ico", faviconHandler)
		client.GET("/posts/:identity", frontPostHandler.PostInfo)
		client.GET("/random", frontPostHandler.Random)
		client.GET("/api/search", searchLimiter.Limit(), searchHandler.Search)
		client.POST("/api/posts/:identity/like", likeLimiter.Limit(), frontPostHandler.Like)
		client.POST("/api/hooks/git", hookLimiter.Limit(), gitWebhook.Handle)
		client.GET("/reading", frontBookHandler.ReadingList)
		client.GET("/projects", frontProjectHandler.Projects)
		client.GET("/pages/:id", frontPageHandler.Page)
		client.GET("/tags", frontTagHandler.Tag)
		client.GET("/archive", archiveHandler.Archive)
		client.GET("/stats", statsHandler.Stats)
		client.GET("/feed.xml", feedHandler.GetFeedXml)
		client.GET("/feed", feedHandler.GetFeedXml)
		client.GET("/sitemap.xml", sitemapHandler.GetSitemap)
		client.GET("/intro", frontPostHandler.Intro)
		client.GET("/robots.txt", feedHandler.GetRobotTxt)
		client.GET("/manifest.webmanifest", pwaHandler.Manifest)
		client.GET("/sw.js", pwaHandler.ServiceWorker)
		client.GET("/offline", pwaHandler.Offline)
	}
	return func() { repo.Close() }
}

// warnUntrustedProxy logs once when requests come through a proxy on the local
// network that server.trusted_proxies does not name (typically cloudflared or
// nginx in a Docker container): the rate limits then see the proxy's address
// instead of the visitors', and all of them share one limit.
func warnUntrustedProxy() gin.HandlerFunc {
	var once sync.Once
	return func(ctx *gin.Context) {
		if ctx.GetHeader("X-Forwarded-For") != "" || ctx.GetHeader("X-Real-IP") != "" {
			remote := ctx.RemoteIP()
			if ip := net.ParseIP(remote); ip != nil && ip.IsPrivate() && ctx.ClientIP() == remote {
				once.Do(func() {
					slog.Warn("a proxy that is not in server.trusted_proxies forwards requests: all visitors share its rate limits; "+
						"add its address to server.trusted_proxies if it is yours", "proxy", remote)
				})
			}
		}
		ctx.Next()
	}
}

// githubStatsInterval is how often the numbers of the projects' repositories are refreshed.
const githubStatsInterval = 6 * time.Hour

// projectRepos lists the source addresses of all projects, for the GitHub numbers.
func projectRepos(projects repository.ProjectRepository) func() []string {
	return func() []string {
		list, err := projects.GetProjects()
		if err != nil {
			return nil
		}
		repos := make([]string, 0, len(list))
		for _, p := range list {
			if p.Repo != "" {
				repos = append(repos, p.Repo)
			}
		}
		return repos
	}
}
