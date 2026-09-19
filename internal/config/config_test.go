package config

import (
	"os"
	"path/filepath"
	"testing"
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
