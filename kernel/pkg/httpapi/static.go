package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func uiDistPath() string {
	if p := os.Getenv("HERMES_UI_DIST"); p != "" {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	candidates := []string{
		"mission-control/dist",
		filepath.Join("..", "mission-control", "dist"),
		filepath.Join("..", "..", "mission-control", "dist"),
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates,
			filepath.Join(filepath.Dir(exe), "mission-control", "dist"),
			filepath.Join(filepath.Dir(exe), "..", "mission-control", "dist"),
		)
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

// spaFileServer serves Mission Control static files with SPA fallback.
func spaFileServer(root string) http.Handler {
	fs := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") {
			http.NotFound(w, r)
			return
		}
		// path under root
		rel := strings.TrimPrefix(filepath.Clean(r.URL.Path), "/")
		full := filepath.Join(root, rel)
		if st, err := os.Stat(full); err == nil && !st.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(root, "index.html"))
	})
}
