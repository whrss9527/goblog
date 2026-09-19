package main

import (
	"os"
	"strings"
	"testing"
)

// The shipped robots.txt once told every crawler to stay away from the whole
// site ("Disallow: /") while the same release added a sitemap and SEO tags.
func TestRobotsTxtDoesNotBlockTheSite(t *testing.T) {
	raw, err := os.ReadFile("robots.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.EqualFold(strings.Join(strings.Fields(line), ""), "disallow:/") {
			t.Fatalf("robots.txt blocks the whole site: %q", line)
		}
	}
	if !strings.Contains(string(raw), "Disallow: /admin/") {
		t.Errorf("the admin should stay out of search engines")
	}
}
