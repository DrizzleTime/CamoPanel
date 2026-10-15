package files

import (
	"archive/zip"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const MaxTextSize int64 = 4 << 20
const MaxUploadSize int64 = 256 << 20
const MaxExtractSize uint64 = 1 << 30
const MaxExtractFiles = 10000

type Root struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Path  string `json:"path"`
}

type Manager struct {
	roots map[string]string
	mu    sync.Mutex
}

// New keeps the existing application roots. Server access is enabled explicitly
// by NewWithServerRoot so existing consumers retain their directory boundaries.
func New(dataDir string) (*Manager, error) {
	roots := map[string]string{"sites": filepath.Join(dataDir, "sites"), "apps": filepath.Join(dataDir, "apps"), "backups": filepath.Join(dataDir, "backups")}
	for name, path := range roots {
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, err
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		roots[name], err = filepath.EvalSymlinks(abs)
		if err != nil {
			return nil, err
		}
	}
	return &Manager{roots: roots}, nil
}

func NewWithServerRoot(dataDir, serverRoot string) (*Manager, error) {
	m, err := New(dataDir)
	if err != nil {
		return nil, err
	}
	if serverRoot == "" {
		serverRoot = string(os.PathSeparator)
	}
	path, err := filepath.Abs(serverRoot)
	if err != nil {
		return nil, err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("服务器文件根必须是目录")
	}
	m.roots["server"] = path
	return m, nil
}

func (m *Manager) Roots() []Root {
	labels := map[string]string{"server": "服务器文件", "sites": "网站目录", "apps": "应用数据", "backups": "备份目录"}
	roots := []Root{}
	for _, name := range []string{"server", "sites", "apps", "backups"} {
		if path, ok := m.roots[name]; ok {
			roots = append(roots, Root{Name: name, Label: labels[name], Path: path})
		}
	}
	return roots
}

func within(base, path string) bool {
	return path == base || strings.HasPrefix(path, base+string(os.PathSeparator)) || base == string(os.PathSeparator)
}
func isPrivateKey(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	return name == "privkey.pem" || strings.HasSuffix(name, ".key")
}

// cleanName accepts actual absolute server paths and retains relative paths for
// the three application scopes. No operation can escape its configured root.
func (m *Manager) cleanName(root, rel string) (string, error) {
	base, ok := m.roots[root]
	if !ok {
		return "", fmt.Errorf("未知的文件根")
	}
	if strings.ContainsRune(rel, 0) {
		return "", fmt.Errorf("路径包含无效字符")
	}
	if rel == "" {
		rel = "."
	}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part == ".." {
			return "", fmt.Errorf("路径不允许")
		}
	}
	if filepath.IsAbs(rel) {
		if root != "server" {
			return "", fmt.Errorf("请使用相对路径")
		}
		full := filepath.Clean(rel)
		if !within(base, full) {
			return "", fmt.Errorf("路径超出服务器文件根")
		}
		var err error
		rel, err = filepath.Rel(base, full)
		if err != nil {
			return "", err
		}
	}
	rel = filepath.Clean(rel)
	full := filepath.Join(base, rel)
	if !within(base, full) {
		return "", fmt.Errorf("路径不允许")
	}
	if root != "server" && isPrivateKey(full) {
		return "", fmt.Errorf("证书私钥不可访问")
	}
	return rel, nil
}

