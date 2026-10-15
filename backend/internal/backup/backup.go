package backup

import (
	"archive/zip"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Item struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Source    string `json:"source"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
	Checksum  string `json:"checksum"`
	Remote    string `json:"remote,omitempty"`
}

type Store struct {
	dir   string
	data  string
	keep  int
	items []Item
}

func Open(dataDir string, keep int) (*Store, error) {
	if keep <= 0 {
		keep = 5
	}
	dir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, data: dataDir, keep: keep}
	b, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &s.items)
	}
	if s.items == nil {
		s.items = []Item{}
	}
	return s, nil
}

func (s *Store) List() []Item {
	out := append([]Item{}, s.items...)
	return out
}

func (s *Store) CreateSite(siteID string) (Item, error) {
	src := filepath.Join(s.data, "sites", siteID)
	if _, err := os.Stat(src); err != nil {
		return Item{}, fmt.Errorf("网站目录不存在")
	}
	return s.archive("site", siteID, src)
}

func (s *Store) CreateDir(rel string) (Item, error) {
	if filepath.IsAbs(rel) || strings.Contains(rel, "..") {
		return Item{}, fmt.Errorf("目录不在面板数据内")
	}
	src := filepath.Join(s.data, rel)
	absData, _ := filepath.Abs(s.data)
	absSrc, _ := filepath.Abs(src)
	if absSrc != absData && !strings.HasPrefix(absSrc, absData+string(os.PathSeparator)) {
		return Item{}, fmt.Errorf("目录不在面板数据内")
	}
	return s.archive("dir", rel, src)
}

func (s *Store) CreateBytes(kind, source, name string, payload []byte) (Item, error) {
	id := fmt.Sprintf("%s-%d", kind, time.Now().UnixNano())
	path := filepath.Join(s.dir, id+".zip")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return Item{}, err
	}
	item := Item{ID: id, Kind: kind, Source: source, Path: path, CreatedAt: time.Now().UTC().Format(time.RFC3339), Checksum: fmt.Sprintf("%d", len(payload))}
	s.items = append(s.items, item)
	s.enforceKeep(kind, source)
	return item, s.save()
}

func (s *Store) archive(kind, source, src string) (Item, error) {
	id := fmt.Sprintf("%s-%d", kind, time.Now().UnixNano())
	path := filepath.Join(s.dir, id+".zip")
	f, err := os.Create(path)
	if err != nil {
		return Item{}, err
	}
	zw := zip.NewWriter(f)
	manifest, _ := json.Marshal(map[string]string{"kind": kind, "source": source, "at": time.Now().UTC().Format(time.RFC3339)})
	mw, _ := zw.Create("manifest.json")
	_, _ = mw.Write(manifest)
	if err := addTree(zw, src, "payload"); err != nil {
		zw.Close()
		f.Close()
		return Item{}, err
	}
	if err := zw.Close(); err != nil {
		f.Close()
		return Item{}, err
	}
	info, _ := f.Stat()
	f.Close()
	item := Item{ID: id, Kind: kind, Source: source, Path: path, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	if info != nil {
		item.Checksum = fmt.Sprintf("%d", info.Size())
	}
	s.items = append(s.items, item)
	s.enforceKeep(kind, source)
	return item, s.save()
}

func addTree(zw *zip.Writer, src, prefix string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return addFile(zw, src, prefix+"/"+filepath.Base(src))
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		return addFile(zw, path, prefix+"/"+filepath.ToSlash(rel))
	})
}

func addFile(zw *zip.Writer, path, name string) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	_, err = io.Copy(w, in)
	return err
}

func (s *Store) Restore(id string, confirm bool) error {
	if !confirm {
		return fmt.Errorf("恢复前需要确认")
	}
	item, ok := s.find(id)
	if !ok {
		return fmt.Errorf("备份不存在")
	}
	r, err := zip.OpenReader(item.Path)
	if err != nil {
		return err
	}
	defer r.Close()
	var dest string
	switch item.Kind {
	case "site":
		dest = filepath.Join(s.data, "sites", item.Source)
	case "dir":
		dest = filepath.Join(s.data, item.Source)
	default:
		dest = filepath.Join(s.data, "restores", item.ID)
	}
	absData, _ := filepath.Abs(s.data)
	absDest, _ := filepath.Abs(dest)
	if absDest != absData && !strings.HasPrefix(absDest, absData+string(os.PathSeparator)) {
		return fmt.Errorf("恢复目标超出面板数据目录")
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	for _, file := range r.File {
		if !strings.HasPrefix(file.Name, "payload/") || strings.Contains(file.Name, "..") {
			continue
		}
		rel := strings.TrimPrefix(file.Name, "payload/")
		target := filepath.Join(dest, filepath.FromSlash(rel))
		absTarget, _ := filepath.Abs(target)
		if absTarget != absDest && !strings.HasPrefix(absTarget, absDest+string(os.PathSeparator)) {
			return fmt.Errorf("恢复目标超出面板数据目录")
		}
		if file.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		out.Close()
		in.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

func (s *Store) Publish(id, endpoint, bucket, access, secret string) (string, error) {
	item, ok := s.find(id)
	if !ok {
		return "", fmt.Errorf("备份不存在")
	}
	body, err := os.ReadFile(item.Path)
	if err != nil {
		return "", err
	}
	object := item.ID + ".zip"
	url := strings.TrimRight(endpoint, "/") + "/" + bucket + "/" + object
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/zip")
	req.Header.Set("X-Camo-Checksum", item.Checksum)
	signS3(req, body, access, secret)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("对象存储返回 %d %s", resp.StatusCode, string(b))
	}
	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].Remote = url
			item = s.items[i]
		}
	}
	return url, s.save()
}

const s3Region = "us-east-1"

// signS3 sets AWS Signature Version 4 headers for a path-style PUT.
// The secret is an HMAC key and is not copied into any header.
func signS3(req *http.Request, body []byte, access, secret string) {
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	payloadSum := sha256.Sum256(body)
	payloadHex := hex.EncodeToString(payloadSum[:])
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHex)
	host := strings.ToLower(req.Host)
	if host == "" {
		host = strings.ToLower(req.URL.Host)
	}
	uri := req.URL.EscapedPath()
	if uri == "" {
		uri = "/"
	}
	contentType := req.Header.Get("Content-Type")
	canonicalHeaders := "content-type:" + contentType + "\n" +
		"host:" + host + "\n" +
		"x-amz-content-sha256:" + payloadHex + "\n" +
		"x-amz-date:" + amzDate + "\n"
	signedHeaders := "content-type;host;x-amz-content-sha256;x-amz-date"
	canonical := "PUT\n" + uri + "\n\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + payloadHex
	canonicalSum := sha256.Sum256([]byte(canonical))
	date := amzDate[:8]
	scope := date + "/" + s3Region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + hex.EncodeToString(canonicalSum[:])
	key := hmacSHA256(hmacSHA256(hmacSHA256(hmacSHA256([]byte("AWS4"+secret), date), s3Region), "s3"), "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(key, stringToSign))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+access+"/"+scope+", SignedHeaders="+signedHeaders+", Signature="+signature)
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func (s *Store) enforceKeep(kind, source string) {
	var matching []Item
	var rest []Item
	for _, item := range s.items {
		if item.Kind == kind && item.Source == source {
			matching = append(matching, item)
		} else {
			rest = append(rest, item)
		}
	}
	sort.Slice(matching, func(i, j int) bool { return matching[i].CreatedAt < matching[j].CreatedAt })
	for len(matching) > s.keep {
		oldest := matching[0]
		_ = os.Remove(oldest.Path)
		matching = matching[1:]
	}
	s.items = append(rest, matching...)
}

func (s *Store) find(id string) (Item, bool) {
	for _, item := range s.items {
		if item.ID == id {
			return item, true
		}
	}
	return Item{}, false
}

func (s *Store) save() error {
	b, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, "index.json"), b, 0o644)
}
