package auth_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	platformauth "camopanel/server/internal/platform/auth"

	"github.com/gin-gonic/gin"
)

func TestRequireAuth_LoaderErrorSemantics(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	manager := platformauth.NewSessionManager("test-secret")
	token, err := manager.Issue("user-1")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}

	tests := []struct {
		name       string
		loadErr    error
		wantStatus int
	}{
		{
			name:       "subject not found maps to unauthorized",
			loadErr:    platformauth.ErrSubjectNotFound,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "storage failure maps to internal error",
			loadErr:    errors.New("db is down"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := gin.New()
			router.Use(platformauth.RequireAuth("sid", manager, func(_ *gin.Context, _ string) (any, error) {
				return nil, tt.loadErr
			}))
			router.GET("/secure", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			req := httptest.NewRequest(http.MethodGet, "/secure", nil)
			req.AddCookie(&http.Cookie{Name: "sid", Value: token, Path: "/"})
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
