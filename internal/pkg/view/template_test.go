package view

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssetURL(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "static", "css"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "static", "css", "a.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	got := AssetURL("/static/css/a.css")
	if !strings.HasPrefix(got, "/static/css/a.css?v=") || len(got) != len("/static/css/a.css?v=")+10 {
		t.Fatalf("expected fingerprinted URL, got %q", got)
	}
	if again := AssetURL("/static/css/a.css"); again != got {
		t.Errorf("fingerprint must be stable, got %q then %q", got, again)
	}
	if missing := AssetURL("/static/css/missing.css"); missing != "/static/css/missing.css" {
		t.Errorf("missing file should be returned unchanged, got %q", missing)
	}
	if external := AssetURL("https://cdn.example.com/x.css"); external != "https://cdn.example.com/x.css" {
		t.Errorf("non-local URL should be returned unchanged, got %q", external)
	}
}
