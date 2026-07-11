// Package httpapi exposes the Hermes Host Interface over HTTP/WS (INV-11).
// Contract: /api/v1/* with {data,error} envelope — aligned with AESP host UI patterns.
package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/kernel"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/memorystore"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

const Version = "hermesd-host/0.9.0"

// Server is the Host HTTP surface.
type Server struct {
	k *kernel.Kernel
}

func New(k *kernel.Kernel) *Server {
	return &Server{k: k}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthPlain)
	mux.HandleFunc("/api/v1/health", s.apiHealth)
	mux.HandleFunc("/api/v1/missions", s.apiMissions)
	mux.HandleFunc("/api/v1/missions/", s.apiMissionSub)
	mux.HandleFunc("/api/v1/registry/", s.apiRegistry)
	mux.HandleFunc("/api/v1/replay/", s.apiReplay)
	mux.HandleFunc("/api/v1/events", s.apiEvents) // GET catch-up JSON or WS upgrade
	mux.HandleFunc("/api/v1/plugins", s.apiPlugins)
	mux.HandleFunc("/api/v1/memory/search", s.apiMemorySearch)
	mux.HandleFunc("/api/v1/credentials", s.apiCredentials)
	mux.HandleFunc("/api/v1/security/posture", s.apiSecurityPosture)
	mux.HandleFunc("/api/v1/policies", s.apiPolicies)
	s.registerDeckRoutes(mux)
	s.registerPlatformRoutes(mux)
	s.registerProviderMgmtRoutes(mux)

	// Mission Control SPA when mission-control/dist exists (H3 / GAP-UI-002 parity)
	if dist := uiDistPath(); dist != "" {
		mux.Handle("/", spaFileServer(dist))
	}

	// Auth (optional) → CORS
	return withCORS(withAuth(mux))
}

// ——— envelope ———

func writeEnv(w http.ResponseWriter, status int, data any, errObj map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data, "error": errObj})
}

func writeOK(w http.ResponseWriter, data any) { writeEnv(w, 200, data, nil) }

