package admin

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"goblog/internal/config"
	"goblog/internal/pkg/view"
	"goblog/internal/repository"
)

type AuthHandler struct {
	UserRepo repository.UserRepository
	config   *config.Config
}

func NewAuthHandler(userRepo repository.UserRepository, config *config.Config) *AuthHandler {
	return &AuthHandler{
		UserRepo: userRepo,
		config:   config,
	}
}

func (h *AuthHandler) Login(ctx *gin.Context) {
	data := make(map[string]interface{})
	view.AdminRender(data, ctx.Writer, "login", h.config.App)
}

func (h *AuthHandler) Register(ctx *gin.Context) {
	data := make(map[string]interface{})
	view.AdminRender(data, ctx.Writer, "register", h.config.App)
}

func (h *AuthHandler) Logout(ctx *gin.Context) {
	session := sessions.Default(ctx)
	session.Clear()
	session.Save()
	http.Redirect(ctx.Writer, ctx.Request, "/admin/login", http.StatusFound)
}

func (h *AuthHandler) Signup(ctx *gin.Context) {
	email := ctx.Request.FormValue("email")
	password := ctx.Request.FormValue("password")
	repassword := ctx.Request.FormValue("repassword")
	if email == "" || password == "" || repassword == "" || password != repassword {
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	// 不允许注册
}

func (h *AuthHandler) Signin(ctx *gin.Context) {
	email := ctx.Request.FormValue("email")
	password := ctx.Request.FormValue("password")

	if email == "" || password == "" {
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	user, err := h.UserRepo.GetUserByEmail(email)
	if err != nil {
		data := make(map[string]interface{})
		data["msg"] = "用户不存在，请重试"
		view.AdminRender(data, ctx.Writer, "401", h.config.App)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		data := make(map[string]interface{})
		data["msg"] = "密码不正确，请重试"
		view.AdminRender(data, ctx.Writer, "401", h.config.App)
		return
	}
	session := sessions.Default(ctx)
	session.Set("email", email)
	session.Save()
	http.Redirect(ctx.Writer, ctx.Request, "/admin", http.StatusFound)
}
