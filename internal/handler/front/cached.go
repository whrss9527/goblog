package front

import (
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// cachedDoc is a generated document (feed, sitemap) kept in memory and served
// with validators, so that feed readers and crawlers that poll it get a small
// "304 Not Modified" until something actually changes.
type cachedDoc struct {
	mu       sync.RWMutex
	body     []byte
	etag     string
	modified time.Time
}

// set replaces the document. modified is when its content last changed (the
// newest post, not the moment it was generated), so restarts do not make it look new.
func (d *cachedDoc) set(body []byte, modified time.Time) {
	sum := sha1.Sum(body)
	d.mu.Lock()
	defer d.mu.Unlock()
	d.body = body
	d.etag = `W/"` + hex.EncodeToString(sum[:8]) + `"` // weak: the gzip middleware re-encodes the body
	d.modified = modified.UTC().Truncate(time.Second)
}

func (d *cachedDoc) get() (body []byte, etag string, modified time.Time) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.body, d.etag, d.modified
}

// serve writes the document, or 304 when the client's copy is current. Clients
// revalidate on every use (no-cache), which is cheap thanks to the validators.
func (d *cachedDoc) serve(ctx *gin.Context, contentType string) {
	body, etag, modified := d.get()
	if body == nil {
		ctx.Status(http.StatusServiceUnavailable)
		return
	}
	header := ctx.Writer.Header()
	header.Set("ETag", etag)
	if !modified.IsZero() {
		header.Set("Last-Modified", modified.Format(http.TimeFormat))
	}
	header.Set("Cache-Control", "no-cache")
	if notModified(ctx.Request, etag, modified) {
		ctx.Status(http.StatusNotModified)
		return
	}
	ctx.Data(http.StatusOK, contentType, body)
}

// notModified evaluates If-None-Match (which wins when present) and If-Modified-Since.
func notModified(r *http.Request, etag string, modified time.Time) bool {
	if match := r.Header.Get("If-None-Match"); match != "" {
		for _, candidate := range strings.Split(match, ",") {
			candidate = strings.TrimSpace(candidate)
			if candidate == "*" || strings.TrimPrefix(candidate, "W/") == strings.TrimPrefix(etag, "W/") {
				return true
			}
		}
		return false
	}
	if since, err := http.ParseTime(r.Header.Get("If-Modified-Since")); err == nil && !modified.IsZero() {
		return !modified.After(since)
	}
	return false
}
