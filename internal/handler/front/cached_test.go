package front

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCachedDocLastModifiedFollowsTheContent(t *testing.T) {
	var d cachedDoc
	newest := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	d.set([]byte("feed with posts A, B"), newest)
	_, first, modified := d.get()
	assert.True(t, modified.Equal(newest), "dated by the newest post, got %s", modified)

	d.set([]byte("feed with posts A, B"), newest)
	_, again, unchanged := d.get()
	assert.Equal(t, first, again)
	assert.True(t, unchanged.Equal(newest), "regenerating the same document keeps its date")

	// B, the newest post, is deleted: the document changed but its newest post is older
	d.set([]byte("feed with post A"), newest.Add(-24*time.Hour))
	_, etag, modified := d.get()
	assert.NotEqual(t, first, etag)
	assert.True(t, modified.After(newest), "a change without a newer post is dated now, got %s", modified)
	req := httptest.NewRequest(http.MethodGet, "/feed.xml", nil)
	req.Header.Set("If-Modified-Since", newest.Format(http.TimeFormat))
	assert.False(t, notModified(req, etag, modified), "a copy that still lists B is not current")

	// dates never go back: a client holding the copy dated now must see the next change
	d.set([]byte("feed with posts A, C"), newest.Add(48*time.Hour))
	_, _, next := d.get()
	assert.False(t, next.Before(modified), "got %s after %s", next, modified)

	later := next.Add(time.Hour)
	d.set([]byte("feed with posts A, C, D"), later)
	_, _, modified = d.get()
	assert.True(t, modified.Equal(later), "a post newer than the last change dates it, got %s", modified)
}
