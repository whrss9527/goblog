# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go Markdown blog system (`goblog`). Server-rendered web app with an admin backend and a public frontend, using **Gin + a Git-backed file store** (no database). Follows the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md).

## Commands

```bash
# Build (cross-compiles for linux/amd64 by default)
make build

# Build for macOS arm64
make mac

# Tidy dependencies
make tidy

# Format code
make fmt

# Run locally
./goblog -config=./conf/dev.yaml

# Run tests
go test ./...

# Run a single package's tests
go test ./pkg/utils/...

# Package for deployment
make tar
```

## Architecture

### Request Flow

`main.go` -> `config.LoadConfig` -> `view.InitTemplates` -> `routers.NewServer` -> `server.InitRouter(gin.Engine)` -> `gin.RunGin` (graceful shutdown).

The Gin engine is initialized in `internal/pkg/gin/gin.go` (CORS, error handling, graceful shutdown via `signal.NotifyContext`). Routes are wired up in `internal/routers/router.go`, which constructs the file-backed repository and injects it into all handlers.

### Layer Structure

- **Handlers** (`internal/handler/`) — HTTP handlers split into `admin/` (authenticated CRUD) and `front/` (public pages). Each handler struct receives repository interfaces via constructor injection.
- **Repository** (`internal/filestore/`) — `FileRepository` is the single concrete implementation of all repository interfaces (post / category / tag / page / book / user). Data is read once at startup into in-memory slices/maps and protected by `sync.RWMutex`. Mutations are persisted by writing files back into the data directory and (optionally) committing to its git remote.
- **Models** (`internal/pkg/model/`) — Plain Go structs (`Post`, `Category`, `Tag`, `Page`, `Book`, `User`). The `gorm:` struct tags are vestigial — they are not consumed by any active code, but kept for now to avoid touching every field rename.
- **Views** (`internal/pkg/view/`) — Go `html/template` rendering with startup-time caching. Call `view.InitTemplates()` before serving. Templates live in `tpl/` with `default/` (frontend), `admin/`, and `intro/` subdirectories.

### Key Patterns

- **Content storage**: All mutable data lives under `app.data_dir` on disk, organized as Markdown files with YAML frontmatter (see `internal/filestore/frontmatter.go`) plus JSON sidecar files for non-content entities. The directory is initialized by `git clone` of `app.git_repo` (uses `app.git_token` if private). Saves write the file then `git add && git commit && git push` if a git remote is configured.
- **Post ID**: Posts use UUID v4 (dashes removed) as primary key, stored as string. The `identity` field is a separate URL-friendly slug used in `/posts/:identity` routes.
- **Config**: Viper-based YAML config with defaults embedded in `internal/config/config.go`. Environment configs in `conf/dev.yaml` and `conf/prod.yaml` (both gitignored; use `conf/{dev,prod}.yaml.example` as templates).
- **Auth**: Admin routes use `gin-contrib/sessions` with signed cookie store via `middleware.AuthWithSession`. Session secret configured in `app.session_secret` (must be a real random value in production).
- **Markdown rendering**: posts and pages are rendered on the server by `internal/pkg/md2html` (`RenderArticle`: blackfriday +
  a normalizer that makes it read loosely written lists / fences the way marked — the editor.md preview — does, then goquery
  post-processing: sanitising, heading ids, task lists, shortcodes, marked-style typography). The feed uses the same HTML.
  `app.markdown_render: client` or content that needs editor.md-only features (```` ```flow ````, ```` ```seq ````, TeX)
  falls back to the in-browser editor.md renderer (`markdown-scripts` partial instead of `article-scripts`).
  When touching the renderer, keep `go test ./internal/pkg/md2html/` green — it pins the marked-compatibility rules.
- **SEO**: handlers set `canonical`; post pages add `og_image` (first image of the post), `published_iso` / `modified_iso`
  and `json_ld` (see `internal/handler/front/seo.go`). `view.RenderStatus` derives `page_title` and the default `og_image`.
- **Static caching**: `middleware.StaticCache` — fingerprinted URLs (`?v=`, from the `asset` template func) are immutable
  for a year, other static files are cached for a day.
- **PWA**: `internal/handler/front/pwa.go` serves `/manifest.webmanifest` (generated from config), `/sw.js`
  (`static/js/sw.js` with `self.__GOBLOG = {version, precache, offline}` prepended; the version is derived from the
  fingerprints of the precached files) and `/offline`. Strategy: navigations network-first with the visited copy as
  fallback, fingerprinted static files cache-first, other static files stale-while-revalidate; `/admin`, `/api`,
  `/random`, feeds and cross-origin requests are never touched. `app.pwa: false` turns `/sw.js` into a worker that
  clears the caches and unregisters itself. Icons in `static/icons` come from `static/logo.png` via `make icons`.
  New files a page needs offline must be added to `precachedAssets`.
- **Analytics**: the Google tag in `tpl/default/layout.html` is injected by a script that skips `localhost`, private
  IPv4 ranges and `*.local`, so previews and development never reach the statistics.
- **Counters**: view counts (`views.json`) and likes (`likes.json`) live in memory, are flushed to the data dir every 5 minutes and committed/pushed hourly. Both are keyed by post slug and migrate on slug rename.
- **JSON API**: `GET /api/search?q=` (instant search), `POST /api/posts/:identity/like`; both are rate limited per IP via `middleware.NewRateLimiter`. `GET /random` redirects to a random post.
- **Cron jobs**: Heatmap data aggregation runs hourly via `robfig/cron`, writing `heatmap.txt`.
- **Feed**: RSS/Atom generated at startup and served at `/feed.xml`. Sitemap generated at startup and served at `/sitemap.xml`.
- **Logging**: Unified on `log/slog`. `internal/pkg/slogx/` provides a custom handler with trace ID support.
- **Graceful shutdown**: `gin.RunGin` uses `http.Server` + `signal.NotifyContext(SIGINT, SIGTERM)`. Timeout configured via `server.graceful_shutdown_timeout`.
- **Delete operations**: All delete routes use POST method to prevent CSRF via GET.

