package openresty

import "testing"

func TestErrorAliases(t *testing.T) {
	if ErrUnavailable == nil {
		t.Fatal("expected ErrUnavailable")
	}
}
