package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func AuthWithSession(ctx *gin.Context) {
	session := sessions.Default(ctx)
	email := session.Get("email")
	if email == nil || email == "" {
		http.Redirect(ctx.Writer, ctx.Request, "/admin/login", http.StatusFound)
		ctx.Abort()
		return
	}
	ctx.Next()
}
