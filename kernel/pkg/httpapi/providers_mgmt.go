package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/providercfg"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

func (s *Server) registerProviderMgmtRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/provider-templates", s.apiProviderTemplates)
	mux.HandleFunc("/api/v1/provider-configs", s.apiProviderConfigs)
	mux.HandleFunc("/api/v1/provider-configs/", s.apiProviderConfigSub)
}

func (s *Server) apiProviderTemplates(w http.ResponseWriter, r *http.Request) {
	writeOK(w, providercfg.Templates())
}

func (s *Server) apiProviderConfigs(w http.ResponseWriter, r *http.Request) {
	if s.k.ProviderMgr == nil {
		writeErr(w, 503, "unavailable", "provider manager not ready", "")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.k.ProviderMgr.List())
	case http.MethodPost:
		body, _ := io.ReadAll(r.Body)
		var req providercfg.CreateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeErr(w, 400, "bad_json", err.Error(), "")
			return
		}
		cfg, err := s.k.ProviderMgr.Create(r.Context(), req)
		if err != nil {
			writeErr(w, 400, "create_failed", err.Error(), "Check template id / baseUrl")
			return
		}
		// Never return apiKey
		writeOK(w, cfg)
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}

func (s *Server) apiProviderConfigSub(w http.ResponseWriter, r *http.Request) {
	if s.k.ProviderMgr == nil {
		writeErr(w, 503, "unavailable", "provider manager not ready", "")
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/provider-configs/"), "/")
	if id == "" {
		writeErr(w, 400, "bad_request", "missing id", "")
		return
	}
	pid := types.PluginID(id)
	switch r.Method {
	case http.MethodGet:
		cfg, ok := s.k.ProviderMgr.Get(pid)
		if !ok {
			writeErr(w, 404, "not_found", "unknown provider config", "")
			return
		}
		writeOK(w, cfg)
	case http.MethodPut, http.MethodPatch:
		body, _ := io.ReadAll(r.Body)
		var req providercfg.CreateRequest
		_ = json.Unmarshal(body, &req)
		cfg, err := s.k.ProviderMgr.Update(r.Context(), pid, req)
		if err != nil {
			writeErr(w, 400, "update_failed", err.Error(), "")
			return
		}
		writeOK(w, cfg)
	case http.MethodDelete:
		if err := s.k.ProviderMgr.Delete(pid); err != nil {
			writeErr(w, 400, "delete_failed", err.Error(), "Only UI-managed providers can be deleted")
			return
		}
		writeOK(w, map[string]any{"deleted": id})
	default:
		writeErr(w, 405, "method", "GET, PUT, DELETE", "")
	}
}