func writeErr(w http.ResponseWriter, status int, code, msg, remediation string) {
	writeEnv(w, status, nil, map[string]any{
		"code": code, "message": msg, "remediation": remediation,
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When API token is required, echo Origin only if present; still allow * for local SPA.
		// Production should put UI behind same origin or set HERMES_CORS_ORIGIN.
		origin := os.Getenv("HERMES_CORS_ORIGIN")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Hermes-Token")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ——— handlers ———

func (s *Server) healthPlain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) apiHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.k.Health(r.Context()); err != nil {
		writeErr(w, 500, "unhealthy", err.Error(), "Restart hermesd.")
		return
	}
	pol := s.k.Policy()
	writeOK(w, map[string]any{
		"status":  "ok",
		"profile": "host-neutral",
		"version": Version,
		"product": "Hermes-Agent-OS",
		"seq":     s.k.Bus().Seq(),
		"policyId": pol.ID,
		"modes":    []string{"full", "assist", "observe"},
	})
}

func (s *Server) apiMissions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.k.ListMissions(r.Context(), r.URL.Query().Get("state"))
		if err != nil {
			writeErr(w, 500, "list_failed", err.Error(), "Retry.")
			return
		}
		out := make([]map[string]any, 0, len(list))
		for _, m := range list {
			out = append(out, missionJSON(m))
		}
		writeOK(w, out)
	case http.MethodPost:
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Name                 string            `json:"name"`
			Goal                 string            `json:"goal"`
			Mode                 string            `json:"mode"`
			RequiredCapabilities []string          `json:"requiredCapabilities"`
			Labels               map[string]string `json:"labels"`
			// Multi-provider selection
			PreferProvider  string   `json:"preferProvider"`
			RequireProvider string   `json:"requireProvider"`
			PreferModel     string   `json:"preferModel"`
			Model           string   `json:"model"` // alias
			Providers       []string `json:"providers"`
			Failover        *bool    `json:"failover"`
		}
		if err := json.Unmarshal(body, &req); err != nil && len(body) > 0 {
			writeErr(w, 400, "bad_json", err.Error(), "Send valid JSON.")
			return
		}
		if req.Goal == "" {
			req.Goal = req.Name
		}
		if req.Goal == "" {
			writeErr(w, 400, "bad_request", "goal required", "Provide goal or name.")
			return
		}
		if len(req.RequiredCapabilities) == 0 {
			writeErr(w, 400, "bad_request", "requiredCapabilities required",
				"Declare capabilities (e.g. coding, tools) — never model names.")
			return
		}
		caps := make([]types.Capability, 0, len(req.RequiredCapabilities))
		for _, c := range req.RequiredCapabilities {
			caps = append(caps, types.Capability(c))
		}
		labels := req.Labels
		if req.Mode != "" {
			if labels == nil {
				labels = map[string]string{}
			}
			if labels["security.mode"] == "" {
				labels["security.mode"] = req.Mode
			}
		}
		model := req.PreferModel
		if model == "" {
			model = req.Model
		}
		var provs []types.PluginID
		for _, p := range req.Providers {
			provs = append(provs, types.PluginID(p))
		}
		id, err := s.k.SubmitMission(r.Context(), host.Mission{
			Name: req.Name, Goal: req.Goal, RequiredCaps: caps, Labels: labels,
			Mode:            types.AgentMode(req.Mode),
			PreferProvider:  types.PluginID(req.PreferProvider),
			RequireProvider: types.PluginID(req.RequireProvider),
			PreferModel:     model,
			Providers:       provs,
			Failover:        req.Failover,
		})
		if err != nil {
			writeErr(w, 400, "submit_failed", err.Error(), "Fix capabilities or goal.")
			return
		}
		m, err := s.k.GetMission(r.Context(), id)
		if err != nil {
			writeErr(w, 500, "get_failed", err.Error(), "Retry.")
			return
		}
		writeOK(w, missionJSON(m))
	default:
		writeErr(w, 405, "method", "GET or POST", "Use GET/POST /api/v1/missions")
	}
}

func (s *Server) apiMissionSub(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/missions/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeErr(w, 400, "bad_request", "missing id", "Provide mission id")
		return
	}
	id := types.MissionID(parts[0])
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeErr(w, 405, "method", "GET", "GET /api/v1/missions/:id")
			return
		}
		m, err := s.k.GetMission(r.Context(), id)
		if err != nil {
			writeErr(w, 404, "not_found", "Mission not found", "Check mission id.")
			return
		}
		writeOK(w, missionJSON(m))
		return
	}
	switch parts[1] {
	case "cancel":
		if r.Method != http.MethodPost {
			writeErr(w, 405, "method", "POST", "POST /api/v1/missions/:id/cancel")
			return
		}
		reason := "host-cancel"
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Reason string `json:"reason"`
		}
		_ = json.Unmarshal(body, &req)
		if req.Reason != "" {
			reason = req.Reason
		}
		if err := s.k.CancelMission(r.Context(), id, reason); err != nil {
			writeErr(w, 404, "not_found", err.Error(), "Check mission id.")
			return
		}
		writeOK(w, map[string]any{"id": string(id), "state": string(host.StateCancelled)})
	case "events":
		// Per-mission event list (HTTP)
		evs, err := s.k.Replay(r.Context(), id)
		if err != nil {
			writeErr(w, 500, "replay_failed", err.Error(), "Retry.")
			return
		}
		writeOK(w, eventsJSON(evs))
	default:
		writeErr(w, 404, "not_found", "unknown subpath", "Use cancel|events")
	}
}

