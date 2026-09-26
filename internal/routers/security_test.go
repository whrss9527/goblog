package routers

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"goblog/internal/config"
)

func signIn(h http.Handler, remote, forwardedFor string) int {
	form := url.Values{"email": {"nobody@example.com"}, "password": {"wrong password"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/sign-in", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = remote
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

// The login limit (5 attempts per 15 minutes and client) used to be keyed by
// the first X-Forwarded-For entry, which the client writes itself: a fresh
// made-up address per request meant unlimited password guesses.
func TestForgedForwardedForDoesNotEscapeRateLimits(t *testing.T) {
	h := newTestServer(t)

	for i := 1; i <= 6; i++ {
		status := signIn(h, "203.0.113.9:4000", "10.0.0."+strconv.Itoa(i))
		if i <= 5 && status != http.StatusUnauthorized {
			t.Fatalf("attempt %d = %d, want 401", i, status)
		}
		if i == 6 && status != http.StatusTooManyRequests {
			t.Fatalf("a direct client that forges X-Forwarded-For is still limited, attempt 6 = %d", status)
		}
	}

	// behind cloudflared / nginx on the same machine the proxy's last entry is the client
	for i := 1; i <= 6; i++ {
		if status := signIn(h, "127.0.0.1:5000", "6.6.6.6, 198.51.100."+strconv.Itoa(i)); status != http.StatusUnauthorized {
			t.Fatalf("different clients behind the local proxy have their own limits, attempt %d = %d", i, status)
		}
	}
	for i := 1; i <= 6; i++ {
		status := signIn(h, "127.0.0.1:5000", strconv.Itoa(i)+".6.6.6, 198.51.100.77")
		if i == 6 && status != http.StatusTooManyRequests {
			t.Fatalf("one client behind the proxy cannot rotate the forged part, attempt 6 = %d", status)
		}
	}
}

func TestTrustedProxiesFromConfig(t *testing.T) {
	h := newTestServer(t, func(c *config.Config) { c.Server.TrustedProxies = []string{"10.0.0.0/8"} })
	for i := 1; i <= 6; i++ {
		if status := signIn(h, "10.1.2.3:4000", "198.51.100."+strconv.Itoa(i)); status != http.StatusUnauthorized {
			t.Fatalf("a configured proxy is believed, attempt %d = %d", i, status)
		}
	}
}

// nginx configured with only "proxy_set_header X-Real-IP $remote_addr" passes
// the visitor's own X-Forwarded-For through, and gin reads that one first.
func TestClientIPHeaderFromConfig(t *testing.T) {
	h := newTestServer(t, func(c *config.Config) { c.Server.ClientIPHeader = "X-Real-IP" })
	attempt := func(realIP, forged string) int {
		form := url.Values{"email": {"nobody@example.com"}, "password": {"wrong password"}}
		req := httptest.NewRequest(http.MethodPost, "/admin/sign-in", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.RemoteAddr = "127.0.0.1:5000"
		req.Header.Set("X-Real-IP", realIP)
		req.Header.Set("X-Forwarded-For", forged)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	for i := 1; i <= 6; i++ {
		status := attempt("198.51.100.7", "10.9.8."+strconv.Itoa(i))
		if i == 6 && status != http.StatusTooManyRequests {
			t.Fatalf("a forged X-Forwarded-For is ignored, attempt 6 = %d", status)
		}
	}
	if status := attempt("198.51.100.8", "10.9.8.1"); status != http.StatusUnauthorized {
		t.Errorf("another visitor behind the proxy has a limit of its own, got %d", status)
	}
}

func TestSecurityHeadersOnPages(t *testing.T) {
	h := newTestServer(t)
	for target, frame := range map[string]string{"/": "SAMEORIGIN", "/posts/hello": "SAMEORIGIN", "/admin/login": "DENY", "/nope": "SAMEORIGIN"} {
		rec := get(t, h, target)
		if got := rec.Header().Get("X-Frame-Options"); got != frame {
			t.Errorf("%s: X-Frame-Options = %q, want %q", target, got, frame)
		}
		if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%s: no nosniff", target)
		}
	}
}

func conditionalGet(h http.Handler, target string, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestFeedAndSitemapRevalidate(t *testing.T) {
	h := newTestServer(t)
	for _, target := range []string{"/feed.xml", "/feed", "/sitemap.xml"} {
		first := get(t, h, target)
		etag, modified := first.Header().Get("ETag"), first.Header().Get("Last-Modified")
		if first.Code != http.StatusOK || !strings.HasPrefix(etag, `W/"`) || modified == "" || first.Header().Get("Cache-Control") != "no-cache" {
			t.Fatalf("%s = %d, ETag %q, Last-Modified %q", target, first.Code, etag, modified)
		}
		if lastMod, err := http.ParseTime(modified); err != nil || time.Since(lastMod) < 24*time.Hour {
			t.Errorf("%s: Last-Modified %q must be the newest post (a month ago), not the start of the server", target, modified)
		}
		for _, header := range []map[string]string{
			{"If-None-Match": etag},
			{"If-None-Match": `"other", ` + strings.TrimPrefix(etag, "W/")},
			{"If-Modified-Since": modified},
			{"If-Modified-Since": time.Now().UTC().Format(http.TimeFormat), "Accept-Encoding": "gzip"},
		} {
			rec := conditionalGet(h, target, header)
			if rec.Code != http.StatusNotModified || rec.Body.Len() != 0 {
				t.Errorf("%s with %v = %d (%d bytes), want an empty 304", target, header, rec.Code, rec.Body.Len())
			}
		}
		for _, header := range []map[string]string{
			{"If-None-Match": `W/"stale"`},
			{"If-None-Match": `W/"stale"`, "If-Modified-Since": modified}, // If-None-Match wins
			{"If-Modified-Since": "Mon, 02 Jan 2006 15:04:05 GMT"},
		} {
			if rec := conditionalGet(h, target, header); rec.Code != http.StatusOK || rec.Body.Len() == 0 {
				t.Errorf("%s with %v = %d, want the document", target, header, rec.Code)
			}
		}
	}

	feed := get(t, h, "/feed.xml").Body.String()
	for _, want := range []string{"<subtitle>写给测试的博客</subtitle>", "<updated>" + recentDate().Format("2006-01-02"), "<summary type=\"html\">第一行摘要第二行摘要</summary>"} {
		if !strings.Contains(feed, want) {
			t.Errorf("feed does not contain %q", want)
		}
	}
	if strings.Contains(feed, "style=") || strings.Contains(feed, "Draft") {
		t.Errorf("feed entries carry no inline styles and no hidden posts")
	}

	sitemap := get(t, h, "/sitemap.xml").Body.String()
	for _, want := range []string{"<loc>https://blog.example.com/</loc>", "/pages/about</loc>", "/pages/flow</loc>", "<loc>https://blog.example.com/posts/older</loc>\n    <lastmod>2021-01-02</lastmod>"} {
		if !strings.Contains(sitemap, want) {
			t.Errorf("sitemap does not contain %q", want)
		}
	}
}

func TestUntrustedLocalProxyIsReported(t *testing.T) {
	h := newTestServer(t)
	var logs strings.Builder
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	for i := 0; i < 3; i++ {
		signIn(h, "203.0.113.9:4000", "10.0.0.1") // a stranger forging the header: not worth a word
	}
	if strings.Contains(logs.String(), "trusted_proxies") {
		t.Fatalf("forged headers from public addresses are ignored silently")
	}
	for i := 0; i < 3; i++ {
		signIn(h, "172.18.0.5:4000", "198.51.100.7") // cloudflared in a Docker network
	}
	if n := strings.Count(logs.String(), "proxy=172.18.0.5"); n != 1 {
		t.Errorf("a proxy on the local network that is not trusted is reported once, got %d times:\n%s", n, logs.String())
	}
}
