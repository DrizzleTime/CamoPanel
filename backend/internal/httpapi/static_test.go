package httpapi

import (
	"os"
	"path/filepath"
	"testing"

	"camopanel/backend/internal/config"
	"camopanel/backend/internal/docker"
)

func TestStaticFilesAndSPAFallback(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{"index.html": "<html>CamoPanel</html>", "app.js": "console.log('camo')"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	router := New(config.Config{WebDir: root}, docker.NewDemo())
	for _, test := range []struct {
		path   string
		status int
		body   string
	}{
		{"/", 200, "<html>CamoPanel</html>"}, {"/containers", 200, "<html>CamoPanel</html>"}, {"/app.js", 200, "console.log('camo')"}, {"/missing.js", 404, ""}, {"/api/v1/missing", 404, ""},
	} {
		result := call(router, "GET", test.path, "", "", "")
		if result.Code != test.status || (test.body != "" && result.Body.String() != test.body) {
			t.Fatalf("path=%s status=%d body=%s", test.path, result.Code, result.Body.String())
		}
	}
}
