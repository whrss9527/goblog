package gin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"strconv"
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

// RunGin serves until SIGINT / SIGTERM and returns nil after a graceful shutdown. host "" listens on
// every interface; "127.0.0.1" keeps the server private to the machine (behind nginx / cloudflared, or
// for a local preview). A server that cannot listen — port taken, address not available — returns the
// error: it used to log it and then idle forever, which systemd reported as a healthy service.
func RunGin(router *gin.Engine, host string, port uint32, shutdownTimeout time.Duration) error {
	addr := net.JoinHostPort(host, strconv.FormatUint(uint64(port), 10))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	srv := &http.Server{
		Handler: HeadAsGet(router),
		// bound how long a client may take to send its request headers and how
		// long idle keep-alive connections are kept around
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("server started", "addr", listener.Addr().String())
		serveErr <- srv.Serve(listener)
	}()

	select {
	case <-ctx.Done():
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	}
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
	return nil
}
