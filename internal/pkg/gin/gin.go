package gin

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"goblog/pkg/exception"
)

func InitGinConfig(mode string) *gin.Engine {
	switch mode {
	case "debug":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()
	router.Use(exception.ErrHandle)
	router.Use(cors.Default())
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, nil)
	})
	return router
}

// OriginalMethodHeader is set by HeadAsGet on requests that arrived as HEAD.
const OriginalMethodHeader = "X-Goblog-Original-Method"

// HeadAsGet lets HEAD requests (uptime monitors, link checkers, curl -I) reach
// the GET routes instead of ending in a 404. The routes see a copy of the
// request; net/http still knows the original was a HEAD and leaves the body
// out of the response. Handlers that count visits check OriginalMethodHeader.
func HeadAsGet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del(OriginalMethodHeader) // only ever set here
		if r.Method == http.MethodHead {
			r = r.Clone(r.Context())
			r.Method = http.MethodGet
			r.Header.Set(OriginalMethodHeader, http.MethodHead)
		}
		next.ServeHTTP(w, r)
	})
}

func RunGin(router *gin.Engine, port uint32, shutdownTimeout time.Duration) {
	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Addr:    addr,
		Handler: HeadAsGet(router),
		// bound how long a client may take to send its request headers and how
		// long idle keep-alive connections are kept around
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server started", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server listen failed", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server...")

	if shutdownTimeout == 0 {
		shutdownTimeout = 15 * time.Second
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "err", err)
	}
	slog.Info("server exited")
}
