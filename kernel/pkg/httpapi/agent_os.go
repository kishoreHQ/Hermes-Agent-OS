package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/mcpclient"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/research"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/scheduler"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/skills"
)

func (s *Server) registerAgentOSRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/chat/sessions", s.apiChatSessions)
	mux.HandleFunc("/api/v1/chat/sessions/", s.apiChatSessionSub)
	mux.HandleFunc("/api/v1/mcp/servers", s.apiMCPServers)
	mux.HandleFunc("/api/v1/mcp/servers/", s.apiMCPServerSub)
	mux.HandleFunc("/api/v1/skills", s.apiSkills)
	mux.HandleFunc("/api/v1/research", s.apiResearch)
	mux.HandleFunc("/api/v1/approvals", s.apiApprovals)
	mux.HandleFunc("/api/v1/approvals/", s.apiApprovalSub)
	mux.HandleFunc("/api/v1/jobs", s.apiJobs)
	mux.HandleFunc("/api/v1/jobs/", s.apiJobSub)
	mux.HandleFunc("/api/v1/stream/events", s.apiStreamEvents)
}

// —— Chat ——

func (s *Server) apiChatSessions(w http.ResponseWriter, r *http.Request) {
	if s.k.Chat == nil {
		writeErr(w, 503, "unavailable", "chat not ready", "")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.k.Chat.List())
	case http.MethodPost:
		var req struct {
			Title string `json:"title"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		writeOK(w, s.k.Chat.Create(req.Title))
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}

func (s *Server) apiChatSessionSub(w http.ResponseWriter, r *http.Request) {
	if s.k.Chat == nil {
		writeErr(w, 503, "unavailable", "chat not ready", "")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/chat/sessions/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	id := parts[0]
	if id == "" {
		writeErr(w, 400, "bad_request", "missing id", "")
		return
	}
	// POST .../messages
	if len(parts) >= 2 && parts[1] == "messages" && r.Method == http.MethodPost {
		var req struct {
			Text           string   `json:"text"`
			PreferProvider string   `json:"preferProvider"`
			PreferModel    string   `json:"preferModel"`
			SkillIDs       []string `json:"skillIds"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		sess, err := s.k.Chat.Send(r.Context(), id, req.Text, req.PreferProvider, req.PreferModel, req.SkillIDs)
		if err != nil {
			writeErr(w, 400, "chat_failed", err.Error(), "")
			return
		}
		writeOK(w, sess)
		return
	}
	if r.Method == http.MethodGet {
		sess, ok := s.k.Chat.Get(id)
		if !ok {
			writeErr(w, 404, "not_found", "session", "")
			return
		}
		writeOK(w, sess)
		return
	}
	writeErr(w, 405, "method", "GET or POST messages", "")
}

// —— MCP ——

func (s *Server) apiMCPServers(w http.ResponseWriter, r *http.Request) {
	if s.k.MCPClient == nil {
		writeErr(w, 503, "unavailable", "mcp client not ready", "")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeOK(w, map[string]any{
			"servers":  s.k.MCPClient.List(),
			"statuses": s.k.MCPClient.Statuses(),
		})
	case http.MethodPost:
		var cfg mcpclient.ServerConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeErr(w, 400, "bad_json", err.Error(), "")
			return
		}
		cfg.Enabled = true
		if err := s.k.MCPClient.Upsert(cfg); err != nil {
			writeErr(w, 400, "upsert_failed", err.Error(), "")
			return
		}
		if err := s.k.MCPClient.Connect(r.Context(), cfg.ID); err != nil {
			// still return config; status will show error
			writeOK(w, map[string]any{"config": cfg, "connectError": err.Error()})
			return
		}
		writeOK(w, map[string]any{"config": cfg, "statuses": s.k.MCPClient.Statuses()})
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}

