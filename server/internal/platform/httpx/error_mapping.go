package httpx

import (
	"net/http"

	"camopanel/server/internal/platform/errs"
)

func MapError(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}

	platformErr, ok := errs.As(err)
	if !ok {
		return http.StatusInternalServerError, "internal server error"
	}

	switch platformErr.Code {
	case errs.CodeInvalidArgument:
		return http.StatusBadRequest, platformErr.ClientMessage()
	case errs.CodeUnauthenticated:
		return http.StatusUnauthorized, platformErr.ClientMessage()
	case errs.CodeNotFound:
		return http.StatusNotFound, platformErr.ClientMessage()
	case errs.CodeConflict:
		return http.StatusConflict, platformErr.ClientMessage()
	case errs.CodeUnavailable:
		return http.StatusServiceUnavailable, platformErr.ClientMessage()
	case errs.CodeInternal:
		return http.StatusInternalServerError, platformErr.ClientMessage()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
