package httpapi

import (
	"archive/zip"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGoalDataJobs(t *testing.T) {
	var dataDir string
	seen := &acmeSeen{}
	directory := startACMEDirectory(t, &dataDir, seen)
	t.Setenv("CAMO_ACME_DIRECTORY", directory)
	router, dir := goalRouter(t, "", "demo")
	dataDir = dir
	created := callAs(router, "POST", "/api/v1/sites", `{"name":"filesite","primary_domain":"files.local","type":"static"}`, "", "")
	mustCode(t, created, 201)
	siteID := jsonField(t, created.Body.Bytes(), "id")
	edited := callAs(router, "PUT", "/api/v1/files", `{"root":"sites","path":"`+siteID+`/index.html","content":"hello-camo-file"}`, "", "")
	mustCode(t, edited, 200)
	got := callAs(router, "GET", "/api/v1/sites/"+siteID+"/files/index.html", "", "", "")
	mustCode(t, got, 200)
	if !strings.Contains(got.Body.String(), "hello-camo-file") {
		t.Fatalf("site file unchanged: %s", got.Body.String())
	}
	for _, path := range []string{"/etc/passwd", "../../etc/passwd", siteID + "/privkey.pem"} {
		rec := callAs(router, "GET", "/api/v1/files/content?root=sites&path="+path, "", "", "")
		if rec.Code == 200 {
			t.Fatalf("path accepted: %s %s", path, rec.Body.String())
		}
	}
	_ = os.WriteFile(filepath.Join(dir, "sites", siteID, "secret.key"), []byte("nope"), 0o600)
	if rec := callAs(router, "GET", "/api/v1/files/content?root=sites&path="+siteID+"/secret.key", "", "", ""); rec.Code == 200 {
		t.Fatal("private key was readable")
	}
	mustCode(t, callAs(router, "POST", "/api/v1/files/mkdir", `{"root":"apps","path":"demo/data"}`, "", ""), 201)
	mustCode(t, callAs(router, "POST", "/api/v1/files/upload", `{"root":"apps","path":"demo/data/note.txt","content":"app-data"}`, "", ""), 201)
	mustCode(t, callAs(router, "POST", "/api/v1/files/rename", `{"root":"apps","from":"demo/data/note.txt","to":"demo/data/renamed.txt"}`, "", ""), 200)
	mustCode(t, callAs(router, "POST", "/api/v1/files/zip", `{"root":"apps","path":"demo/data","dest":"demo/data.zip"}`, "", ""), 201)
	mustCode(t, callAs(router, "POST", "/api/v1/files/unzip", `{"root":"apps","zip":"demo/data.zip","dest":"demo/unzipped"}`, "", ""), 201)
	mustCode(t, callAs(router, "DELETE", "/api/v1/files?root=apps&path=demo/data/renamed.txt", "", "", ""), 200)

	mustCode(t, callAs(router, "POST", "/api/v1/databases/mysql/databases", `{"name":"appdb"}`, "", ""), 201)
	mustCode(t, callAs(router, "POST", "/api/v1/databases/mysql/users", `{"name":"app","password":"db-pass-should-not-log"}`, "", ""), 201)
	exported := callAs(router, "POST", "/api/v1/databases/mysql/export", `{}`, "", "")
	mustCode(t, exported, 200)
	if !strings.Contains(exported.Body.String(), "CREATE DATABASE appdb") || strings.Contains(exported.Body.String(), "db-pass-should-not-log") {
		t.Fatalf("export: %s", exported.Body.String())
	}
	sql := jsonField(t, exported.Body.Bytes(), "sql")
	mustCode(t, callAs(router, "POST", "/api/v1/databases/postgres/import", `{"sql":`+mustJSON(sql)+`}`, "", ""), 200)
	again := callAs(router, "POST", "/api/v1/databases/postgres/export", `{}`, "", "")
	mustCode(t, again, 200)
	if !strings.Contains(again.Body.String(), "CREATE DATABASE appdb") {
		t.Fatalf("import missing: %s", again.Body.String())
	}
	mustCode(t, callAs(router, "POST", "/api/v1/databases/mysql/exec", `{"sql":"SELECT 1","password":"db-pass-should-not-log"}`, "", ""), 200)
	mustCode(t, callAs(router, "POST", "/api/v1/databases/redis/keys", `{"key":"user:1"}`, "", ""), 201)
	mustCode(t, callAs(router, "POST", "/api/v1/databases/redis/keys", `{"key":"user:2"}`, "", ""), 201)
	mustCode(t, callAs(router, "POST", "/api/v1/databases/redis/keys", `{"key":"other"}`, "", ""), 201)
	scanned := callAs(router, "GET", "/api/v1/databases/redis/keys?prefix=user:&limit=1", "", "", "")
	mustCode(t, scanned, 200)
	if strings.Count(scanned.Body.String(), "user:") != 1 {
		t.Fatalf("scan not limited: %s", scanned.Body.String())
	}
	mustCode(t, callAs(router, "DELETE", "/api/v1/databases/redis/keys?key=user:1", "", "", ""), 200)
	left := callAs(router, "GET", "/api/v1/databases/redis/keys?prefix=user:", "", "", "")
	if strings.Contains(left.Body.String(), "user:1") {
		t.Fatalf("key still present: %s", left.Body.String())
	}
	missing := callAs(router, "GET", "/api/v1/databases/mongo", "", "", "")
	mustCode(t, missing, 200)
	if !strings.Contains(missing.Body.String(), `"installed":false`) || !strings.Contains(missing.Body.String(), "/store?app=mongo") {
		t.Fatalf("missing db: %s", missing.Body.String())
	}
	auditRaw, _ := os.ReadFile(filepath.Join(dir, "audit.jsonl"))
	if !strings.Contains(string(auditRaw), "SELECT 1") || strings.Contains(string(auditRaw), "db-pass-should-not-log") {
		t.Fatalf("sql audit: %s", auditRaw)
	}

	backup := callAs(router, "POST", "/api/v1/backups/sites/"+siteID, `{}`, "", "")
	mustCode(t, backup, 201)
	backupID := jsonField(t, backup.Body.Bytes(), "id")
	mustCode(t, callAs(router, "DELETE", "/api/v1/files?root=sites&path="+siteID+"/index.html", "", "", ""), 200)
	refused := callAs(router, "POST", "/api/v1/backups/"+backupID+"/restore", `{"confirm":false}`, "", "")
	if refused.Code == 200 {
		t.Fatal("restore without confirm succeeded")
	}
	mustCode(t, callAs(router, "POST", "/api/v1/backups/"+backupID+"/restore", `{"confirm":true}`, "", ""), 200)
	restored := callAs(router, "GET", "/api/v1/sites/"+siteID+"/files/index.html", "", "", "")
	if !strings.Contains(restored.Body.String(), "hello-camo-file") {
		t.Fatalf("restore failed: %s", restored.Body.String())
	}
	var sawAuth bool
	var leaked bool
	s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		auth := r.Header.Get("Authorization")
		joined := auth + "\n" + string(body) + "\n" + r.Header.Get("x-amz-date") + "\n" + r.Header.Get("x-amz-content-sha256")
		if strings.Contains(joined, "s3-secret-value") {
			leaked = true
		}
		sum := sha256.Sum256(body)
		payloadHex := hex.EncodeToString(sum[:])
		amzDate := r.Header.Get("x-amz-date")
		if !strings.Contains(auth, "Signature=") || r.Header.Get("x-amz-content-sha256") != payloadHex || len(amzDate) < 8 {
			http.Error(w, "unsigned", 401)
			return
		}
		want := s3v4Signature(r.URL.EscapedPath(), strings.ToLower(r.Host), amzDate, r.Header.Get("Content-Type"), payloadHex, "s3-secret-value")
		if authParam(auth, "Signature") != want {
			http.Error(w, "bad-signature", 401)
			return
		}
		sawAuth = true
		w.WriteHeader(200)
	}))
	defer s3.Close()
	published := callAs(router, "POST", "/api/v1/backups/"+backupID+"/publish", `{"endpoint":"`+s3.URL+`","bucket":"camo","access_key":"AKIA","secret":"s3-secret-value"}`, "", "")
	mustCode(t, published, 200)
	if !sawAuth || leaked || strings.Contains(published.Body.String(), "s3-secret-value") {
		t.Fatalf("s3 auth=%v leaked=%v body=%s", sawAuth, leaked, published.Body.String())
	}
	var first string
	for i := 0; i < 6; i++ {
		rec := callAs(router, "POST", "/api/v1/backups/sites/"+siteID, `{}`, "", "")
		mustCode(t, rec, 201)
		if first == "" {
			first = jsonField(t, rec.Body.Bytes(), "id")
		}
	}
	listed := callAs(router, "GET", "/api/v1/backups", "", "", "")
	mustCode(t, listed, 200)
	if strings.Contains(listed.Body.String(), first) {
		t.Fatalf("oldest backup kept: %s", listed.Body.String())
	}
	slip := zipSlip()
	imported := callAs(router, "POST", "/api/v1/backups/import", `{"kind":"site","source":"slip-site","zip":"`+slip+`"}`, "", "")
	mustCode(t, imported, 201)
	slipID := jsonField(t, imported.Body.Bytes(), "id")
	mustCode(t, callAs(router, "POST", "/api/v1/backups/"+slipID+"/restore", `{"confirm":true}`, "", ""), 200)
	if _, err := os.Stat(filepath.Join(dir, "slip-outside.txt")); err == nil {
		t.Fatal("restore wrote outside the site directory")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "slip-outside.txt")); err == nil {
		t.Fatal("restore wrote outside the data root")
	}

	job := callAs(router, "POST", "/api/v1/schedules", `{"name":"minute","type":"backup_site","cron":"* * * * *","payload":{"site_id":"`+siteID+`"}}`, "", "")
	mustCode(t, job, 201)
	jobID := jsonField(t, job.Body.Bytes(), "id")
	ran := callAs(router, "POST", "/api/v1/schedules/"+jobID+"/run", `{}`, "", "")
	mustCode(t, ran, 200)
	if !strings.Contains(ran.Body.String(), `"status":"success"`) && !strings.Contains(ran.Body.String(), `"status":"failed"`) {
		t.Fatalf("backup run: %s", ran.Body.String())
	}
	mustCode(t, callAs(router, "POST", "/api/v1/schedules/"+jobID+"/enabled", `{"enabled":false}`, "", ""), 200)
	mustCode(t, callAs(router, "POST", "/api/v1/schedules/scripts", `{"name":"slow.sh","content":"sleep 30\n"}`, "", ""), 201)
	scriptJob := callAs(router, "POST", "/api/v1/schedules", `{"name":"slow","type":"script","cron":"* * * * *","timeout_sec":1,"payload":{"script":"slow.sh"}}`, "", "")
	mustCode(t, scriptJob, 201)
	scriptID := jsonField(t, scriptJob.Body.Bytes(), "id")
	timed := callAs(router, "POST", "/api/v1/schedules/"+scriptID+"/run", `{}`, "", "")
	mustCode(t, timed, 200)
	if !strings.Contains(timed.Body.String(), `"status":"timeout"`) {
		t.Fatalf("script: %s", timed.Body.String())
	}
	past := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	once := callAs(router, "POST", "/api/v1/schedules", `{"name":"once","type":"backup_site","once":true,"run_at":"`+past+`","payload":{"site_id":"`+siteID+`"}}`, "", "")
	mustCode(t, once, 201)
	onceID := jsonField(t, once.Body.Bytes(), "id")
	reloaded := callAs(router, "POST", "/api/v1/schedules/reload", `{}`, "", "")
	mustCode(t, reloaded, 200)
	if !strings.Contains(reloaded.Body.String(), jobID) {
		t.Fatal("job disappeared after reload")
	}
	var jobs []struct {
		ID     string `json:"id"`
		Missed bool   `json:"missed"`
		Runs   []any  `json:"runs"`
	}
	if err := json.Unmarshal(reloaded.Body.Bytes(), &jobs); err != nil {
		t.Fatal(err)
	}
	for _, item := range jobs {
		if item.ID == onceID {
			if !item.Missed || len(item.Runs) != 0 {
				t.Fatalf("missed job replayed: %+v", item)
			}
		}
	}
	expiry := time.Now().Add(10 * 24 * time.Hour).UTC().Format(time.RFC3339)
	issued := callAs(router, "POST", "/api/v1/certificates/issued", `{"provider":"letsencrypt","domains":["renew.example"],"expiry":"`+expiry+`"}`, "", "")
	mustCode(t, issued, 201)
	certID := jsonField(t, issued.Body.Bytes(), "id")
	before := callAs(router, "GET", "/api/v1/certificates", "", "", "")
	if strings.Contains(before.Body.String(), "HTTP-01") {
		t.Fatal("renewal ran before the scheduler")
	}
	renewJob := callAs(router, "POST", "/api/v1/schedules", `{"name":"renew","type":"renew_certs","cron":"* * * * *"}`, "", "")
	mustCode(t, renewJob, 201)
	mustCode(t, callAs(router, "POST", "/api/v1/schedules/"+jsonField(t, renewJob.Body.Bytes(), "id")+"/run", `{}`, "", ""), 200)
	after := callAs(router, "GET", "/api/v1/certificates", "", "", "")
	mustCode(t, after, 200)
	order, challenge := seen.snapshot()
	body := after.Body.String()
	if !strings.Contains(body, certID) || !strings.Contains(body, "http-01") || !strings.Contains(body, `"status":"failed"`) {
		t.Fatalf("renewal: %s", body)
	}
	if !order || !challenge || !strings.Contains(body, "stand-in rejected the challenge") {
		t.Fatalf("http-01 was not submitted order=%v challenge=%v body=%s", order, challenge, body)
	}
	if strings.Contains(body, "HTTP-01 验证未完成") {
		t.Fatalf("renewal kept the canned failure: %s", body)
	}
	mustCode(t, callAs(router, "GET", "/api/health", "", "", ""), 200)
}

