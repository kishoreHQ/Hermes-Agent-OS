package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/kernel"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

func setup() *Server {
	reg := plugin.NewMemoryRegistry()
	_ = reg.Register(plugin.Manifest{
		APIVersion: "hermes.plugin/v1",
		Kind:       plugin.KindProvider,
		Metadata:   plugin.Metadata{ID: "provider.example.echo", Version: "0.0.1", Name: "Echo"},
		Spec:       map[string]any{"capabilities": []any{"coding"}},
	}, nil)
	_ = reg.Register(plugin.Manifest{
		APIVersion: "hermes.plugin/v1",
		Kind:       plugin.KindRuntime,
		Metadata:   plugin.Metadata{ID: "runtime.example.echo", Version: "0.0.1", Name: "Echo RT"},
	}, nil)
	return New(kernel.New(reg))
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
	s := setup()
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

func TestMissionLifecycleAndEvents(t *testing.T) {
	s := setup()
	h := s.Handler()

	// Create
	body := `{"goal":"build host api","requiredCapabilities":["coding","tools"]}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/missions", bytes.NewBufferString(body)))
	if rec.Code != 200 {
		t.Fatalf("create %d %s", rec.Code, rec.Body.String())
	}
	data, _ := decodeEnv(t, rec)
	row := data.(map[string]any)
	id, _ := row["id"].(string)
	if id == "" || row["state"] != "running" {
		t.Fatalf("%v", row)
	}

	// List
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/missions", nil))
	data, _ = decodeEnv(t, rec)
	list := data.([]any)
	if len(list) != 1 {
		t.Fatalf("%v", list)
	}

	// Get
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/missions/"+id, nil))
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}

	// Events JSON catch-up
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?since=0&format=json", nil)
	h.ServeHTTP(rec, req)
	data, _ = decodeEnv(t, rec)
	evs := data.([]any)
	if len(evs) < 2 {
		t.Fatalf("events %v", evs)
	}
	first := evs[0].(map[string]any)
	if first["seq"].(float64) != 1 {
		t.Fatalf("seq %v", first)
	}

	// Replay
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/replay/"+id, nil))
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}

	// Cancel
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/missions/"+id+"/cancel",
		bytes.NewBufferString(`{"reason":"test"}`)))
	data, _ = decodeEnv(t, rec)
	if data.(map[string]any)["state"] != "cancelled" {
		t.Fatalf("%v", data)
	}
}

func TestMissionRejectsModelNameCaps(t *testing.T) {
	s := setup()
	body := `{"goal":"x","requiredCapabilities":["gpt-4"]}`
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/missions", bytes.NewBufferString(body)))
	if rec.Code == 200 {
		t.Fatal("expected failure")
	}
}

func TestRegistry(t *testing.T) {
	s := setup()
	h := s.Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/registry/providers", nil))
	data, _ := decodeEnv(t, rec)
	list := data.([]any)
	if len(list) != 1 {
		t.Fatalf("%v", list)
	}
	if list[0].(map[string]any)["id"] != "provider.example.echo" {
		t.Fatalf("%v", list)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/registry/runtimes", nil))
	data, _ = decodeEnv(t, rec)
	if len(data.([]any)) != 1 {
		t.Fatalf("%v", data)
	}
}

func TestPluginsList(t *testing.T) {
	s := setup()
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/plugins", nil))
	data, _ := decodeEnv(t, rec)
	if len(data.([]any)) != 2 {
		t.Fatalf("%v", data)
	}
	_ = types.Capability("coding")
}
