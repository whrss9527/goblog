package front

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// maxWebhookBody caps the payload read for signature checking; GitHub's push
// events are a few KB.
const maxWebhookBody = 5 << 20

// GitWebhook handles POST /api/hooks/git. GitHub calls it after every push to
// the content repository and the server syncs right away, instead of at the
// next scheduled sync. Without app.git_webhook_secret the endpoint does not exist.
type GitWebhook struct {
	Secret string
	// RequestSync schedules a sync; it must not block.
	RequestSync func()
}

func (h *GitWebhook) Handle(ctx *gin.Context) {
	if h.Secret == "" || h.RequestSync == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, maxWebhookBody))
	if err != nil || !validSignature(h.Secret, body, ctx.GetHeader("X-Hub-Signature-256")) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}
	switch ctx.GetHeader("X-GitHub-Event") {
	case "ping":
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	case "push":
		h.RequestSync()
		ctx.JSON(http.StatusAccepted, gin.H{"sync": "scheduled"})
	default:
		ctx.Status(http.StatusNoContent)
	}
}

// validSignature checks GitHub's X-Hub-Signature-256: "sha256=" + hex HMAC of the body.
func validSignature(secret string, body []byte, header string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	got, err := hex.DecodeString(header[len(prefix):])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(got, mac.Sum(nil))
}
