package gin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHeadAsGet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	var sawMethod, sawMarker string
	router.GET("/page", func(c *gin.Context) {
		sawMethod, sawMarker = c.Request.Method, c.GetHeader(OriginalMethodHeader)
		c.String(http.StatusOK, "body")
	})
	server := httptest.NewServer(HeadAsGet(router))
	defer server.Close()

	res, err := http.Head(server.URL + "/page")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK || sawMethod != http.MethodGet || sawMarker != http.MethodHead {
		t.Errorf("HEAD must reach the GET route marked as HEAD: status %d, method %q, marker %q", res.StatusCode, sawMethod, sawMarker)
	}
	if res.ContentLength > 0 && res.Header.Get("Content-Length") == "" {
		t.Errorf("unexpected body for HEAD")
	}

	// a client cannot fake the marker to dodge visit counting… or to fake it
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/page", nil)
	req.Header.Set(OriginalMethodHeader, "HEAD")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if sawMethod != http.MethodGet || sawMarker != "" {
		t.Errorf("a marker sent by the client must be dropped, got %q", sawMarker)
	}

	if res, err := http.Head(server.URL + "/missing"); err != nil || res.StatusCode != http.StatusNotFound {
		t.Errorf("unknown paths stay 404 for HEAD")
	}
}
