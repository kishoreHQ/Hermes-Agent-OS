package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/bootstrap"
)

func setup(t *testing.T) *Server {
	t.Helper()
	res, err := bootstrap.New(bootstrap.Options{SeedBuiltins: true, PluginRoots: []string{"/no-plugins"}})
	if err != nil {
		t.Fatal(err)
	}
	return New(res.Kernel)
}

func decodeEnv(t *testing.T, rec *httptest.ResponseRecorder) (any, map[string]any) {
	t.Helper()
	var env struct {
		Data  any            `json:"data"`
		Error map[string]any `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	return env.Data, env.Error
}

func TestHealth(t *testing.T) {
	s := setup(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	data, errObj := decodeEnv(t, rec)
	if errObj != nil {
		t.Fatalf("%v", errObj)
	}
	m := data.(map[string]any)
	if m["status"] != "ok" {
		t.Fatalf("%v", m)
	}
}

func TestMissionLifecycleRouteAndMemory(t *testing.T) {
	s := setup(t)
	h := s.Handler()

	body := `{"goal":"build host api","requiredCapabilities":["coding","tools"]}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/missions", bytes.NewBufferString(body)))
	if rec.Code != 200 {
		t.Fatalf("create %d %s", rec.Code, rec.Body.String())
	}
	data, _ := decodeEnv(t, rec)
	row := data.(map[string]any)
	id, _ := row["id"].(string)
	if id == "" {
		t.Fatalf("%v", row)
	}
	if row["state"] != "succeeded" {
		t.Fatalf("want succeeded got %v output=%v", row["state"], row["output"])
	}
	if row["providerId"] != "provider.example.echo" {
		t.Fatalf("provider %v", row["providerId"])
	}
	// Default prefer agent-loop; echo still valid if preferred explicitly
	rt, _ := row["runtimeId"].(string)
	if rt != "runtime.agent.loop" && rt != "runtime.example.echo" {
		t.Fatalf("runtime %v", row["runtimeId"])
	}

	// Events include route.decided
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?since=0&format=json", nil)
	h.ServeHTTP(rec, req)
	data, _ = decodeEnv(t, rec)
	evs := data.([]any)
	if len(evs) < 3 {
		t.Fatalf("events %v", evs)
	}

	// Memory search
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/memory/search?mission="+id, nil))
	data, _ = decodeEnv(t, rec)
	if len(data.([]any)) < 1 {
		t.Fatalf("memory %v", data)
	}

	// Credentials handles only
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/credentials", nil))
	data, _ = decodeEnv(t, rec)
	creds := data.([]any)
	if len(creds) < 1 {
		t.Fatal("expected credential handle")
	}
	c0 := creds[0].(map[string]any)
	if c0["handle"] == nil || c0["secret"] != nil {
		// secret field must never appear
		if _, ok := c0["secret"]; ok {
			t.Fatal("secret leaked")
		}
	}

	// Cancel after success still ok
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/missions/"+id+"/cancel",
		bytes.NewBufferString(`{"reason":"test"}`)))
	data, _ = decodeEnv(t, rec)
	if data.(map[string]any)["state"] != "cancelled" {
		t.Fatalf("%v", data)
	}
}

func TestMissionRejectsModelNameCaps(t *testing.T) {
	s := setup(t)
	body := `{"goal":"x","requiredCapabilities":["gpt-4"]}`
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/missions", bytes.NewBufferString(body)))
	if rec.Code == 200 {
		t.Fatal("expected failure")
	}
}

func TestRegistry(t *testing.T) {
	s := setup(t)
	h := s.Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/registry/providers", nil))
	data, _ := decodeEnv(t, rec)
	list := data.([]any)
	if len(list) < 2 {
		t.Fatalf("want ≥2 providers, got %v", list)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/registry/runtimes", nil))
	data, _ = decodeEnv(t, rec)
	if len(data.([]any)) < 1 {
		t.Fatalf("%v", data)
	}
}
