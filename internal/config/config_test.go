package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAccountInContentRepo(t *testing.T) {
	plain := t.TempDir()
	checkout := t.TempDir()
	if err := os.Mkdir(filepath.Join(checkout, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		conf AppConfig
		want bool
	}{
		{"local data dir, users.json is not published", AppConfig{DataDir: plain}, false},
		{"git_repo configured", AppConfig{DataDir: plain, GitRepo: "https://github.com/me/blog-data.git"}, true},
		{"data dir is a git checkout", AppConfig{DataDir: checkout}, true},
		{
			"account moved into the config file",
			AppConfig{DataDir: checkout, GitRepo: "https://github.com/me/blog-data.git", AdminEmail: "me@example.com", AdminPasswordHash: "$2a$12$abc"},
			false,
		},
		{"half a config account does not count", AppConfig{DataDir: checkout, AdminEmail: "me@example.com"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.conf.AccountInContentRepo(); got != tt.want {
				t.Errorf("AccountInContentRepo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccountInContentRepoNilConfig(t *testing.T) {
	var conf *AppConfig
	if conf.AccountInContentRepo() {
		t.Error("a missing app section has no account to warn about")
	}
}

// The example configs are what people copy: they must load, and the options
// they spell out must reach the right fields.
func TestExampleConfigsLoad(t *testing.T) {
	for _, file := range []string{"../../conf/prod.yaml.example", "../../conf/dev.yaml.example"} {
		conf := LoadConfig(file)
		if conf.App == nil || conf.Server == nil {
			t.Fatalf("%s: missing sections", file)
		}
		if !conf.App.GitHubStatsEnabled() || !conf.App.PWAEnabled() {
			t.Errorf("%s: github_stats and pwa are on", file)
		}
		if conf.Server.ClientIPHeader != "" {
			t.Errorf("%s: client_ip_header = %q, want empty (gin's default headers)", file, conf.Server.ClientIPHeader)
		}
		if conf.Server.TrustedProxies == nil || len(conf.Server.TrustedProxies) != 0 {
			t.Errorf("%s: trusted_proxies = %#v, want an empty list", file, conf.Server.TrustedProxies)
		}
	}

	dir := t.TempDir()
	custom := filepath.Join(dir, "custom.yaml")
	yaml := "app:\n  name: x\n  github_stats: false\nserver:\n  http_port: 1\n  trusted_proxies: [\"172.18.0.0/16\", \"10.0.0.1\"]\n"
	if err := os.WriteFile(custom, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	conf := LoadConfig(custom)
	if conf.App.GitHubStatsEnabled() {
		t.Error("github_stats: false switches the numbers off")
	}
	if got := conf.Server.TrustedProxies; len(got) != 2 || got[0] != "172.18.0.0/16" || got[1] != "10.0.0.1" {
		t.Errorf("trusted_proxies = %#v", got)
	}
}

func TestGitSyncInterval(t *testing.T) {
	tests := map[string]time.Duration{"": DefaultGitSync, "off": 0, "OFF": 0, "0": 0, "false": 0, "5m": 5 * time.Minute, " 1h ": time.Hour}
	for value, want := range tests {
		got, err := (&AppConfig{GitSync: value}).GitSyncInterval()
		if err != nil || got != want {
			t.Errorf("git_sync %q = %v, %v; want %v", value, got, err, want)
		}
	}
	for _, bad := range []string{"soon", "10", "30s"} {
		got, err := (&AppConfig{GitSync: bad}).GitSyncInterval()
		if err == nil || got != DefaultGitSync {
			t.Errorf("git_sync %q = %v, %v; want the default and an error", bad, got, err)
		}
	}
}