### Front-end conventions

- `tpl/default/layout.html` exposes three blocks: `head`, `content`, `scripts`. Page-specific JS must go
  into `scripts` (rendered after `enhance.js`). Shared partials: `icons.html` (inline SVG icons via
  `{{template "icon-eye"}}`), `markdown.html` (editor.md renderer + giscus), `heatmap.html`.
- No CSS/JS framework on the public site: `static/css/style.css` (design tokens in `:root` /
  `[data-theme="dark"]`) and `static/js/enhance.js` (vanilla). jQuery is loaded only by
  `markdown-scripts` (the client-side rendering fallback) because editor.md needs it; server-rendered
  articles load just `article.js` + prettify (`article-scripts`). Bootstrap is used by the admin only.
- Reference local assets through `{{asset "/static/..."}}` so they get a content fingerprint.
  Files that also live on the external CDN (`app.cdn`) keep using `{{.cdn}}/...`.
- Every handler should set `data["nav"]` (`home|archive|tags|reading|about`) for the active nav item. Keys the
  layout prints unconditionally need a default in `view.RenderStatus`, otherwise html/template prints `<no value>`.
- Release flow: bump `internal/version/version.go`, add a section to `CHANGELOG.md`, one commit per version.

### Admin conventions

- Every admin template is parsed together with `tpl/admin/_partials.html` and the template func map
  (`asset`, `formatTime`, `dict`, `pathEscape`, …). List and form pages use `admin-head` / `admin-open` /
  `admin-close`; the two Markdown editors (`posts/add`, `pages/add`) use `editor-head` / `editor-scripts`.
- Vendor files (SB Admin / Bootstrap 5, simple-datatables, editor.md) come from `{{.cdn}}`; anything goblog adds
  lives in `static/admin/` (`css/goblog-admin.css`, `js/goblog-admin.js`, `js/goblog-editor.js`) and is referenced
  with `{{asset "/static/admin/..."}}` — the CDN bucket does not have these files.
- `view.AdminRenderStatus` sets `site_name`, `site_version`, `this_year` (prefixed on purpose: handlers use plain
  keys such as `name` for form values).
- A rejected save re-renders the editor (HTTP 422) with everything the author typed plus `error`; never send the
  author to a bare error page. Slug rules live in `internal/handler/admin/validate.go`; `filestore.PostSave` /
  `PageSave` have their own last-line checks (`ErrInvalidSlug`, `ErrSlugTaken`). An unchanged legacy slug always passes.
- **Drafts** (`internal/filestore/draft.go`, `repository.DraftRepository`): unpublished posts in `<data_dir>/.drafts`,
  read from disk on demand, excluded from git through `.git/info/exclude` (`ensureDraftsPrivate` — a draft is not
  written when that cannot be guaranteed). They keep tag *names* (`tag_names` frontmatter) so no tag is created in the
  public `tags.json` before publication. `action=draft` in `PostSave` only applies to unpublished posts; publishing
  deletes the draft. `front.RenderPostPreview` renders a draft with the public template (`.preview`).
- **Admin account**: `app.admin_email` + `app.admin_password_hash` win over `users.json` (which sits in the — usually
  public — content repository). Login failures all look the same (one message, bcrypt always runs).
- **Tags / categories**: `filestore.RenameTag` renames, and merges when the name already belongs to another tag
  (posts are retagged on disk first, memory second, `updated_at` untouched); `DeleteTag` takes the tag off every post.
  Tag counts are recounted at startup (posts may arrive through git, not the admin). `CategoryDelete` refuses while
  posts use the category (`*filestore.CategoryInUseError`). Names go through `checkLabel` (no commas: the editor's
  tag field splits on them). List pages report refusals in place (`renderList(ctx, status, problem)`).
- Content files are written in the format the migrated repository uses (`tag_ids: [1, 2]`, one trailing newline):
  `postToFrontmatter(parsePost(x)) == x` for files the app wrote, so saving shows up in git as the real change only.
- `view.AdminRenderStatus` sets `account_warning` while the admin account still comes from `users.json` of a
  git-backed data dir (`accountInContentRepo`); `_partials.html` shows it on every list / form page.
- `model.Tag` / `Category` / `User` carry JSON names matching the migrated data files; do not remove them, or the
  next save rewrites the files with Go field names and drops the timestamps.
- The editor keeps a local backup in `localStorage` (`goblog:draft:<kind>:<id|new>`); a successful save redirects to
  the list with `?saved=<slug>`, which is what clears the draft.

### Package Map

- `pkg/` — Generic libraries (cache, utils, exception handling).
- `internal/pkg/` — App-specific internals (gin setup, markdown-to-HTML, models, slogx structured logging, view rendering).

## Server

Port configured via `server.http_port`, listen address via `server.host` (empty: every interface). `gin.RunGin` returns
an error when it cannot listen and `main` exits non-zero. Health check at `GET /ping`.

## Deployment

Production target is systemd. See `conf/goblog.service` for the unit template and `README.md` for the full deploy walkthrough. There is also a `Dockerfile` for container-based deployment if desired.
