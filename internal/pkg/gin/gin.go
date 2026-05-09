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

func RunGin(router *gin.Engine, port uint32, shutdownTimeout time.Duration) {
	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
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
