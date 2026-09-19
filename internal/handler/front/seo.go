package front

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"goblog/internal/config"
	"goblog/internal/pkg/model"
	"goblog/internal/pkg/view"
)

var (
	fencedCode    = regexp.MustCompile("(?s)```.*?```|~~~.*?~~~")
	markdownImage = regexp.MustCompile(`!\[[^\]]*\]\(\s*<?([^)\s>]+)`)
	htmlImage     = regexp.MustCompile(`(?i)<img\b[^>]*?\bsrc\s*=\s*["']([^"']+)["']`)
)

// firstImage returns the address of the first image of a Markdown document,
// or "" when it has none. Images inside code blocks do not count.
func firstImage(markdown string) string {
	text := fencedCode.ReplaceAllString(markdown, "")
	md := markdownImage.FindStringSubmatchIndex(text)
	raw := htmlImage.FindStringSubmatchIndex(text)
	switch {
	case md != nil && (raw == nil || md[0] < raw[0]):
		return text[md[2]:md[3]]
	case raw != nil:
		return text[raw[2]:raw[3]]
	}
	return ""
}

// listCanonical is the canonical address of a post list: filters and page
// numbers are part of it, page 1 and empty filters are not.
func listCanonical(host, categoryId, tagId string, page int) string {
	q := url.Values{}
	if categoryId != "" {
		q.Set("category_id", categoryId)
	}
	if tagId != "" {
		q.Set("tag_id", tagId)
	}
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	base := strings.TrimRight(host, "/") + "/"
	if len(q) == 0 {
		return base
	}
	return base + "?" + q.Encode()
}

// jsonLD serialises structured data for a <script type="application/ld+json">
// block. encoding/json escapes <, > and &, so the result cannot close the
// script element.
func jsonLD(v any) template.JS {
	raw, err := json.Marshal(v)
	if err != nil {
		slog.Error("marshal json-ld failed", "err", err)
		return ""
	}
	return template.JS(raw)
}

func publisherLD(conf *config.AppConfig) map[string]any {
	publisher := map[string]any{"@type": "Person", "name": conf.Name, "url": strings.TrimRight(conf.Host, "/") + "/"}
	if logo := view.SiteLogo(conf.Host, conf.Cdn); logo != "" {
		publisher["image"] = logo
	}
	return publisher
}

// siteJSONLD describes the blog itself (home page), including the search box
// search engines may show for the site.
func siteJSONLD(conf *config.AppConfig) template.JS {
	home := strings.TrimRight(conf.Host, "/") + "/"
	site := map[string]any{
		"@context":   "https://schema.org",
		"@type":      "Blog",
		"name":       conf.Name,
		"url":        home,
		"inLanguage": "zh-CN",
		"author":     publisherLD(conf),
		"potentialAction": map[string]any{
			"@type":       "SearchAction",
			"target":      home + "?keyword={search_term_string}",
			"query-input": "required name=search_term_string",
		},
	}
	if conf.Description != "" {
		site["description"] = conf.Description
	}
	return jsonLD(site)
}

// postJSONLD describes an article as schema.org BlogPosting.
func postJSONLD(conf *config.AppConfig, post model.Post, description, image string, tags []model.Tag) template.JS {
	address := strings.TrimRight(conf.Host, "/") + "/posts/" + post.Identity
	article := map[string]any{
		"@context":         "https://schema.org",
		"@type":            "BlogPosting",
		"headline":         post.Title,
		"description":      description,
		"url":              address,
		"mainEntityOfPage": map[string]any{"@type": "WebPage", "@id": address},
		"datePublished":    post.CreatedAt.Format(time.RFC3339),
		"dateModified":     post.UpdatedAt.Format(time.RFC3339),
		"inLanguage":       "zh-CN",
		"author":           publisherLD(conf),
		"publisher":        publisherLD(conf),
	}
	if post.UpdatedAt.Before(post.CreatedAt) {
		article["dateModified"] = article["datePublished"]
	}
	if image != "" {
		article["image"] = []string{image}
	}
	if post.WordCount > 0 {
		article["wordCount"] = post.WordCount
	}
	if post.CategoryName != "" {
		article["articleSection"] = post.CategoryName
	}
	if len(tags) > 0 {
		names := make([]string, 0, len(tags))
		for _, tag := range tags {
			names = append(names, tag.Name)
		}
		article["keywords"] = strings.Join(names, ",")
	}
	return jsonLD(article)
}
