package docker

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

func TestDockerLogDecoding(t *testing.T) {
	frame := func(stream byte, text string) []byte {
		header := make([]byte, 8)
		header[0] = stream
		binary.BigEndian.PutUint32(header[4:], uint32(len(text)))
		return append(header, []byte(text)...)
	}
	framed := append(frame(1, "stdout\n"), frame(2, "stderr\n")...)
	for _, test := range []struct {
		name  string
		input []byte
		want  string
	}{
		{"multiplexed", framed, "stdout\nstderr\n"}, {"tty", []byte("plain tty output\n"), "plain tty output\n"}, {"empty", nil, ""}, {"truncated", frame(1, "abc")[:9], string(frame(1, "abc")[:9])},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := decodeLogs(test.input); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestCreateUsesDaemonVersionAndLoopbackPortBinding(t *testing.T) {
	requests := 0
	engine := &EngineClient{client: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.URL.Path == "/version" {
			return response(200, `{"ApiVersion":"1.47"}`), nil
		}
		if r.Method != "POST" || r.URL.Path != "/v1.47/containers/create" || r.URL.Query().Get("name") != "my-web" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
		}
		body, _ := io.ReadAll(r.Body)
		var decoded struct {
			Image        string
			ExposedPorts map[string]json.RawMessage
			HostConfig   struct {
				PortBindings map[string][]struct {
					HostIp   string
					HostPort string
				}
			}
		}
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatal(err)
		}
		bindings := decoded.HostConfig.PortBindings["80/tcp"]
		if decoded.Image != "nginx:alpine" || len(bindings) != 1 || bindings[0].HostIp != "127.0.0.1" || bindings[0].HostPort != "8088" {
			t.Fatalf("invalid create body: %s", body)
		}
		if _, ok := decoded.ExposedPorts["80/tcp"]; !ok {
			t.Fatal("container port is missing")
		}
		return response(201, `{"Id":"created-id"}`), nil
	})}}
	id, err := engine.Create(context.Background(), CreateRequest{Name: "my-web", Image: "nginx:alpine", HostPort: 8088, ContainerPort: 80})
	if err != nil || id != "created-id" || requests != 2 {
		t.Fatalf("id=%s requests=%d err=%v", id, requests, err)
	}
}

func TestDockerErrorsAndNonForcedRemoval(t *testing.T) {
	engine := &EngineClient{client: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/version" {
			return response(200, `{"ApiVersion":"1.47"}`), nil
		}
		if r.Method != "DELETE" || r.URL.Path != "/v1.47/containers/abc" || r.URL.Query().Get("force") != "" || r.URL.Query().Get("v") != "" {
			t.Fatalf("unexpected delete: %s %s", r.Method, r.URL)
		}
		return response(409, `{"message":"container is running"}`), nil
	})}}
	err := engine.Action(context.Background(), "abc", "remove")
	apiErr, ok := err.(*Error)
	if !ok || apiErr.Status != 409 || apiErr.Message != "container is running" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSnapshotReadsAllResources(t *testing.T) {
	fixtures := map[string]string{"/version": `{"ApiVersion":"1.47"}`, "/v1.47/info": `{"Name":"host","NCPU":4}`, "/v1.47/containers/json": `[]`, "/v1.47/images/json": `[]`, "/v1.47/networks": `[]`, "/v1.47/volumes": `{"Volumes":null}`}
	engine := &EngineClient{client: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		body, ok := fixtures[r.URL.Path]
		if !ok {
			t.Fatalf("unexpected endpoint %s", r.URL.Path)
		}
		if r.URL.Path == "/v1.47/containers/json" && r.URL.Query().Get("all") != "true" {
			t.Fatal("stopped containers must be included")
		}
		return response(200, body), nil
	})}}
	s, err := engine.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.Mode != "docker" || s.Engine.Name != "host" || s.Volumes == nil || s.Containers == nil {
		t.Fatalf("invalid snapshot: %+v", s)
	}
}

func TestNewEngineRejectsOtherTransports(t *testing.T) {
	for _, host := range []string{"tcp://localhost:2375", "unix://", "https://example.com"} {
		if _, err := NewEngine(host); err == nil {
			t.Fatalf("accepted unsupported host %s", host)
		}
	}
	if _, err := NewEngine("unix:///var/run/docker.sock"); err != nil {
		t.Fatal(err)
	}
}
