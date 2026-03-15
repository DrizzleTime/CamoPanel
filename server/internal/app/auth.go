package app

import (
	"fmt"
	"net/http"

	"camopanel/server/internal/config"
	"camopanel/server/internal/model"
	"camopanel/server/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func seedAdmin(db *gorm.DB, cfg config.Config, auth *services.AuthService) error {
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	hash, err := services.HashPassword(cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	user := model.User{
		ID:           uuid.NewString(),
		Username:     cfg.AdminUsername,
		PasswordHash: hash,
		Role:         model.RoleSuperAdmin,
	}
	if err := db.Create(&user).Error; err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	_ = auth
	return nil
}

func (a *App) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(a.cfg.CookieName)
		if err != nil {
			writeError(c, http.StatusUnauthorized, "需要先登录")
			return
		}

		userID, err := a.auth.ParseSession(token)
		if err != nil {
			writeError(c, http.StatusUnauthorized, "登录状态已失效")
			return
		}

		var user model.User
		if err := a.db.First(&user, "id = ?", userID).Error; err != nil {
			writeError(c, http.StatusUnauthorized, "用户不存在")
			return
		}

		c.Set("current_user", user)
		c.Next()
	}
}

func (a *App) handleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	var user model.User
	if err := a.db.First(&user, "username = ?", req.Username).Error; err != nil {
		_ = a.recordAudit("", "login_failed", "user", req.Username, map[string]any{"reason": "user_not_found"})
		writeError(c, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	if !services.CheckPassword(user.PasswordHash, req.Password) {
		_ = a.recordAudit(user.ID, "login_failed", "user", user.ID, map[string]any{"reason": "invalid_password"})
		writeError(c, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	token, err := a.auth.IssueSession(user)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "签发会话失败")
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(a.cfg.CookieName, token, 7*24*3600, "/", "", false, true)
	_ = a.recordAudit(user.ID, "login_success", "user", user.ID, nil)
	c.JSON(http.StatusOK, gin.H{"user": sanitizeUser(user)})
}

func (a *App) handleLogout(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(a.cfg.CookieName, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *App) handleMe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"user": sanitizeUser(currentUser(c))})
}

func sanitizeUser(user model.User) gin.H {
	return gin.H{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	}
}

func currentUser(c *gin.Context) model.User {
	value, _ := c.Get("current_user")
	user, _ := value.(model.User)
	return user
}
