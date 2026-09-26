package routers

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"goblog/internal/config"
)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	img.Set(1, 1, color.RGBA{R: 32, G: 148, B: 96, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// upload posts a file the way the editor does: multipart, token in X-CSRF-Token.
func (a *adminClient) upload(token, name string, data []byte) (int, map[string]any) {
	a.t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, _ := w.CreateFormFile("file", name)
	part.Write(data)
	w.Close()
	req, _ := http.NewRequest(http.MethodPost, a.base+"/admin/uploads", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set("X-CSRF-Token", token)
	}
	res, err := a.client.Do(req)
	if err != nil {
		a.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	var out map[string]any
	json.Unmarshal(raw, &out)
	return res.StatusCode, out
}

var uploadedName = regexp.MustCompile(`^/images/\d{4}/\d{2}/\d{13}(-\d+)?\.png$`)

func TestUploadImages(t *testing.T) {
	a := newAdminClient(t, func(c *config.Config) { c.App.Upload = &config.UploadConfig{MaxMB: 1} })
	token := a.token("/admin/posts/add")

	status, out := a.upload(token, "截图.png", pngBytes(t))
	address, _ := out["url"].(string)
	if status != http.StatusOK || !uploadedName.MatchString(address) {
		t.Fatalf("upload = %d %v", status, out)
	}
	next, _ := out["csrf"].(string)
	if next == "" || next == token {
		t.Fatalf("the answer carries the next CSRF token")
	}
	stored, err := os.ReadFile(filepath.Join(a.dataDir, filepath.FromSlash(strings.TrimPrefix(address, "/"))))
	if err != nil || !bytes.Equal(stored, pngBytes(t)) {
		t.Fatalf("the image is stored in the content repository: %v", err)
	}

	res, err := a.client.Get(a.base + address)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/png" ||
		res.Header.Get("Cache-Control") != "public, max-age=31536000, immutable" || res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("GET %s = %d %v", address, res.StatusCode, res.Header)
	}
	if res, _ := a.client.Get(a.base + "/images/2026/09/0000000000000.png"); res.StatusCode != http.StatusNotFound || res.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("a missing image is a 404 that is not cached, got %d %q", res.StatusCode, res.Header.Get("Cache-Control"))
	}

	// the old token is spent, the new one works — for the next upload and for saving the post
	if status, _ := a.upload(token, "again.png", pngBytes(t)); status != http.StatusForbidden {
		t.Errorf("a spent token = %d, want 403", status)
	}
	token = a.token("/admin/posts/add")
	for _, tt := range []struct {
		name   string
		data   []byte
		status int
		msg    string
	}{
		{"notes.txt", []byte("just text"), http.StatusUnsupportedMediaType, "只能上传"},
		{"drawing.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), http.StatusUnsupportedMediaType, "只能上传"},
		{"page.png", []byte("<html><script>alert(1)</script></html>"), http.StatusUnsupportedMediaType, "只能上传"},
		{"huge.png", append(pngBytes(t), make([]byte, 1<<20)...), http.StatusRequestEntityTooLarge, "最大 1 MB"},
	} {
		status, out := a.upload(token, tt.name, tt.data)
		if status != tt.status || !strings.Contains(out["error"].(string), tt.msg) {
			t.Errorf("%s = %d %v", tt.name, status, out)
		}
		token, _ = out["csrf"].(string) // rejected uploads still hand over the next token
	}
	if status, _ := a.upload("", "no-token.png", pngBytes(t)); status != http.StatusForbidden {
		t.Errorf("upload without token = %d, want 403", status)
	}

	// several uploads in one go get names of their own
	names := map[string]bool{}
	for i := 0; i < 3; i++ {
		_, out := a.upload(a.token("/admin/posts/add"), "burst.png", pngBytes(t))
		names[out["url"].(string)] = true
	}
	if len(names) != 3 {
		t.Errorf("uploads within the same millisecond must not share a name: %v", names)
	}
}

func TestUploadNeedsLogin(t *testing.T) {
	h := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/admin/uploads", strings.NewReader("x"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/admin/login" {
		t.Errorf("anonymous upload = %d", rec.Code)
	}
}

func TestUploadToObjectStorage(t *testing.T) {
	var mu sync.Mutex
	var puts []string
	bucket := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		puts = append(puts, r.Method+" "+r.URL.Path+" "+r.Header.Get("Content-Type")+" "+strings.Split(r.Header.Get("Authorization"), ",")[0]+" "+
			strings.Repeat("x", len(body)))
		mu.Unlock()
	}))
	defer bucket.Close()

	a := newAdminClient(t, func(c *config.Config) {
		c.App.Upload = &config.UploadConfig{S3Endpoint: bucket.URL, S3Bucket: "pics", S3AccessKey: "AK", S3SecretKey: "SK",
			PublicURL: "https://pic.example.com/", Prefix: "/blog/"}
	})
	status, out := a.upload(a.token("/admin/posts/add"), "shot.png", pngBytes(t))
	address, _ := out["url"].(string)
	if status != http.StatusOK || !regexp.MustCompile(`^https://pic\.example\.com/blog/\d{4}/\d{2}/\d{13}\.png$`).MatchString(address) {
		t.Fatalf("upload = %d %v", status, out)
	}
	key := strings.TrimPrefix(address, "https://pic.example.com/")
	mu.Lock()
	defer mu.Unlock()
	if len(puts) != 1 || !strings.HasPrefix(puts[0], "PUT /pics/"+key+" image/png AWS4-HMAC-SHA256 Credential=AK/") {
		t.Errorf("object storage got %v", puts)
	}
	if _, err := os.Stat(filepath.Join(a.dataDir, "images")); !os.IsNotExist(err) {
		t.Errorf("nothing is written to the content repository")
	}
}
