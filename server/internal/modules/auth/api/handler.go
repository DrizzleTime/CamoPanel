package api

import (
	"context"
	"errors"
	"net/http"

	"camopanel/server/internal/modules/auth/domain"
	"camopanel/server/internal/modules/auth/usecase"
	platformauth "camopanel/server/internal/platform/auth"
	"camopanel/server/internal/platform/errs"
	"camopanel/server/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

type LoginUsecase interface {
	Execute(ctx context.Context, input usecase.LoginInput) (usecase.LoginOutput, error)
}

type MeUsecase interface {
	Execute(ctx context.Context, userID string) (domain.User, error)
}

type Handler struct {
	cookieName string
	sessions   *platformauth.SessionManager
	login      LoginUsecase
	me         MeUsecase
}

func NewHandler(cookieName string, sessions *platformauth.SessionManager, login LoginUsecase, me MeUsecase) *Handler {
	return &Handler{
		cookieName: cookieName,
		sessions:   sessions,
		login:      login,
		me:         me,
	}
}

func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}

	output, err := h.login.Execute(c.Request.Context(), usecase.LoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidCredentials) {
			httpx.ErrorFrom(c, errs.E(errs.CodeUnauthenticated, "用户名或密码错误"))
			return
		}
		httpx.ErrorFrom(c, errs.E(errs.CodeInternal, "签发会话失败"))
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.cookieName, output.Token, 7*24*3600, "/", "", false, true)
	httpx.OK(c, gin.H{"user": output.User})
}

func (h *Handler) Logout(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.cookieName, "", -1, "/", "", false, true)
	httpx.OK(c, gin.H{"ok": true})
}

func (h *Handler) Me(c *gin.Context) {
	value, ok := platformauth.CurrentSubject(c)
	if !ok {
		httpx.ErrorFrom(c, errs.E(errs.CodeUnauthenticated, "需要先登录"))
		return
	}

	user, ok := value.(domain.User)
	if !ok {
		httpx.ErrorFrom(c, errs.E(errs.CodeInternal, "读取当前用户失败"))
		return
	}
	httpx.OK(c, gin.H{"user": sanitizeUser(user)})
}

func (h *Handler) loadSubject(c *gin.Context, userID string) (any, error) {
	user, err := h.me.Execute(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, platformauth.ErrSubjectNotFound
		}
		return nil, err
	}
	return user, nil
}

func sanitizeUser(user domain.User) gin.H {
	return gin.H{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	}
}
