package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/bootstrap"
)

func TestAuthOptionalOpen(t *testing.T) {
	os.Unsetenv("HERMES_API_TOKEN")
	res, err := bootstrap.New(bootstrap.Options{SeedBuiltins: true, PluginRoots: []string{"/none"}})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	New(res.Kernel).Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil))
	if rec.Code != 200 {
		t.Fatalf("%d", rec.Code)
	}
}

func TestAuthRequired(t *testing.T) {
	os.Setenv("HERMES_API_TOKEN", "secret-token")
	defer os.Unsetenv("HERMES_API_TOKEN")
	res, err := bootstrap.New(bootstrap.Options{SeedBuiltins: true, PluginRoots: []string{"/none"}})
	if err != nil {
		t.Fatal(err)
	}
	h := New(res.Kernel).Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil))
	if rec.Code != 401 {
		t.Fatalf("want 401 got %d", rec.Code)
	}
	// health open
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if rec.Code != 200 {
		t.Fatalf("health %d", rec.Code)
	}
	// bearer ok
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("auth %d %s", rec.Code, rec.Body.String())
	}
}