// Canonicalize existing ancestors to support normal absolute Linux symlinks
// (e.g. /var/run). os.Root still checks every actual filesystem operation.
func (m *Manager) scoped(root, rel string, followFinal bool) (*os.Root, string, error) {
	name, err := m.cleanName(root, rel)
	if err != nil {
		return nil, "", err
	}
	base := m.roots[root]
	full := filepath.Join(base, name)
	tail := []string{}
	probe := full
	if !followFinal && name != "." {
		tail = append(tail, filepath.Base(probe))
		probe = filepath.Dir(probe)
	}
	for {
		_, statErr := os.Lstat(probe)
		if statErr == nil {
			break
		}
		if !os.IsNotExist(statErr) {
			return nil, "", statErr
		}
		if probe == base {
			return nil, "", statErr
		}
		tail = append(tail, filepath.Base(probe))
		probe = filepath.Dir(probe)
	}
	canonical, err := filepath.EvalSymlinks(probe)
	if err != nil {
		return nil, "", err
	}
	for i := len(tail) - 1; i >= 0; i-- {
		canonical = filepath.Join(canonical, tail[i])
	}
	if !within(base, canonical) {
		return nil, "", fmt.Errorf("符号链接超出文件根")
	}
	if root != "server" && isPrivateKey(canonical) {
		return nil, "", fmt.Errorf("证书私钥不可访问")
	}
	name, err = filepath.Rel(base, canonical)
	if err != nil {
		return nil, "", err
	}
	handle, err := os.OpenRoot(base)
	return handle, name, err
}

func (m *Manager) Resolve(root, rel string) (string, error) {
	r, name, err := m.scoped(root, rel, true)
	if err != nil {
		return "", err
	}
	defer r.Close()
	return filepath.Join(m.roots[root], name), nil
}
func (m *Manager) DisplayPath(root, rel string) string {
	name, err := m.cleanName(root, rel)
	if err != nil {
		return rel
	}
	if root == "server" {
		return filepath.ToSlash(filepath.Join(m.roots[root], name))
	}
	if name == "." {
		return ""
	}
	return filepath.ToSlash(name)
}

type Entry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Dir         bool   `json:"dir"`
	Size        int64  `json:"size"`
	Modified    string `json:"modified"`
	Mode        string `json:"mode"`
	Permissions string `json:"permissions"`
	Owner       string `json:"owner"`
	Group       string `json:"group"`
	Type        string `json:"type"`
	Symlink     bool   `json:"symlink"`
	LinkTarget  string `json:"link_target,omitempty"`
}