func mustJSON(value string) string {
	b, _ := json.Marshal(value)
	return string(b)
}

func authParam(header, key string) string {
	_, rest, ok := strings.Cut(header, " ")
	if !ok {
		return ""
	}
	for _, part := range strings.Split(rest, ",") {
		name, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && name == key {
			return value
		}
	}
	return ""
}

func s3v4Signature(uri, host, amzDate, contentType, payloadHex, secret string) string {
	canonicalHeaders := "content-type:" + contentType + "\n" +
		"host:" + host + "\n" +
		"x-amz-content-sha256:" + payloadHex + "\n" +
		"x-amz-date:" + amzDate + "\n"
	signedHeaders := "content-type;host;x-amz-content-sha256;x-amz-date"
	canonical := "PUT\n" + uri + "\n\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + payloadHex
	sum := sha256.Sum256([]byte(canonical))
	date := amzDate[:8]
	scope := date + "/us-east-1/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + hex.EncodeToString(sum[:])
	key := hmacSHA256(hmacSHA256(hmacSHA256(hmacSHA256([]byte("AWS4"+secret), date), "us-east-1"), "s3"), "aws4_request")
	return hex.EncodeToString(hmacSHA256(key, stringToSign))
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

type acmeSeen struct {
	mu        sync.Mutex
	order     bool
	challenge bool
	posts     int
}

func (a *acmeSeen) markOrder() {
	a.mu.Lock()
	a.order = true
	a.mu.Unlock()
}

func (a *acmeSeen) beginChallenge() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.posts++
	return a.posts
}

