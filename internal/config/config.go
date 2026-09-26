package config

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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
		// GitHubStats makes /projects show the stars, language and last push of
		// the projects' GitHub repositories (fetched in the background every few
		// hours, public data only). Unset means on.
		GitHubStats *bool `mapstructure:"github_stats"`
		// GitHubUser is the account whose recent repositories the admin offers
		// to add as projects; empty means the owner of git_repo.
		GitHubUser string `mapstructure:"github_user"`
		// GitHubToken is optional: it raises the API limit from 60 to 5000
		// requests an hour. Any token works; it needs no permissions.
		GitHubToken string `mapstructure:"github_token"`
		// GitHubAPI is the API address, for GitHub Enterprise (and tests).
		GitHubAPI string `mapstructure:"github_api"`
		// GitSync is how often the content repository is synced with its remote
		// while the server runs ("10m" when empty, "off" to only sync at
		// startup, from the admin and through the webhook).
		GitSync string `mapstructure:"git_sync"`
		// GitWebhookSecret enables POST /api/hooks/git: a GitHub webhook signed
		// with this secret makes the server sync right after every push.
		GitWebhookSecret string `mapstructure:"git_webhook_secret"`
	}
	ServerConfig struct {
		// Host is the address to listen on. Empty (the default) means every interface; "127.0.0.1" keeps
		// the server private to the machine — what you want behind nginx / cloudflared and for previews.
		Host                    string        `mapstructure:"host"`
		HttpPort                uint32        `mapstructure:"http_port"`
		GracefulShutdownTimeout time.Duration `mapstructure:"graceful_shutdown_timeout"`
		// TrustedProxies are the addresses (IPs or CIDRs) of reverse proxies whose
		// X-Forwarded-For / X-Real-IP headers are believed. Empty means this machine
		// only (nginx or cloudflared on the same host). Anyone else could otherwise
		// pick their own IP and slip past the rate limits of login, likes and search.
		TrustedProxies []string `mapstructure:"trusted_proxies"`
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

// AccountInContentRepo reports whether the admin still signs in with the account kept in users.json of a
// git-backed content repository. That repository is usually public, which makes the password hash public
// too, so goblog keeps nagging (log at startup, banner in the admin) until the account has moved into the
// config file.
func (c *AppConfig) AccountInContentRepo() bool {
	if c == nil || c.ConfigAdmin() {
		return false
	}
	if c.GitRepo != "" {
		return true
	}
	info, err := os.Stat(filepath.Join(c.DataDir, ".git"))
	return err == nil && info.IsDir()
}

// PWAEnabled reports whether the site should offer its service worker.
func (c *AppConfig) PWAEnabled() bool {
	return c == nil || c.PWA == nil || *c.PWA
}

// DefaultGitSync is how often the content repository is synced when git_sync is not set.
const DefaultGitSync = 10 * time.Minute

// GitSyncInterval returns how often to sync the content repository; 0 means
// never on a schedule. An unreadable value falls back to the default.
func (c *AppConfig) GitSyncInterval() (time.Duration, error) {
	if c == nil {
		return DefaultGitSync, nil
	}
	switch value := strings.ToLower(strings.TrimSpace(c.GitSync)); value {
	case "":
		return DefaultGitSync, nil
	case "off", "false", "no", "0":
		return 0, nil
	default:
		d, err := time.ParseDuration(value)
		if err != nil || d < time.Minute {
			return DefaultGitSync, fmt.Errorf("git_sync %q: use a duration of at least 1m (e.g. \"10m\") or \"off\"", c.GitSync)
		}
		return d, nil
	}
}

// GitHubStatsEnabled reports whether project pages show live repository numbers.
func (c *AppConfig) GitHubStatsEnabled() bool {
	return c == nil || c.GitHubStats == nil || *c.GitHubStats
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
