package admin

import (
	"crypto/subtle"
	"net/http"
	"strings"

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
	email := strings.TrimSpace(ctx.Request.FormValue("email"))
	password := ctx.Request.FormValue("password")

	if email == "" || password == "" || !h.checkCredentials(email, password) {
		// one message for every failure: no hint about which accounts exist
		data := map[string]interface{}{"msg": "邮箱或密码不正确，请重试"}
		view.AdminRenderStatus(http.StatusUnauthorized, data, ctx.Writer, "401", h.config.App)
		return
	}
	session := sessions.Default(ctx)
	session.Set("email", email)
	session.Save()
	http.Redirect(ctx.Writer, ctx.Request, "/admin", http.StatusFound)
}

// dummyHash keeps the response time of "no such account" in line with "wrong
// password": bcrypt runs either way.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("goblog-timing-equaliser"), bcrypt.DefaultCost)

// checkCredentials verifies a login. The account from the config file wins;
// users.json is only consulted when no such account is configured.
func (h *AuthHandler) checkCredentials(email, password string) bool {
	hash := dummyHash
	known := false
	if app := h.config.App; app.ConfigAdmin() {
		if subtle.ConstantTimeCompare([]byte(strings.ToLower(email)), []byte(strings.ToLower(app.AdminEmail))) == 1 {
			hash, known = []byte(app.AdminPasswordHash), true
		}
	} else if user, err := h.UserRepo.GetUserByEmail(email); err == nil {
		hash, known = []byte(user.Password), true
	}
	err := bcrypt.CompareHashAndPassword(hash, []byte(password))
	return known && err == nil
}
