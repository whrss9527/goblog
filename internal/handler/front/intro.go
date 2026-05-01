package front

import (
	"github.com/gin-gonic/gin"

	"goblog/internal/pkg/view"
)

func (handler *PostHandler) Intro(ctx *gin.Context) {
	data := make(map[string]any)
	view.IntroRender(data, ctx.Writer, handler.conf.App)
}