func (m *Manager) List(root, rel string) ([]Entry, error) {
	r, name, err := m.scoped(root, rel, true)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	dir, err := r.Open(name)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	info, err := dir.Stat()
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("不是目录")
	}
	entries, err := dir.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	result := []Entry{}
	for _, entry := range entries {
		if root != "server" && isPrivateKey(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		itemPath := filepath.Join(rel, entry.Name())
		owner, group := fileOwner(info)
		item := Entry{Name: entry.Name(), Path: m.DisplayPath(root, itemPath), Dir: info.IsDir(), Size: info.Size(), Modified: info.ModTime().UTC().Format(time.RFC3339Nano), Mode: info.Mode().String(), Permissions: fmt.Sprintf("%04o", info.Mode().Perm()), Owner: owner, Group: group, Type: "file", Symlink: info.Mode()&os.ModeSymlink != 0}
		if item.Symlink {
			item.Type = "symlink"
			item.LinkTarget, _ = r.Readlink(filepath.Join(name, entry.Name()))
			if linked, err := m.Resolve(root, itemPath); err == nil {
				if target, err := os.Stat(linked); err == nil {
					item.Dir = target.IsDir()
				}
			}
		} else if item.Dir {
			item.Type = "directory"
		} else if !info.Mode().IsRegular() {
			item.Type = "special"
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Dir != result[j].Dir {
			return result[i].Dir
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result, nil
}

func (m *Manager) Open(root, rel string) (*os.File, error) {
	r, name, err := m.scoped(root, rel, true)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	info, err := r.Stat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("只支持普通文件")
	}
	file, err := r.Open(name)
	if err != nil {
		return nil, err
	}
	info, err = file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() {
		file.Close()
		return nil, fmt.Errorf("只支持普通文件")
	}
	return file, nil
}
func (m *Manager) Read(root, rel string) (string, error) {
	file, err := m.Open(root, rel)
	if err != nil {
		return "", err
	}
	defer file.Close()
	b, err := io.ReadAll(io.LimitReader(file, MaxTextSize+1))
	if err != nil {
		return "", err
	}
	if int64(len(b)) > MaxTextSize {
		return "", fmt.Errorf("文件超过 4 MB，请下载后编辑")
	}
	if !utf8.Valid(b) || strings.ContainsRune(string(b), 0) {
		return "", fmt.Errorf("此文件不是 UTF-8 文本，请使用下载功能")
	}
	return string(b), nil
}

func randomName() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return ".camopanel-" + hex.EncodeToString(b)
}
func (m *Manager) writeStream(root, rel string, source io.Reader, limit int64, overwrite bool) error {
	r, name, err := m.scoped(root, rel, true)
	if err != nil {
		return err
	}
	defer r.Close()
	if name == "." {
		return fmt.Errorf("请指定文件名称")
	}
	mode := fs.FileMode(0644)
	var existing fs.FileInfo
	if info, err := r.Lstat(name); err == nil {
		if !overwrite {
			return fmt.Errorf("文件已存在，请重命名或选择覆盖")
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("目标不是普通文件")
		}
		mode = info.Mode().Perm()
		existing = info
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := r.MkdirAll(filepath.Dir(name), 0755); err != nil {
		return err
	}
	tmp := filepath.Join(filepath.Dir(name), randomName())
	output, err := r.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer r.Remove(tmp)
	n, copyErr := io.Copy(output, io.LimitReader(source, limit+1))
	if copyErr == nil && n > limit {
		copyErr = fmt.Errorf("文件大小超过 %d MB 限制", limit>>20)
	}
	if copyErr == nil && existing != nil {
		copyErr = preserveOwnership(output, existing)
	}
	if copyErr == nil && existing != nil {
		copyErr = output.Chmod(mode)
	}
	if copyErr == nil {
		copyErr = output.Sync()
	}
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	// Link implements an atomic no-clobber commit for new uploads and files.
	if !overwrite {
		if err := r.Link(tmp, name); err != nil {
			return err
		}
		return nil
	}
	return r.Rename(tmp, name)
}
func (m *Manager) Write(root, rel, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writeStream(root, rel, strings.NewReader(content), MaxTextSize, true)
}
func (m *Manager) Upload(root, rel string, reader io.Reader, overwrite bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writeStream(root, rel, reader, MaxUploadSize, overwrite)
}
func (m *Manager) Create(root, rel string, directory bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !directory {
		return m.writeStream(root, rel, strings.NewReader(""), MaxTextSize, false)
	}
	r, name, err := m.scoped(root, rel, false)
	if err != nil {
		return err
	}
	defer r.Close()
	if name == "." {
		return fmt.Errorf("目录已存在")
	}
	return r.Mkdir(name, 0755)
}
func (m *Manager) Mkdir(root, rel string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, name, err := m.scoped(root, rel, false)
	if err != nil {
		return err
	}
	defer r.Close()
	return r.MkdirAll(name, 0755)
}
func (m *Manager) Rename(root, from, to string) error { return m.Transfer(root, from, root, to, true) }
func (m *Manager) guardRoot(root, name string) error {
	if name == "." {
		return fmt.Errorf("不能修改文件根目录")
	}
	return nil
}

func (m *Manager) Transfer(srcRoot, from, dstRoot, to string, move bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	src, srcName, err := m.scoped(srcRoot, from, false)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, dstName, err := m.scoped(dstRoot, to, false)
	if err != nil {
		return err
	}
	defer dst.Close()
	if err := m.guardRoot(srcRoot, srcName); err != nil {
		return err
	}
	if dstName == "." {
		return fmt.Errorf("请指定目标名称")
	}
	srcFull := filepath.Join(m.roots[srcRoot], srcName)
	dstFull := filepath.Join(m.roots[dstRoot], dstName)
	if within(srcFull, dstFull) {
		return fmt.Errorf("目标不能是源目录本身或其子目录")
	}
	if _, err := dst.Lstat(dstName); err == nil {
		return fmt.Errorf("目标已存在")
	} else if !os.IsNotExist(err) {
		return err
	}
	if move && srcRoot == dstRoot {
		err := renameNoReplace(src, srcName, dst, dstName)
		if err == nil {
			return nil
		}
		if !isCrossDevice(err) {
			return err
		}
	}
	info, err := src.Lstat(srcName)
	if err != nil {
		return err
	}
	tmp := filepath.Join(filepath.Dir(dstName), randomName())
	if err = m.copyNode(src, srcName, dst, tmp, info, 0); err != nil {
		_ = dst.RemoveAll(tmp)
		return err
	}
	if err = renameNoReplace(dst, tmp, dst, dstName); err != nil {
		_ = dst.RemoveAll(tmp)
	}
	if err == nil && move {
		if removeErr := src.RemoveAll(srcName); removeErr != nil {
			return fmt.Errorf("已复制，但源文件未删除：%w", removeErr)
		}
	}
	return err
}
func (m *Manager) copyNode(src *os.Root, from string, dst *os.Root, to string, info fs.FileInfo, depth int) error {
	if src.Name() != m.roots["server"] && isPrivateKey(from) {
		return fmt.Errorf("证书私钥不可复制")
	}
	if depth > 128 {
		return fmt.Errorf("目录层级过深")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := src.Readlink(from)
		if err != nil {
			return err
		}
		return dst.Symlink(target, to)
	}
	if info.IsDir() {
		if err := dst.Mkdir(to, info.Mode().Perm()|0700); err != nil {
			return err
		}
		dir, err := src.Open(from)
		if err != nil {
			return err
		}
		entries, err := dir.ReadDir(-1)
		dir.Close()
		if err != nil {
			return err
		}
		for _, entry := range entries {
			child, err := entry.Info()
			if err != nil {
				return err
			}
			if err := m.copyNode(src, filepath.Join(from, entry.Name()), dst, filepath.Join(to, entry.Name()), child, depth+1); err != nil {
				return err
			}
		}
		if err := dst.Chmod(to, info.Mode().Perm()); err != nil {
			return err
		}
		return dst.Chtimes(to, info.ModTime(), info.ModTime())
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("不支持复制设备或特殊文件")
	}
	in, err := src.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := dst.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if err == nil {
		err = out.Chmod(info.Mode().Perm())
	}
	closeErr := out.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return dst.Chtimes(to, info.ModTime(), info.ModTime())
}
func (m *Manager) Delete(root, rel string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, name, err := m.scoped(root, rel, false)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := m.guardRoot(root, name); err != nil {
		return err
	}
	return r.RemoveAll(name)
}
func (m *Manager) Chmod(root, rel, mode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var permission uint32
	if len(mode) < 3 || len(mode) > 4 {
		return fmt.Errorf("请输入 3 或 4 位八进制权限")
	}
	for _, char := range mode {
		if char < '0' || char > '7' {
			return fmt.Errorf("权限只能包含 0–7")
		}
		permission = permission*8 + uint32(char-'0')
	}
	if permission > 0777 {
		return fmt.Errorf("只支持 0000–0777 权限")
	}
	r, name, err := m.scoped(root, rel, false)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := m.guardRoot(root, name); err != nil {
		return err
	}
	info, err := r.Lstat(name)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("请修改符号链接目标文件的权限")
	}
	return r.Chmod(name, fs.FileMode(permission))
}

func (m *Manager) Zip(root, rel, destRel string) error {
	return m.Archive(root, []string{rel}, destRel)
}
func (m *Manager) Archive(root string, paths []string, destRel string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(paths) == 0 {
		return fmt.Errorf("请选择要压缩的文件")
	}
	dst, name, err := m.scoped(root, destRel, false)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := dst.Lstat(name); err == nil {
		return fmt.Errorf("压缩包已存在")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := dst.MkdirAll(filepath.Dir(name), 0755); err != nil {
		return err
	}
	temp := filepath.Join(filepath.Dir(name), randomName())
	output, err := dst.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer dst.Remove(temp)
	writer := zip.NewWriter(output)
	var count int
	var total uint64
	for _, path := range paths {
		src, srcName, err := m.scoped(root, path, false)
		if err != nil {
			writer.Close()
			output.Close()
			return err
		}
		if srcName == "." {
			src.Close()
			writer.Close()
			output.Close()
			return fmt.Errorf("不能打包整个文件根")
		}
		archiveErr := fs.WalkDir(src.FS(), filepath.ToSlash(srcName), func(current string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if current == filepath.ToSlash(temp) {
				return nil
			}
			if root != "server" && isPrivateKey(current) {
				return fmt.Errorf("证书私钥不可打包")
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			count++
			if count > MaxExtractFiles {
				return fmt.Errorf("文件数量超过 10000")
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("压缩暂不支持符号链接：%s", entry.Name())
			}
			if !info.IsDir() && !info.Mode().IsRegular() {
				return fmt.Errorf("不能打包特殊文件")
			}
			if info.Size() > 0 && !info.IsDir() {
				total += uint64(info.Size())
				if total > MaxExtractSize {
					return fmt.Errorf("打包数据超过 1 GB")
				}
			}
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(filepath.Dir(srcName), filepath.FromSlash(current))
			if err != nil {
				return err
			}
			header.Name = filepath.ToSlash(relative)
			header.Method = zip.Deflate
			if info.IsDir() {
				header.Name += "/"
			}
			target, err := writer.CreateHeader(header)
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			file, err := src.Open(current)
			if err != nil {
				return err
			}
			_, err = io.Copy(target, file)
			file.Close()
			return err
		})
		src.Close()
		if archiveErr != nil {
			writer.Close()
			output.Close()
			return archiveErr
		}
	}
	if err := writer.Close(); err != nil {
		output.Close()
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	return dst.Link(temp, name)
}

func (m *Manager) Unzip(root, zipRel, destRel string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	input, err := m.Open(root, zipRel)
	if err != nil {
		return err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return err
	}
	reader, err := zip.NewReader(input, info.Size())
	if err != nil {
		return err
	}
	if len(reader.File) > MaxExtractFiles {
		return fmt.Errorf("压缩包包含超过 10000 个文件")
	}
	var total uint64
	for _, file := range reader.File {
		if file.UncompressedSize64 > MaxExtractSize-total {
			return fmt.Errorf("解压内容超过 1 GB")
		}
		total += file.UncompressedSize64
		name := filepath.FromSlash(file.Name)
		if filepath.IsAbs(name) || !filepath.IsLocal(name) || file.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("压缩包路径或符号链接不允许")
		}
		for _, part := range strings.Split(filepath.ToSlash(name), "/") {
			if part == ".." {
				return fmt.Errorf("压缩包路径不允许")
			}
		}
		if root != "server" && isPrivateKey(name) {
			return fmt.Errorf("压缩包包含证书私钥")
		}
		r, target, err := m.scoped(root, filepath.Join(destRel, name), false)
		if err != nil {
			return err
		}
		_, statErr := r.Lstat(target)
		r.Close()
		if statErr == nil && !file.FileInfo().IsDir() {
			return fmt.Errorf("目标文件已存在：%s", file.Name)
		}
		if statErr != nil && !os.IsNotExist(statErr) {
			return statErr
		}
	}
	for _, file := range reader.File {
		target := filepath.Join(destRel, filepath.FromSlash(file.Name))
		if file.FileInfo().IsDir() {
			r, name, err := m.scoped(root, target, false)
			if err != nil {
				return err
			}
			err = r.MkdirAll(name, 0755)
			r.Close()
			if err != nil {
				return err
			}
			continue
		}
		in, err := file.Open()
		if err != nil {
			return err
		}
		err = m.writeStream(root, target, in, int64(file.UncompressedSize64), false)
		in.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