func (s *Server) apiRegistry(w http.ResponseWriter, r *http.Request) {
	kind := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/registry/"), "/")
	var items []map[string]any
	switch kind {
	case "providers":
		for _, m := range s.k.Plugins().List(plugin.KindProvider) {
			items = append(items, manifestJSON(m, "provider"))
		}
	case "runtimes":
		for _, m := range s.k.Plugins().List(plugin.KindRuntime) {
			items = append(items, manifestJSON(m, "runtime"))
		}
	case "tools":
		if s.k.Tools() != nil {
			for _, t := range s.k.Tools().List() {
				items = append(items, map[string]any{
					"id": t.ID, "name": t.Name, "kind": "tool", "enabled": t.Enabled,
					"description": t.Description,
				})
			}
		}
		for _, m := range s.k.Plugins().List(plugin.KindTool) {
			items = append(items, manifestJSON(m, "tool"))
		}
	case "agents":
		if s.k.Agents != nil {
			for _, a := range s.k.Agents.List() {
				caps := make([]string, 0, len(a.Capabilities))
				for _, c := range a.Capabilities {
					caps = append(caps, string(c))
				}
				items = append(items, map[string]any{
					"id": string(a.ID), "name": a.Name, "kind": "agent",
					"roles": a.Roles, "capabilities": caps, "enabled": a.Enabled,
				})
			}
		}
	default:
		writeErr(w, 404, "not_found", "unknown registry kind", "providers|runtimes|tools|agents")
		return
	}
	if items == nil {
		items = []map[string]any{}
	}
	writeOK(w, items)
}

func (s *Server) apiPlugins(w http.ResponseWriter, r *http.Request) {
	kind := plugin.Kind(r.URL.Query().Get("kind"))
	list := s.k.Plugins().List(kind)
	out := make([]map[string]any, 0, len(list))
	for _, m := range list {
		out = append(out, manifestJSON(m, string(m.Kind)))
	}
	writeOK(w, out)
}

