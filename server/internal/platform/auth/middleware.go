package auth

import (
	"camopanel/server/internal/platform/errs"
	"camopanel/server/internal/platform/httpx"
	"errors"

	"github.com/gin-gonic/gin"
)

const CurrentUserKey = "current_user"

var ErrSubjectNotFound = errors.New("subject not found")

type SubjectLoader func(c *gin.Context, subjectID string) (any, error)

func RequireAuth(cookieName string, sessions *SessionManager, loadSubject SubjectLoader) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err != nil {
			httpx.ErrorFrom(c, errs.E(errs.CodeUnauthenticated, "需要先登录"))
			return
		}

		userID, err := sessions.Parse(token)
		if err != nil {
			httpx.ErrorFrom(c, errs.E(errs.CodeUnauthenticated, "登录状态已失效"))
			return
		}

		subject, err := loadSubject(c, userID)
		if err != nil {
			if errors.Is(err, ErrSubjectNotFound) {
				httpx.ErrorFrom(c, errs.E(errs.CodeUnauthenticated, "用户不存在"))
				return
			}
			httpx.ErrorFrom(c, errs.Wrap(errs.CodeInternal, "load auth subject", err))
			return
		}
		if subject == nil {
			httpx.ErrorFrom(c, errs.E(errs.CodeUnauthenticated, "用户不存在"))
			return
		}

		c.Set(CurrentUserKey, subject)
		c.Next()
	}
}

func CurrentSubject(c *gin.Context) (any, bool) {
	value, ok := c.Get(CurrentUserKey)
	return value, ok
}
