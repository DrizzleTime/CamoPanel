package httpapi

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"camopanel/backend/internal/files"
	"github.com/gin-gonic/gin"
)

func fileRoot(c *gin.Context) string {
	if root := c.Query("root"); root != "" {
		return root
	}
	return "server"
}
func (p *platform) mountFiles(api *gin.RouterGroup) {
	api.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/files") && p.files == nil {
			c.AbortWithStatusJSON(503, gin.H{"error": gin.H{"message": "文件服务初始化失败，请检查 CAMO_FILES_ROOT 和数据目录权限"}})
			return
		}
		c.Next()
	})
	api.GET("/files/roots", func(c *gin.Context) {
		roots := p.files.Roots()
		serverRoot := "/"
		for _, root := range roots {
			if root.Name == "server" {
				serverRoot = root.Path
			}
		}
		home, _ := os.UserHomeDir()
		if relative, err := filepath.Rel(serverRoot, home); err != nil || !filepath.IsLocal(relative) {
			home = serverRoot
		}
		hostname, _ := os.Hostname()
		c.JSON(200, gin.H{"roots": roots, "server_root": serverRoot, "home": home, "hostname": hostname, "read_only": c.GetString("role") == "viewer", "upload_limit": files.MaxUploadSize, "text_limit": files.MaxTextSize})
	})
	api.GET("/files", func(c *gin.Context) {
		root := fileRoot(c)
		path := c.Query("path")
		entries, err := p.files.List(root, path)
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		path = p.files.DisplayPath(root, path)
		parent := filepath.ToSlash(filepath.Dir(path))
		if parent == "." {
			parent = ""
		}
		for _, scope := range p.files.Roots() {
			if root == "server" && scope.Name == root && path == scope.Path {
				parent = path
			}
		}
		c.JSON(200, gin.H{"entries": entries, "path": path, "parent": parent, "root": root})
	})
	api.GET("/files/content", func(c *gin.Context) {
		text, err := p.files.Read(fileRoot(c), c.Query("path"))
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"content": text})
	})
	api.GET("/files/download-link", func(c *gin.Context) {
		root, path := fileRoot(c), c.Query("path")
		file, err := p.files.Open(root, path)
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		file.Close()
		ticket, err := p.downloads.issue(downloadClaim{root: root, path: path, actor: c.GetString("actor"), role: c.GetString("role")})
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"url": "/api/v1/files/download?ticket=" + ticket})
	})
	api.GET("/files/download", func(c *gin.Context) {
		root, path := fileRoot(c), c.Query("path")
		if c.Query("ticket") != "" {
			root, path = c.GetString("file_download_root"), c.GetString("file_download_path")
		}
		file, err := p.files.Open(root, path)
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filepath.Base(path)}))
		c.Header("Content-Type", "application/octet-stream")
		http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), file)
	})
	api.PUT("/files", func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, files.MaxTextSize*6+1024)
		var body struct {
			Root    string `json:"root"`
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			badRequest(c, "请填写文件路径和内容（文本最大 4 MB）")
			return
		}
		if err := p.files.Write(body.Root, body.Path, body.Content); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"message": "文件已保存"})
	})
	api.POST("/files/upload", func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, files.MaxUploadSize+(1<<20))
		if strings.HasPrefix(c.ContentType(), "multipart/") {
			if err := c.Request.ParseMultipartForm(8 << 20); err != nil {
				badRequest(c, "无法读取上传文件（单文件最大 256 MB）："+err.Error())
				return
			}
			if c.Request.MultipartForm != nil {
				defer c.Request.MultipartForm.RemoveAll()
			}
			source, header, err := c.Request.FormFile("file")
			if err != nil {
				badRequest(c, "请选择文件")
				return
			}
			defer source.Close()
			path := c.PostForm("path")
			if path == "" {
				path = header.Filename
			}
			if err := p.files.Upload(c.PostForm("root"), path, source, c.PostForm("overwrite") == "true"); err != nil {
				badRequest(c, err.Error())
				return
			}
			c.JSON(201, gin.H{"message": "文件已上传", "path": path, "size": header.Size})
			return
		}
		var body struct {
			Root    string `json:"root"`
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			badRequest(c, "请填写上传内容")
			return
		}
		if err := p.files.Write(body.Root, body.Path, body.Content); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(201, gin.H{"message": "已上传"})
	})
	api.POST("/files/create", func(c *gin.Context) {
		var body struct {
			Root      string `json:"root"`
			Path      string `json:"path"`
			Directory bool   `json:"directory"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Path == "" {
			badRequest(c, "请填写文件或目录名称")
			return
		}
		if err := p.files.Create(body.Root, body.Path, body.Directory); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(201, gin.H{"message": "已创建"})
	})
	api.POST("/files/mkdir", func(c *gin.Context) {
		var body struct {
			Root string `json:"root"`
			Path string `json:"path"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			badRequest(c, "请填写目录")
			return
		}
		if err := p.files.Mkdir(body.Root, body.Path); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(201, gin.H{"message": "目录已创建"})
	})
	api.POST("/files/rename", func(c *gin.Context) {
		var body struct {
			Root string `json:"root"`
			From string `json:"from"`
			To   string `json:"to"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			badRequest(c, "请填写原路径和新路径")
			return
		}
		if err := p.files.Rename(body.Root, body.From, body.To); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"message": "已重命名"})
	})
	api.POST("/files/transfer", func(c *gin.Context) {
		var body struct {
			Root     string   `json:"root"`
			Paths    []string `json:"paths"`
			DestRoot string   `json:"dest_root"`
			Dest     string   `json:"dest"`
			Move     bool     `json:"move"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || len(body.Paths) == 0 || len(body.Paths) > 500 {
			badRequest(c, "请选择 1–500 个文件")
			return
		}
		results := []gin.H{}
		succeeded := 0
		for _, path := range body.Paths {
			target := filepath.Join(body.Dest, filepath.Base(path))
			err := p.files.Transfer(body.Root, path, body.DestRoot, target, body.Move)
			result := gin.H{"path": path, "dest": target}
			if err != nil {
				result["error"] = err.Error()
			} else {
				succeeded++
			}
			results = append(results, result)
		}
		c.JSON(200, gin.H{"results": results, "succeeded": succeeded})
	})
	api.POST("/files/delete", func(c *gin.Context) {
		var body struct {
			Root  string   `json:"root"`
			Paths []string `json:"paths"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || len(body.Paths) == 0 || len(body.Paths) > 500 {
			badRequest(c, "请选择 1–500 个文件")
			return
		}
		results := []gin.H{}
		succeeded := 0
		for _, path := range body.Paths {
			result := gin.H{"path": path}
			if err := p.files.Delete(body.Root, path); err != nil {
				result["error"] = err.Error()
			} else {
				succeeded++
			}
			results = append(results, result)
		}
		c.JSON(200, gin.H{"results": results, "succeeded": succeeded})
	})
	api.DELETE("/files", func(c *gin.Context) {
		if err := p.files.Delete(fileRoot(c), c.Query("path")); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"message": "已删除"})
	})
	api.POST("/files/permissions", func(c *gin.Context) {
		var body struct {
			Root string `json:"root"`
			Path string `json:"path"`
			Mode string `json:"mode"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			badRequest(c, "请填写文件权限")
			return
		}
		if err := p.files.Chmod(body.Root, body.Path, body.Mode); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"message": "权限已更新"})
	})
	api.POST("/files/zip", func(c *gin.Context) {
		var body struct {
			Root  string   `json:"root"`
			Path  string   `json:"path"`
			Paths []string `json:"paths"`
			Dest  string   `json:"dest"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			badRequest(c, "请填写打包路径")
			return
		}
		if len(body.Paths) == 0 {
			body.Paths = []string{body.Path}
		}
		if len(body.Paths) > 500 {
			badRequest(c, "最多选择 500 个项目")
			return
		}
		if err := p.files.Archive(body.Root, body.Paths, body.Dest); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(201, gin.H{"message": "已压缩"})
	})
	api.POST("/files/unzip", func(c *gin.Context) {
		var body struct {
			Root string `json:"root"`
			Zip  string `json:"zip"`
			Dest string `json:"dest"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			badRequest(c, "请填写压缩包路径")
			return
		}
		if err := p.files.Unzip(body.Root, body.Zip, body.Dest); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(201, gin.H{"message": "已解压"})
	})
}