func (s *Server) apiReplay(w http.ResponseWriter, r *http.Request) {
	id := types.MissionID(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/replay/"), "/"))
	if id == "" {
		writeErr(w, 400, "bad_request", "mission id required", "GET /api/v1/replay/:id")
		return
	}
	evs, err := s.k.Replay(r.Context(), id)
	if err != nil {
		writeErr(w, 500, "replay_failed", err.Error(), "Retry.")
		return
	}
	writeOK(w, map[string]any{
		"missionId": string(id),
		"events":    eventsJSON(evs),
	})
}

func (s *Server) apiSecurityPosture(w http.ResponseWriter, r *http.Request) {
	pol := s.k.Policy()
	writeOK(w, map[string]any{
		"modes": []map[string]any{
			{"id": "full", "description": "Autonomous execution within policy budgets"},
			{"id": "assist", "description": "External actions require human approval"},
			{"id": "observe", "description": "Route and journal only; no runtime execute"},
		},
		"credentialBroker": "handles-only",
		"pluginSigning": map[string]any{
			"hmac":            "HERMES_PLUGIN_HMAC_KEY",
			"requireSigned":   "HERMES_REQUIRE_SIGNED_PLUGINS",
			"label":           "hermes.signature",
		},
		"policy": map[string]any{
			"id": pol.ID, "defaultMode": pol.DefaultMode,
			"maxSteps": pol.MaxSteps, "maxCostUsd": pol.MaxCostUSD,
			"minSandboxTier": pol.MinSandboxTier,
		},
		"sandboxTiers": []string{"process-pty", "container", "micro-vm"},
	})
}

func (s *Server) apiPolicies(w http.ResponseWriter, r *http.Request) {
	pol := s.k.Policy()
	writeOK(w, []map[string]any{{
		"id": pol.ID, "defaultMode": string(pol.DefaultMode),
		"maxSteps": pol.MaxSteps, "maxCostUsd": pol.MaxCostUSD,
		"minSandboxTier": pol.MinSandboxTier, "preferLocal": pol.PreferLocal,
	}})
}

func (s *Server) apiMemorySearch(w http.ResponseWriter, r *http.Request) {
	q := memorystore.Query{
		Text:      r.URL.Query().Get("q"),
		MissionID: types.MissionID(r.URL.Query().Get("mission")),
		Kind:      memorystore.Kind(r.URL.Query().Get("kind")),
		Limit:     50,
	}
	hits, err := s.k.Memory().Search(r.Context(), q)
	if err != nil {
		writeErr(w, 500, "search_failed", err.Error(), "Retry.")
		return
	}
	writeOK(w, memorystore.AsMaps(hits))
}

func (s *Server) apiCredentials(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Metadata only — never secrets (INV-07)
		list, err := s.k.Creds().List(r.Context())
		if err != nil {
			writeErr(w, 500, "list_failed", err.Error(), "Retry.")
			return
		}
		out := make([]map[string]any, 0, len(list))
		for _, rec := range list {
			out = append(out, map[string]any{
				"handle":    string(rec.Handle),
				"scope":     rec.Scope,
				"label":     rec.Label,
				"pluginId":  string(rec.PluginID),
				"createdAt": rec.CreatedAt.UTC().Format(time.RFC3339Nano),
			})
		}
		writeOK(w, out)
	case http.MethodPost:
		// Store secret, return handle only
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Scope    string `json:"scope"`
			Label    string `json:"label"`
			PluginID string `json:"pluginId"`
			Secret   string `json:"secret"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeErr(w, 400, "bad_json", err.Error(), "")
			return
		}
		if req.Secret == "" {
			writeErr(w, 400, "bad_request", "secret required", "Send API key as secret; never log it")
			return
		}
		if req.Label == "" {
			req.Label = "api-key"
		}
		if req.Scope == "" {
			req.Scope = req.PluginID
		}
		h, err := s.k.Creds().Put(r.Context(), req.Scope, req.Label, types.PluginID(req.PluginID), req.Secret)
		if err != nil {
			writeErr(w, 400, "put_failed", err.Error(), "")
			return
		}
		// Never echo secret
		writeOK(w, map[string]any{
			"handle":   string(h),
			"scope":    req.Scope,
			"label":    req.Label,
			"pluginId": req.PluginID,
		})
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}


func missionJSON(m host.Mission) map[string]any {
	caps := make([]string, 0, len(m.RequiredCaps))
	for _, c := range m.RequiredCaps {
		caps = append(caps, string(c))
	}
	name := m.Name
	if name == "" {
		name = m.Goal
	}
	return map[string]any{
		"id":                   string(m.ID),
		"name":                 name,
		"goal":                 m.Goal,
		"state":                string(m.State),
		"mode":                 string(m.Mode),
		"requiredCapabilities": caps,
		"labels":               m.Labels,
		"costUsd":              m.CostUSD,
		"output":               m.Output,
		"providerId":           string(m.ProviderID),
		"runtimeId":            string(m.RuntimeID),
		"modelId":              m.ModelID,
		"routeReason":          m.RouteReason,
		"securityNote":         m.SecurityNote,
		"createdAt":            m.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt":            m.UpdatedAt.UTC().Format(time.RFC3339Nano),
		"cancelReason":         m.CancelReason,
	}
}

func eventsJSON(evs []host.Event) []map[string]any {
	out := make([]map[string]any, 0, len(evs))
	for _, e := range evs {
		out = append(out, map[string]any{
			"seq":       e.Seq,
			"type":      e.Type,
			"missionId": string(e.MissionID),
			"ts":        e.TS.UTC().Format(time.RFC3339Nano),
			"data":      e.Data,
		})
	}
	return out
}

func manifestJSON(m plugin.Manifest, kind string) map[string]any {
	return map[string]any{
		"id":      string(m.Metadata.ID),
		"name":    m.Metadata.Name,
		"version": m.Metadata.Version,
		"kind":    kind,
		"spec":    m.Spec,
		"labels":  m.Labels,
		"enabled": true,
	}
}
