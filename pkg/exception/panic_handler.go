package exception

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func ErrHandle(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			apiErr, isApiErr := r.(*ApiError)
			if isApiErr {
				slog.Error("PanicHandler handled apiError", "err", apiErr.Error())
				c.JSON(GetResultHttpCode(apiErr.Code), apiErr)
			} else {
				var msg string
				switch v := r.(type) {
				case error:
					msg = v.Error()
				case string:
					msg = v
				default:
					msg = fmt.Sprintf("%v", v)
				}
				slog.Error("PanicHandler handled ordinaryError", "err", msg, "stack", string(debug.Stack()))
				c.JSON(http.StatusInternalServerError, NewApiError(InternalServerErr))
			}
			c.Abort()
		}
	}()
	c.Next()
}
