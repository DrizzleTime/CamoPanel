package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	authdomain "camopanel/server/internal/modules/auth/domain"
	filesusecase "camopanel/server/internal/modules/files/usecase"
	platformauth "camopanel/server/internal/platform/auth"
	"camopanel/server/internal/platform/errs"
	"camopanel/server/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *filesusecase.Service
}

func NewHandler(service *filesusecase.Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	item, err := h.service.List(c.Query("path"))
	if err != nil {
		httpx.ErrorFrom(c, mapFileError(err))
		return
	}
	httpx.OK(c, item)
}

func (h *Handler) Read(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	item, err := h.service.Read(c.Query("path"))
	if err != nil {
		httpx.ErrorFrom(c, mapFileError(err))
		return
	}
	httpx.OK(c, item)
}

func (h *Handler) Write(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	path, err := h.service.Write(req.Path, req.Content)
	if err != nil {
		httpx.ErrorFrom(c, mapFileError(err))
		return
	}
	httpx.OK(c, gin.H{"path": path})
}

func (h *Handler) Create(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	path, err := h.service.Create(req.Path, req.Content)
	if err != nil {
		httpx.ErrorFrom(c, mapFileError(err))
		return
	}
	httpx.OK(c, gin.H{"path": path})
}

func (h *Handler) Mkdir(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	path, err := h.service.Mkdir(req.Path)
	if err != nil {
		httpx.ErrorFrom(c, mapFileError(err))
		return
	}
	httpx.OK(c, gin.H{"path": path})
}

func (h *Handler) Move(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var req struct {
		FromPath string `json:"from_path"`
		ToPath   string `json:"to_path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	path, err := h.service.Move(req.FromPath, req.ToPath)
	if err != nil {
		httpx.ErrorFrom(c, mapFileError(err))
		return
	}
	httpx.OK(c, gin.H{"path": path})
}

func (h *Handler) Delete(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	path, err := h.service.Delete(req.Path)
	if err != nil {
		httpx.ErrorFrom(c, mapFileError(err))
		return
	}
	httpx.OK(c, gin.H{"path": path})
}

func (h *Handler) Upload(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "上传数据格式错误"))
		return
	}
	files := append(form.File["files"], form.File["file"]...)
	if len(files) == 0 {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请选择要上传的文件"))
		return
	}
	items, err := h.service.Upload(c.PostForm("path"), files)
	if err != nil {
		httpx.ErrorFrom(c, mapFileError(err))
		return
	}
	httpx.OK(c, gin.H{"items": items})
}

func (h *Handler) Download(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	item, err := h.service.Download(c.Query("path"))
	if err != nil {
		httpx.ErrorFrom(c, mapFileError(err))
		return
	}
	defer item.File.Close()

	c.Header("Content-Type", item.ContentType)
	c.Header("Content-Length", fmt.Sprintf("%d", item.Size))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", item.Name))
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, item.File)
}

func requireSuperAdmin(c *gin.Context) bool {
	value, ok := platformauth.CurrentSubject(c)
	if !ok {
		httpx.ErrorFrom(c, errs.E(errs.CodeUnauthenticated, "需要先登录"))
		return false
	}
	user, ok := value.(authdomain.User)
	if !ok || user.Role != authdomain.RoleSuperAdmin {
		httpx.Error(c, http.StatusForbidden, "没有权限")
		return false
	}
	return true
}

func mapFileError(err error) error {
	switch {
	case errors.Is(err, os.ErrNotExist):
		return errs.E(errs.CodeNotFound, "路径不存在")
	default:
		return errs.E(errs.CodeInvalidArgument, err.Error())
	}
}
