package httpapi

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGoalSitesDocker(t *testing.T) {
	router, dir := goalRouter(t, "")
	legacy := callAs(router, "POST", "/api/v1/docker/containers", `{"name":"legacy-web","image":"nginx:alpine","host_port":8091,"container_port":80}`, "", "")
	mustCode(t, legacy, 201)
	badMount := callAs(router, "POST", "/api/v1/docker/containers", `{"name":"mount-bad","image":"nginx:alpine","binds":["/etc/hosts:/mnt:ro"]}`, "", "")
	if badMount.Code == 201 {
		t.Fatalf("unconfirmed mount accepted: %s", badMount.Body.String())
	}
	mustCode(t, callAs(router, "POST", "/api/v1/docker/containers", `{"name":"mount-ok","image":"nginx:alpine","binds":["/etc/hosts:/mnt:ro"],"confirm_host_mount":true}`, "", ""), 201)
	created := callAs(router, "POST", "/api/v1/docker/containers", `{"name":"multi-web","image":"nginx:alpine","ports":[{"host_port":18081,"container_port":80},{"host_port":18082,"container_port":443}],"env":["FOO=bar"],"command":["nginx","-g","daemon off;"],"restart_policy":"unless-stopped"}`, "", "")
	mustCode(t, created, 201)
	id := jsonField(t, created.Body.Bytes(), "id")
	detail := callAs(router, "GET", "/api/v1/docker/containers/"+id, "", "", "")
	mustCode(t, detail, 200)
	if !strings.Contains(detail.Body.String(), "FOO=bar") || !strings.Contains(detail.Body.String(), "unless-stopped") || !strings.Contains(detail.Body.String(), "18081") || !strings.Contains(detail.Body.String(), "nginx") {
		t.Fatalf("detail: %s", detail.Body.String())
	}
	unknown := callAs(router, "POST", "/api/v1/docker/containers", `{"name":"not-local","image":"example.invalid:1"}`, "", "")
	if unknown.Code == 201 {
		t.Fatal("pull was implicit")
	}
	pulled := callAs(router, "POST", "/api/v1/docker/images/pull", `{"image":"example.invalid:1"}`, "", "")
	mustCode(t, pulled, 200)
	if !strings.Contains(pulled.Body.String(), `"progress":100`) {
		t.Fatalf("pull: %s", pulled.Body.String())
	}
	mustCode(t, callAs(router, "POST", "/api/v1/docker/containers", `{"name":"pulled-box","image":"example.invalid:1","pull":true}`, "", ""), 201)
	mustCode(t, callAs(router, "POST", "/api/v1/docker/containers/c3d4e5f6a1b2/pause", "", "", ""), 200)
	mustCode(t, callAs(router, "POST", "/api/v1/docker/containers/c3d4e5f6a1b2/unpause", "", "", ""), 200)
	stats := callAs(router, "GET", "/api/v1/docker/containers/c3d4e5f6a1b2/stats", "", "", "")
	mustCode(t, stats, 200)
	var statBody struct {
		CPU    float64 `json:"cpu_percent"`
		Memory int64   `json:"memory_bytes"`
	}
	if json.Unmarshal(stats.Body.Bytes(), &statBody) != nil || statBody.CPU <= 0 || statBody.Memory <= 0 {
		t.Fatalf("stats: %s", stats.Body.String())
	}
	stream := callAs(router, "GET", "/api/v1/docker/containers/a1b2c3d4e5f6/logs/stream?tail=2", "", "", "")
	mustCode(t, stream, 200)
	if !strings.Contains(stream.Header().Get("Content-Type"), "text/event-stream") || !strings.Contains(stream.Body.String(), "data:") {
		t.Fatalf("stream: %s %s", stream.Header().Get("Content-Type"), stream.Body.String())
	}
	pruned := callAs(router, "POST", "/api/v1/docker/images/prune", `{}`, "", "")
	mustCode(t, pruned, 200)
	if !strings.Contains(pruned.Body.String(), `"removed":1`) {
		t.Fatalf("prune: %s", pruned.Body.String())
	}
	mustCode(t, callAs(router, "DELETE", "/api/v1/docker/images/sha256:444444444444", "", "", ""), 200)
	beforeVol := callAs(router, "GET", "/api/v1/docker/overview", "", "", "")
	mustCode(t, callAs(router, "POST", "/api/v1/docker/containers/a1b2c3d4e5f6/force-remove", `{}`, "", ""), 400)
	mustCode(t, callAs(router, "POST", "/api/v1/docker/containers/a1b2c3d4e5f6/force-remove", `{"confirm":true}`, "", ""), 200)
	afterVol := callAs(router, "GET", "/api/v1/docker/overview", "", "", "")
	if strings.Contains(afterVol.Body.String(), "a1b2c3d4e5f6") || !strings.Contains(afterVol.Body.String(), "postgres-data") || !strings.Contains(beforeVol.Body.String(), "postgres-data") {
		t.Fatal("force remove deleted a volume or left the container")
	}

	staticSite := callAs(router, "POST", "/api/v1/sites", `{"name":"static-a","primary_domain":"static-a.local","type":"static"}`, "", "")
	mustCode(t, staticSite, 201)
	proxySite := callAs(router, "POST", "/api/v1/sites", `{"name":"proxy-a","primary_domain":"proxy-a.local","type":"proxy","proxy_target":"http://127.0.0.1:3000"}`, "", "")
	mustCode(t, proxySite, 201)
	proxyID := jsonField(t, proxySite.Body.Bytes(), "id")
	mustCode(t, callAs(router, "POST", "/api/v1/sites/"+proxyID+"/ssl/apply", `{"provider":"self-signed","force_https":true}`, "", ""), 200)
	proxyConf := callAs(router, "GET", "/api/v1/sites/"+proxyID+"/config", "", "", "")
	mustCode(t, proxyConf, 200)
	if !strings.Contains(proxyConf.Body.String(), "listen 443 ssl") || !strings.Contains(proxyConf.Body.String(), "proxy_pass http://127.0.0.1:3000") || !strings.Contains(proxyConf.Body.String(), "/etc/nginx/ssl/"+proxyID+"/fullchain.pem") {
		t.Fatalf("existing https: %s", proxyConf.Body.String())
	}
	issued := callAs(router, "POST", "/api/v1/certificates/issued", `{"provider":"custom","domains":["static-a.local","proxy-b.local"],"expiry":"`+time.Now().Add(20*24*time.Hour).UTC().Format(time.RFC3339)+`"}`, "", "")
	mustCode(t, issued, 201)
	certID := jsonField(t, issued.Body.Bytes(), "id")
	other := callAs(router, "POST", "/api/v1/sites", `{"name":"static-b","primary_domain":"proxy-b.local","type":"static"}`, "", "")
	mustCode(t, other, 201)
	a := jsonField(t, staticSite.Body.Bytes(), "id")
	b := jsonField(t, other.Body.Bytes(), "id")
	mustCode(t, callAs(router, "POST", "/api/v1/certificates/"+certID+"/bind", `{"sites":["`+a+`","`+b+`"]}`, "", ""), 200)
	for _, id := range []string{a, b} {
		conf := callAs(router, "GET", "/api/v1/sites/"+id+"/config", "", "", "")
		if !strings.Contains(conf.Body.String(), "/etc/nginx/ssl/certs/"+certID+"/fullchain.pem") {
			t.Fatalf("shared cert missing on %s: %s", id, conf.Body.String())
		}
	}
	keys := 0
	_ = filepath.Walk(filepath.Join(dir, "ssl", "certs", certID), func(path string, info os.FileInfo, err error) error {
		if err == nil && info.Name() == "privkey.pem" {
			keys++
		}
		return nil
	})
	if keys != 1 {
		t.Fatalf("private key copies=%d", keys)
	}
	listed := callAs(router, "GET", "/api/v1/sites", "", "", "")
	if !strings.Contains(listed.Body.String(), `"days_remaining":`) {
		t.Fatalf("days remaining missing: %s", listed.Body.String())
	}
	badSnippet := callAs(router, "POST", "/api/v1/sites", `{"name":"bad-snippet","primary_domain":"bad-snippet.local","type":"static","custom_config":"include /etc/passwd;"}`, "", "")
	if badSnippet.Code == 201 {
		t.Fatal("unsafe snippet accepted")
	}
	safe := callAs(router, "POST", "/api/v1/sites", `{"name":"ruled","primary_domain":"ruled.local","type":"static","custom_config":"add_header X-Camo 1;","redirects":[{"from":"/old","to":"/new","code":301}],"locations":[{"path":"/api","proxy":"http://127.0.0.1:9"}],"basic_auth_user":"ada","basic_auth_password":"secret-pass","hotlink":true,"rate_limit":20}`, "", "")
	mustCode(t, safe, 201)
	safeID := jsonField(t, safe.Body.Bytes(), "id")
	safeConf := callAs(router, "GET", "/api/v1/sites/"+safeID+"/config", "", "", "")
	for _, piece := range []string{"add_header X-Camo 1;", "return 301 /new;", "location /api", "auth_basic", "valid_referers", "limit_req zone=camo_site", "access_log", "error_log"} {
		if !strings.Contains(safeConf.Body.String(), piece) {
			t.Fatalf("missing %s in %s", piece, safeConf.Body.String())
		}
	}
	if strings.Contains(safeConf.Body.String(), "secret-pass") {
		t.Fatal("basic auth password landed in the config")
	}
	logs := callAs(router, "GET", "/api/v1/sites/"+safeID+"/logs", "", "", "")
	mustCode(t, logs, 200)
	if !strings.Contains(logs.Body.String(), "access") || !strings.Contains(logs.Body.String(), "error") {
		t.Fatalf("logs: %s", logs.Body.String())
	}
	for _, runtime := range []string{"php", "node", "java", "go", "python"} {
		rec := callAs(router, "POST", "/api/v1/sites", `{"name":"rt-`+runtime+`","primary_domain":"`+runtime+`.runtime.local","type":"runtime","runtime":"`+runtime+`"}`, "", "")
		mustCode(t, rec, 201)
		rid := jsonField(t, rec.Body.Bytes(), "id")
		conf := callAs(router, "GET", "/api/v1/sites/"+rid+"/config", "", "", "")
		want := "proxy_pass http://runtime-" + runtime + "-" + rid + ":8080;"
		if !strings.Contains(conf.Body.String(), want) {
			t.Fatalf("%s config: %s", runtime, conf.Body.String())
		}
		overview := callAs(router, "GET", "/api/v1/docker/overview", "", "", "")
		mustCode(t, overview, 200)
		if !strings.Contains(overview.Body.String(), "runtime-"+runtime+"-"+rid) {
			t.Fatalf("%s container missing: %s", runtime, overview.Body.String())
		}
	}

	certPEM, keyPEM := pemPair()
	uploaded := callAs(router, "POST", "/api/v1/certificates/upload", `{"cert_pem":`+mustJSON(certPEM)+`,"key_pem":`+mustJSON(keyPEM)+`,"domains":["upload.example"]}`, "", "")
	mustCode(t, uploaded, 201)
	self := callAs(router, "POST", "/api/v1/certificates/self-signed", `{"domains":["self.example"]}`, "", "")
	mustCode(t, self, 201)
	all := callAs(router, "GET", "/api/v1/certificates", "", "", "")
	if !strings.Contains(all.Body.String(), "upload.example") || !strings.Contains(all.Body.String(), "self.example") || !strings.Contains(all.Body.String(), certID) {
		t.Fatalf("store: %s", all.Body.String())
	}

	dnsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer dns-token-very-secret" {
			http.Error(w, "no", 401)
			return
		}
		w.WriteHeader(204)
	}))
	defer dnsSrv.Close()
	saved := callAs(router, "PUT", "/api/v1/certificates/dns", `{"provider":"example","api_base":"`+dnsSrv.URL+`","zone":"example.com","token":"dns-token-very-secret"}`, "", "")
	mustCode(t, saved, 200)
	if strings.Contains(saved.Body.String(), "dns-token-very-secret") || !strings.Contains(saved.Body.String(), `"token_set":true`) {
		t.Fatalf("dns echo: %s", saved.Body.String())
	}
	manual := callAs(router, "POST", "/api/v1/certificates/dns/manual", `{"name":"_acme-challenge.example.com","value":"txt-value"}`, "", "")
	mustCode(t, manual, 200)
	if !strings.Contains(manual.Body.String(), `"type":"TXT"`) || !strings.Contains(manual.Body.String(), "txt-value") {
		t.Fatalf("manual: %s", manual.Body.String())
	}
	issuedDNS := callAs(router, "POST", "/api/v1/certificates/dns/issue", `{"domains":["example.com"]}`, "", "")
	mustCode(t, issuedDNS, 200)
	if strings.Contains(issuedDNS.Body.String(), "dns-token-very-secret") || !strings.Contains(issuedDNS.Body.String(), "dns-01") {
		t.Fatalf("dns issue: %s", issuedDNS.Body.String())
	}
}

func pemPair() (string, string) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "upload.example"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(48 * time.Hour), DNSNames: []string{"upload.example"}}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	var cert bytesPEM
	cert.Write(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	keyDER, _ := x509.MarshalPKCS8PrivateKey(key)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))
}

type bytesPEM struct{ b []byte }

func (b *bytesPEM) Write(p []byte) (int, error) { b.b = append(b.b, p...); return len(p), nil }
