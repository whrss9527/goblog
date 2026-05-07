package routers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/filestore"
	"goblog/internal/handler/admin"
	"goblog/internal/handler/front"
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

func (server *Server) InitRouter(router *gin.Engine) (cleanup func()) {
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
	feedHandler.GenerateFeedXml()
	sitemapHandler := front.NewSitemapHandler(repo, server.config.App.Host)
	sitemapHandler.GenerateSitemap()
	heatmapHandler := admin.NewHeatMapHandler(repo)
	heatmapHandler.RunTask(repo.Done())
	postHandler := admin.NewPostHandler(repo, repo, repo, feedHandler, sitemapHandler, server.config)
	frontPostHandler := front.NewPostHandler(repo, repo, repo, server.config)
	authHandler := admin.NewAuthHandler(repo, server.config)
	categoryHandler := admin.NewCategoryHandler(repo, server.config)
	archiveHandler := front.NewArchiveHandler(repo, server.config)

	pageHandler := admin.NewPageHandler(repo, server.config)
	frontPageHandler := front.NewPageHandler(repo, server.config)
	tagHandler := admin.NewTagHandler(repo, server.config)
	frontTagHandler := front.NewTagHandler(repo, server.config)
	bookHandler := admin.NewBookHandler(repo, server.config)
	frontBookHandler := front.NewBookHandler(repo, server.config)

	loginLimiter := middleware.NewRateLimiter(5, 15*time.Minute)

	router.StaticFS("/static/", http.Dir("static"))

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
		manage.GET("/posts/add", postHandler.PostAdd)
		manage.POST("/posts/save", postHandler.PostSave)
		manage.POST("/posts/delete/:id", postHandler.PostDelete)
		manage.GET("/pages", pageHandler.PageList)
		manage.GET("/pages/add", pageHandler.PageAdd)
		manage.POST("/pages/save", pageHandler.PageSave)
		manage.POST("/pages/delete/:id", pageHandler.PageDelete)
		manage.GET("/categories", categoryHandler.CategoryList)
		manage.GET("/categories/add", categoryHandler.CategoryAdd)
		manage.POST("/categories/save", categoryHandler.CategorySave)
		manage.POST("/categories/delete", categoryHandler.CategoryDelete)
		manage.GET("/tags", tagHandler.TagList)
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
		client.GET("/reading", frontBookHandler.ReadingList)
		client.GET("/pages/:id", frontPageHandler.Page)
		client.GET("/tags", frontTagHandler.Tag)
		client.GET("/archive", archiveHandler.Archive)
		client.GET("/feed.xml", feedHandler.GetFeedXml)
		client.GET("/feed", feedHandler.GetFeedXml)
		client.GET("/sitemap.xml", sitemapHandler.GetSitemap)
		client.GET("/intro", frontPostHandler.Intro)
		client.GET("/robots.txt", feedHandler.GetRobotTxt)
	}
	return func() { repo.Close() }
}
