package httpx_test

import (
	"errors"
	"net/http"
	"testing"

	"camopanel/server/internal/platform/errs"
	"camopanel/server/internal/platform/httpx"
)

func TestMapError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "invalid argument",
			err:        errs.E(errs.CodeInvalidArgument, "请求参数错误"),
			wantStatus: http.StatusBadRequest,
			wantBody:   "请求参数错误",
		},
		{
			name:       "unauthenticated",
			err:        errs.E(errs.CodeUnauthenticated, "需要先登录"),
			wantStatus: http.StatusUnauthorized,
			wantBody:   "需要先登录",
		},
		{
			name:       "not found",
			err:        errs.E(errs.CodeNotFound, "资源不存在"),
			wantStatus: http.StatusNotFound,
			wantBody:   "资源不存在",
		},
		{
			name:       "internal from domain error",
			err:        errs.E(errs.CodeInternal, "内部错误"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal server error",
		},
		{
			name:       "conflict",
			err:        errs.E(errs.CodeConflict, "资源冲突"),
			wantStatus: http.StatusConflict,
			wantBody:   "资源冲突",
		},
		{
			name:       "unavailable",
			err:        errs.E(errs.CodeUnavailable, "服务暂不可用"),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "服务暂不可用",
		},
		{
			name:       "wrapped internal should stay masked",
			err:        errs.Wrap(errs.CodeInternal, "database panic: x", errors.New("dial tcp 10.0.0.1: connect: refused")),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal server error",
		},
		{
			name:       "unknown error fallback",
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal server error",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status, body := httpx.MapError(tt.err)
			if status != tt.wantStatus {
				t.Fatalf("status = %d, want %d", status, tt.wantStatus)
			}
			if body != tt.wantBody {
				t.Fatalf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}
