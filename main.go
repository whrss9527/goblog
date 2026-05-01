package main

import (
	"flag"
	"log/slog"

	"goblog/internal/config"
	ginpkg "goblog/internal/pkg/gin"
	"goblog/internal/pkg/view"
	"goblog/internal/routers"
	"goblog/pkg/cache"
)

var configPath string

func main() {
	flag.StringVar(&configPath, "config", "./conf/dev.yaml", "yaml config file to be load, e.g: -config=/local/config.yaml")
	flag.Parse()
	conf := config.LoadConfig(configPath)

	if conf.Server == nil || conf.Server.HttpPort == 0 {
		slog.Error("server.http_port is not configured")
		return
	}

	view.InitTemplates()
	server := routers.NewServer(conf)
	cache.InitBigCacheConfig()
	router := ginpkg.InitGinConfig()
	server.InitRouter(router)
	ginpkg.RunGin(router, conf.Server.HttpPort, conf.Server.GracefulShutdownTimeout)
}
