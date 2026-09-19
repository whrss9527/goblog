package front

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"goblog/internal/config"
	"goblog/internal/pkg/model"
)

func TestFirstImage(t *testing.T) {
	tests := []struct {
		name, markdown, want string
	}{
		{"markdown image", "text\n\n![alt](https://img.example/a.png \"title\")\n![b](https://img.example/b.png)", "https://img.example/a.png"},
		{"html image first", "<img width=\"10\" src='/covers/x.jpg'>\n\n![alt](https://img.example/a.png)", "/covers/x.jpg"},
		{"images in code do not count", "```md\n![no](https://img.example/code.png)\n```\n\n![yes](https://img.example/real.png)", "https://img.example/real.png"},
		{"angle bracket destination", "![a](<https://img.example/with space.png>)", "https://img.example/with"},
		{"no image", "just [a link](https://example.com)", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstImage(tt.markdown); got != tt.want {
				t.Errorf("firstImage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListCanonical(t *testing.T) {
	tests := []struct {
		categoryId, tagId string
		page              int
		want              string
	}{
		{"", "", 1, "https://blog.example.com/"},
		{"", "", 0, "https://blog.example.com/"},
		{"", "", 3, "https://blog.example.com/?page=3"},
		{"2", "", 1, "https://blog.example.com/?category_id=2"},
		{"", "7", 2, "https://blog.example.com/?page=2&tag_id=7"},
	}
	for _, tt := range tests {
		if got := listCanonical("https://blog.example.com/", tt.categoryId, tt.tagId, tt.page); got != tt.want {
			t.Errorf("listCanonical(%q, %q, %d) = %q, want %q", tt.categoryId, tt.tagId, tt.page, got, tt.want)
		}
	}
}

func TestPostJSONLD(t *testing.T) {
	conf := &config.AppConfig{Name: "测试博客", Host: "https://blog.example.com", Cdn: "/static"}
	created := time.Date(2024, 3, 16, 10, 0, 0, 0, time.FixedZone("CST", 8*3600))
	post := model.Post{Title: `A </script><script>alert(1)</script> title`, Identity: "hello", CreatedAt: created, UpdatedAt: created.Add(-time.Hour), WordCount: 1200, CategoryName: "技术"}

	raw := string(postJSONLD(conf, post, "摘要", "https://img.example/a.png", []model.Tag{{Name: "go"}, {Name: "测试"}}))
	if strings.Contains(raw, "</script>") || strings.Contains(raw, "<script>") {
		t.Fatalf("structured data must not be able to close its script element: %s", raw)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, raw)
	}
	want := map[string]any{
		"@type":          "BlogPosting",
		"headline":       post.Title,
		"url":            "https://blog.example.com/posts/hello",
		"datePublished":  "2024-03-16T10:00:00+08:00",
		"dateModified":   "2024-03-16T10:00:00+08:00", // never before the publication date
		"keywords":       "go,测试",
		"articleSection": "技术",
		"wordCount":      float64(1200),
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s = %v, want %v", key, got[key], value)
		}
	}
	if author, _ := got["author"].(map[string]any); author["name"] != "测试博客" || author["image"] != "https://blog.example.com/static/logo.png" {
		t.Errorf("author = %v", got["author"])
	}
}

func TestSiteJSONLD(t *testing.T) {
	conf := &config.AppConfig{Name: "测试博客", Host: "https://blog.example.com", Cdn: "https://cdn.example.com/blog/", Description: "一句话简介"}
	var got map[string]any
	if err := json.Unmarshal([]byte(siteJSONLD(conf)), &got); err != nil {
		t.Fatal(err)
	}
	action, _ := got["potentialAction"].(map[string]any)
	if got["@type"] != "Blog" || got["description"] != "一句话简介" || action["target"] != "https://blog.example.com/?keyword={search_term_string}" {
		t.Errorf("unexpected site data: %v", got)
	}
	if author, _ := got["author"].(map[string]any); author["image"] != "https://cdn.example.com/blog/logo.png" {
		t.Errorf("logo must come from the CDN: %v", got["author"])
	}
}
