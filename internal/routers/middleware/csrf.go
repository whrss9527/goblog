package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const csrfTokenKey = "csrf_token"

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func CSRFProtect(ctx *gin.Context) {
	session := sessions.Default(ctx)

	if ctx.Request.Method == "GET" || ctx.Request.Method == "HEAD" {
		token, _ := session.Get(csrfTokenKey).(string)
		if token == "" {
			token = generateToken()
			session.Set(csrfTokenKey, token)
			session.Save()
		}
		ctx.Set(csrfTokenKey, token)
		ctx.Next()
		return
	}

	sessionToken, _ := session.Get(csrfTokenKey).(string)
	formToken := ctx.Request.FormValue("_csrf")
	if formToken == "" {
		formToken = ctx.GetHeader("X-CSRF-Token")
	}

	if sessionToken == "" || formToken != sessionToken {
		ctx.AbortWithStatus(http.StatusForbidden)
		return
	}

	token := generateToken()
	session.Set(csrfTokenKey, token)
	session.Save()
	ctx.Set(csrfTokenKey, token)
	ctx.Next()
}
