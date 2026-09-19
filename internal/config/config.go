package config

import (
	"bytes"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/viper"

	"goblog/internal/util"
)

type (
	Config struct {
		LogLevel string        `mapstructure:"log_level"`
		Server   *ServerConfig `mapstructure:"server"`
		App      *AppConfig    `mapstructure:"app"`
	}
	AppConfig struct {
		Name          string `mapstructure:"name"`
		Version       bool   `mapstructure:"version"`
		Mode          string `mapstructure:"mode"`
		Addr          string `mapstructure:"addr"`
		Host          string `mapstructure:"host"`
		Cdn           string `mapstructure:"cdn"`
		Music         string `mapstructure:"music"`
		SessionSecret string `mapstructure:"session_secret"`
		DataDir       string `mapstructure:"data_dir"`
		GitRepo       string `mapstructure:"git_repo"`
		GitToken      string `mapstructure:"git_token"`
		// MarkdownRender selects where articles are rendered: "server" (default,
		// HTML in the response) or "client" (editor.md in the browser, the
		// pre-1.6 behaviour). Posts using flow charts, sequence diagrams or TeX
		// always fall back to the client renderer.
		MarkdownRender string `mapstructure:"markdown_render"`
		// Description is the one-line summary of the site, used for the home
		// page's title, its <meta name="description"> and structured data.
		Description string `mapstructure:"description"`
		// PWA switches the web app manifest and the service worker (offline
		// reading, instant static assets) on or off. Unset means on; setting it
		// to false also retires service workers browsers already installed.
		PWA *bool `mapstructure:"pwa"`
		// AdminEmail and AdminPasswordHash (bcrypt, see "goblog -hash-password")
		// define the admin account in the config file. When both are set they
		// are the only way in and users.json in the content repository is
		// ignored — that repository is often public, and a password hash has no
		// business being downloadable.
		AdminEmail        string `mapstructure:"admin_email"`
		AdminPasswordHash string `mapstructure:"admin_password_hash"`
	}
	ServerConfig struct {
		HttpPort                uint32        `mapstructure:"http_port"`
		GracefulShutdownTimeout time.Duration `mapstructure:"graceful_shutdown_timeout"`
	}
)

var (
	defaultConfigYamlString = []byte(`
log_level: debug
server:
  http_port: 9091
  graceful_shutdown_timeout: 15s
`)
)

// ConfigAdmin reports whether the admin account comes from the config file.
func (c *AppConfig) ConfigAdmin() bool {
	return c != nil && c.AdminEmail != "" && c.AdminPasswordHash != ""
}

// PWAEnabled reports whether the site should offer its service worker.
func (c *AppConfig) PWAEnabled() bool {
	return c == nil || c.PWA == nil || *c.PWA
}

func LoadConfig(config string) *Config {
	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBuffer(defaultConfigYamlString))
	if err != nil {
		slog.Error("LoadConfig", "err", err)
		util.Exit(1)
	}
	_, err = os.Stat(config)
	if err == nil {
		v.SetConfigFile(config)
		if err = v.MergeInConfig(); err != nil {
			slog.Error("LoadConfig", "err", err)
			util.Exit(1)
		}
	}
	conf := &Config{}
	err = v.Unmarshal(&conf)
	if err != nil {
		slog.Error("LoadConfig", "err", err)
		util.Exit(1)
	}
	return conf
}
