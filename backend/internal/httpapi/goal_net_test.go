package httpapi

import (
	"encoding/json"
	"net"
	"testing"
)

type netListener = net.Listener

func netListen() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}

func jsonField(t *testing.T, raw []byte, key string) string {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	value, _ := body[key].(string)
	if value == "" {
		t.Fatalf("missing %s in %s", key, raw)
	}
	return value
}