func (a *acmeSeen) markChallenge() {
	a.mu.Lock()
	a.challenge = true
	a.mu.Unlock()
}

func (a *acmeSeen) snapshot() (bool, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.order, a.challenge
}

func startACMEDirectory(t *testing.T, dataDir *string, seen *acmeSeen) string {
	t.Helper()
	const token = "camohttp01token"
	var seq atomic.Int64
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Replay-Nonce", fmt.Sprintf("n%d", seq.Add(1)))
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = io.Copy(io.Discard, r.Body)
		base := srv.URL
		switch r.URL.Path {
		case "/":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"newNonce":   base + "/new-nonce",
				"newAccount": base + "/new-account",
				"newOrder":   base + "/new-order",
				"revokeCert": base + "/revoke-cert",
				"keyChange":  base + "/key-change",
			})
		case "/new-nonce":
			w.WriteHeader(http.StatusOK)
		case "/new-account":
			w.Header().Set("Location", base+"/accounts/1")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":"valid"}`))
		case "/new-order":
			seen.markOrder()
			w.Header().Set("Location", base+"/orders/0")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, `{"status":"pending","authorizations":[%q],"finalize":%q}`, base+"/authz/0", base+"/finalize/0")
		case "/authz/0":
			fmt.Fprintf(w, `{"status":"pending","identifier":{"type":"dns","value":"renew.example"},"challenges":[{"type":"http-01","url":%q,"token":%q,"status":"pending"}]}`, base+"/challenge/http-01", token)
		case "/challenge/http-01":
			if seen.beginChallenge() == 1 {
				path := filepath.Join(*dataDir, "acme", ".well-known", "acme-challenge", token)
				if b, err := os.ReadFile(path); err == nil && len(bytes.TrimSpace(b)) > 0 {
					seen.markChallenge()
				}
				fmt.Fprintf(w, `{"type":"http-01","url":%q,"token":%q,"status":"pending"}`, base+"/challenge/http-01", token)
				return
			}
			fmt.Fprintf(w, `{"status":"invalid","identifier":{"type":"dns","value":"renew.example"},"challenges":[{"type":"http-01","url":%q,"token":%q,"status":"invalid","error":{"type":"urn:ietf:params:acme:error:connection","detail":"stand-in rejected the challenge","status":400}}]}`, base+"/challenge/http-01", token)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func zipSlip() string {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	ok, _ := zw.Create("payload/ok.txt")
	_, _ = ok.Write([]byte("ok"))
	evil, _ := zw.Create("payload/../../slip-outside.txt")
	_, _ = evil.Write([]byte("evil"))
	_ = zw.Close()
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
