package front

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	likeCookieName    = "likedPosts"
	likeCookieMaxAge  = 365 * 24 * 3600
	likeCookieEntries = 60
	// likeHeader must be present on like requests. Browsers only let same-origin
	// scripts set custom headers without a CORS preflight, and the API's CORS
	// policy does not allow this one, so cross-site forms cannot inflate counters.
	likeHeader      = "X-Requested-With"
	likeHeaderValue = "goblog"
)

// likedSet decodes the cookie that remembers which posts this browser liked.
func likedSet(ctx *gin.Context) []string {
	raw, err := ctx.Cookie(likeCookieName)
	if err != nil || raw == "" {
		return nil
	}
	var slugs []string
	for _, part := range strings.Split(raw, "|") {
		if slug, err := url.QueryUnescape(part); err == nil && slug != "" {
			slugs = append(slugs, slug)
		}
	}
	return slugs
}

func hasLiked(ctx *gin.Context, identity string) bool {
	for _, slug := range likedSet(ctx) {
		if slug == identity {
			return true
		}
	}
	return false
}

// Like handles POST /api/posts/:identity/like. A browser can like a post once
// (tracked by cookie); repeating the request is idempotent.
func (h *PostHandler) Like(ctx *gin.Context) {
	if ctx.GetHeader(likeHeader) != likeHeaderValue {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if site := ctx.GetHeader("Sec-Fetch-Site"); site != "" && site != "same-origin" {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	identity := ctx.Param("identity")
	post, err := h.PostRepo.GetPostByIdentity(identity)
	if err != nil || post.Status != 1 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	liked := likedSet(ctx)
	for _, slug := range liked {
		if slug == identity {
			ctx.JSON(http.StatusOK, gin.H{"likes": post.Likes, "liked": true, "already": true})
			return
		}
	}

	likes, err := h.PostRepo.IncrLike(post.Id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	liked = append(liked, identity)
	if len(liked) > likeCookieEntries {
		liked = liked[len(liked)-likeCookieEntries:]
	}
	escaped := make([]string, len(liked))
	for i, slug := range liked {
		escaped[i] = url.QueryEscape(slug)
	}
	secure := strings.HasPrefix(h.conf.App.Host, "https://")
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(likeCookieName, strings.Join(escaped, "|"), likeCookieMaxAge, "/", "", secure, true)
	ctx.JSON(http.StatusOK, gin.H{"likes": likes, "liked": true})
}