func (s *Server) apiMCPServerSub(w http.ResponseWriter, r *http.Request) {
	if s.k.MCPClient == nil {
		writeErr(w, 503, "unavailable", "mcp client not ready", "")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/mcp/servers/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	id := parts[0]
	if id == "" {
		writeErr(w, 400, "bad_request", "missing id", "")
		return
	}
	if len(parts) >= 2 && parts[1] == "connect" && r.Method == http.MethodPost {
		if err := s.k.MCPClient.Connect(r.Context(), id); err != nil {
			writeErr(w, 400, "connect_failed", err.Error(), "")
			return
		}
		writeOK(w, s.k.MCPClient.Statuses())
		return
	}
	if len(parts) >= 2 && parts[1] == "disconnect" && r.Method == http.MethodPost {
		_ = s.k.MCPClient.Disconnect(id)
		writeOK(w, map[string]any{"disconnected": id})
		return
	}
	if r.Method == http.MethodDelete {
		_ = s.k.MCPClient.Delete(id)
		writeOK(w, map[string]any{"deleted": id})
		return
	}
	writeErr(w, 405, "method", "connect|disconnect|DELETE", "")
}

// —— Skills ——

func (s *Server) apiSkills(w http.ResponseWriter, r *http.Request) {
	if s.k.Skills == nil {
		writeErr(w, 503, "unavailable", "skills not ready", "")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.k.Skills.List())
	case http.MethodPost:
		var sk skills.Skill
		if err := json.NewDecoder(r.Body).Decode(&sk); err != nil {
			writeErr(w, 400, "bad_json", err.Error(), "")
			return
		}
		s.k.Skills.Put(sk)
		writeOK(w, sk)
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}

// —— Research ——

func (s *Server) apiResearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, 405, "method", "POST", "")
		return
	}
	var req research.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "bad_json", err.Error(), "")
		return
	}
	m, err := research.Run(r.Context(), s.k, req)
	if err != nil {
		writeErr(w, 400, "research_failed", err.Error(), "")
		return
	}
	writeOK(w, m)
}

// —— Approvals ——

func (s *Server) apiApprovals(w http.ResponseWriter, r *http.Request) {
	if s.k.Approvals == nil {
		writeErr(w, 503, "unavailable", "approvals not ready", "")
		return
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	writeOK(w, s.k.Approvals.List(status))
}

func (s *Server) apiApprovalSub(w http.ResponseWriter, r *http.Request) {
	if s.k.Approvals == nil {
		writeErr(w, 503, "unavailable", "approvals not ready", "")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/approvals/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	id := parts[0]
	if r.Method == http.MethodPost && len(parts) >= 2 {
		// POST /approvals/{id}/approve|deny
		decision := parts[1]
		req, err := s.k.Approvals.Resolve(id, decision)
		if err != nil {
			writeErr(w, 400, "resolve_failed", err.Error(), "")
			return
		}
		writeOK(w, req)
		return
	}
	writeErr(w, 405, "method", "POST /approvals/{id}/approve|deny", "")
}

// —— Jobs (scheduler) ——

func (s *Server) apiJobs(w http.ResponseWriter, r *http.Request) {
	if s.k.Scheduler == nil {
		writeErr(w, 503, "unavailable", "scheduler not ready", "")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.k.Scheduler.List())
	case http.MethodPost:
		var j scheduler.Job
		if err := json.NewDecoder(r.Body).Decode(&j); err != nil {
			writeErr(w, 400, "bad_json", err.Error(), "")
			return
		}
		writeOK(w, s.k.Scheduler.Upsert(j))
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}

// apiStreamEvents is SSE of the event bus (Odysseus-style live progress).
func (s *Server) apiStreamEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, 500, "no_flush", "streaming unsupported", "")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	filter := r.URL.Query().Get("mission")
	ch, err := s.k.Bus().Subscribe(r.Context(), filter)
	if err != nil {
		writeErr(w, 500, "subscribe_failed", err.Error(), "")
		return
	}
	// catch-up
	if since := r.URL.Query().Get("since"); since != "" {
		// best-effort: full since via EventsSince not needed here
	}
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
			_, _ = w.Write([]byte(": ping\n\n"))
			flusher.Flush()
		case ev, ok := <-ch:
			if !ok {
				return
			}
			b, _ := json.Marshal(ev)
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(b)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}

func (s *Server) apiJobSub(w http.ResponseWriter, r *http.Request) {
	if s.k.Scheduler == nil {
		writeErr(w, 503, "unavailable", "scheduler not ready", "")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	id := parts[0]
	if r.Method == http.MethodDelete {
		s.k.Scheduler.Delete(id)
		writeOK(w, map[string]any{"deleted": id})
		return
	}
	if r.Method == http.MethodPost && len(parts) >= 2 && parts[1] == "run" {
		mid, err := s.k.Scheduler.RunNow(id)
		if err != nil {
			writeErr(w, 400, "run_failed", err.Error(), "")
			return
		}
		writeOK(w, map[string]any{"missionId": mid})
		return
	}
	writeErr(w, 405, "method", "DELETE or POST run", "")
}
