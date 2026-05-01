# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go Markdown blog system (`goblog`). Server-rendered web app with an admin backend and a public frontend, using Gin + GORM + MySQL. Follows [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md).

## Commands

```bash
# Build (cross-compiles for linux/amd64 by default)
make build

# Build for macOS arm64
make mac

# Download dependencies
make tidy

# Format code
make fmt

# Run locally (requires MySQL)
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

`main.go` -> `config.LoadConfig` -> `view.InitTemplates` -> `routers.NewServer` -> `server.InitRouter(gin.Engine)` -> `gin.RunGin` (graceful shutdown)

The Gin engine is initialized in `internal/pkg/gin/gin.go` (CORS, error handling, graceful shutdown via signal). Routes are registered in `internal/routers/router.go`, which wires up all handlers with their repository dependencies.

### Layer Structure

- **Handlers** (`internal/handler/`) — HTTP handlers split into `admin/` (authenticated CRUD) and `front/` (public-facing pages). Each handler struct receives repository interfaces via constructor injection.
- **Repository** (`internal/mysql/`) — `GormRepository` implements all repository interfaces (`PostRepository`, `CategoryRepository`, `TagRepository`, `PageRepository`). Each entity's interface and queries live in its own file (e.g., `post.go`, `tag.go`).
- **Models** (`internal/pkg/model/`) — GORM structs with explicit `TableName()` methods. Post content is stored in a separate `post_content` table.
- **Views** (`internal/pkg/view/`) — Go `html/template` rendering with startup-time caching. Call `view.InitTemplates()` before serving. Templates live in `tpl/` with `default/` (frontend), `admin/`, and `intro/` subdirectories.

### Key Patterns

- **MySQL read/write splitting**: `internal/mysql/mysql.go` configures GORM `dbresolver` with separate writer and reader endpoints from config.
- **Post ID**: Posts use UUID v4 (dashes removed) as primary key, stored as string. The `identity` field is a separate URL-friendly slug used in `/posts/:identity` routes.
- **Tag storage**: Tag IDs are stored as a JSON array in the `tag_ids` column, queried via MySQL `JSON_CONTAINS`.
- **Config**: Viper-based YAML config with defaults embedded in `internal/config/config.go`. Environment configs in `conf/dev.yaml` and `conf/prod.yaml` (gitignored; use `conf/dev.yaml.example` as template).
- **Auth**: Admin routes use `gin-contrib/sessions` with signed cookie store via `middleware.AuthWithSession`. Session secret configured in `app.session_secret`.
- **Cron jobs**: Heatmap data generation runs hourly via `robfig/cron`, writing to `heatmap.txt`.
- **Feed**: RSS/Atom feed generated at startup via `GetPostsWithContent()` (single JOIN query) and served at `/feed.xml`.
- **Logging**: Unified on `log/slog` throughout the codebase.
- **Graceful shutdown**: `gin.RunGin` uses `http.Server` + `signal.NotifyContext(SIGINT, SIGTERM)`. Timeout configured via `server.graceful_shutdown_timeout`.
- **Delete operations**: All delete routes use POST method to prevent CSRF via GET.

### Package Map

- `pkg/` — Reusable libraries (cache, redis, mail, utils, logging, exception handling)
- `internal/pkg/` — App-specific internals (gin setup, markdown-to-HTML, models, slogx structured logging, view rendering)
- `internal/pkg/slogx/` — Custom `slog` handler with trace ID support, used by GORM logger

## Database

MySQL 8.0, charset `utf8mb4`. Schema in `blog.sql`. Tables: `post`, `post_content`, `category`, `tag`, `page`, `user`.

## Server

Port configured via `server.http_port` in YAML config. Health check at `GET /ping`.
