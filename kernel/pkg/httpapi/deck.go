package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/deck"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

func (s *Server) registerDeckRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/connections/probe", s.apiConnProbe)
	mux.HandleFunc("/api/v1/connections", s.apiConnections)
	mux.HandleFunc("/api/v1/sessions", s.apiSessions)
	mux.HandleFunc("/api/v1/sessions/", s.apiSessionSub)
	mux.HandleFunc("/api/v1/boards", s.apiBoards)
	mux.HandleFunc("/api/v1/tasks", s.apiTasks)
	mux.HandleFunc("/api/v1/tasks/", s.apiTaskSub)
	mux.HandleFunc("/api/v1/routines", s.apiRoutines)
	mux.HandleFunc("/api/v1/routines/", s.apiRoutineSub)
	mux.HandleFunc("/api/v1/tools", s.apiTools)
	mux.HandleFunc("/api/v1/tools/invocations", s.apiToolInvocations)
	mux.HandleFunc("/api/v1/tools/", s.apiToolInvoke)
}

func (s *Server) apiConnProbe(w http.ResponseWriter, r *http.Request) {
	if s.k.Connections == nil {
		writeErr(w, 503, "unavailable", "connections not ready", "Restart hermesd")
		return
	}
	writeOK(w, s.k.Connections.Probe(r.Context()))
}

func (s *Server) apiConnections(w http.ResponseWriter, r *http.Request) {
	if s.k.Connections == nil {
		writeErr(w, 503, "unavailable", "connections not ready", "")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.k.Connections.List())
	case http.MethodPost:
		body, _ := io.ReadAll(r.Body)
		var req deck.RegisterRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeErr(w, 400, "bad_request", err.Error(), "Send JSON")
			return
		}
		c, err := s.k.Connections.Register(r.Context(), req)
		if err != nil && c != nil {
			writeEnv(w, 422, c, map[string]any{"code": "handshake_failed", "message": err.Error()})
			return
		}
		if err != nil {
			writeErr(w, 422, "handshake_failed", err.Error(), "Check plugin health")
			return
		}
		writeOK(w, c)
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}

func (s *Server) apiSessions(w http.ResponseWriter, r *http.Request) {
	if s.k.Sessions == nil {
		writeErr(w, 503, "unavailable", "sessions not ready", "")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.k.Sessions.List())
	case http.MethodPost:
		body, _ := io.ReadAll(r.Body)
		var req deck.CreateSessionRequest
		_ = json.Unmarshal(body, &req)
		sess, err := s.k.Sessions.Create(r.Context(), req)
		if err != nil {
			writeErr(w, 500, "session_error", err.Error(), "")
			return
		}
		writeOK(w, sess)
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}

func (s *Server) apiSessionSub(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeErr(w, 400, "bad_request", "missing id", "")
		return
	}
	id := parts[0]
	if len(parts) == 1 {
		sess, ok := s.k.Sessions.Get(id)
		if !ok {
			writeErr(w, 404, "not_found", "session not found", "")
			return
		}
		writeOK(w, sess)
		return
	}
	switch parts[1] {
	case "message":
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal(body, &req)
		sess, err := s.k.Sessions.Message(r.Context(), id, req.Text)
		if err != nil && sess == nil {
			writeErr(w, 400, "message_failed", err.Error(), "")
			return
		}
		writeOK(w, sess)
	case "stop":
		if err := s.k.Sessions.Stop(id); err != nil {
			writeErr(w, 404, "not_found", err.Error(), "")
			return
		}
		writeOK(w, map[string]any{"id": id, "status": "stopped"})
	default:
		writeErr(w, 404, "not_found", "use message|stop", "")
	}
}

func (s *Server) apiBoards(w http.ResponseWriter, r *http.Request) {
	if s.k.Board == nil {
		writeErr(w, 503, "unavailable", "board not ready", "")
		return
	}
	writeOK(w, s.k.Board.ListBoards())
}

func (s *Server) apiTasks(w http.ResponseWriter, r *http.Request) {
	if s.k.Board == nil {
		writeErr(w, 503, "unavailable", "board not ready", "")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.k.Board.ListTasks())
	case http.MethodPost:
		body, _ := io.ReadAll(r.Body)
		var req deck.CreateTaskRequest
		_ = json.Unmarshal(body, &req)
		t, err := s.k.Board.CreateTask(req)
		if err != nil {
			writeErr(w, 400, "bad_request", err.Error(), "")
			return
		}
		writeOK(w, t)
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}

func (s *Server) apiTaskSub(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		writeErr(w, 404, "not_found", "use /tasks/:id/claim|/move", "")
		return
	}
	id := parts[0]
	switch parts[1] {
	case "claim":
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Assignee string `json:"assignee"`
		}
		_ = json.Unmarshal(body, &req)
		if req.Assignee == "" {
			req.Assignee = "operator"
		}
		t, err := s.k.Board.ClaimTask(id, req.Assignee)
		if err != nil {
			writeErr(w, 404, "not_found", err.Error(), "")
			return
		}
		writeOK(w, t)
	case "move":
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Column string `json:"column"`
		}
		_ = json.Unmarshal(body, &req)
		t, err := s.k.Board.MoveTask(id, deck.Column(req.Column))
		if err != nil {
			writeErr(w, 400, "bad_request", err.Error(), "")
			return
		}
		writeOK(w, t)
	default:
		writeErr(w, 404, "not_found", "claim|move", "")
	}
}

func (s *Server) apiRoutines(w http.ResponseWriter, r *http.Request) {
	if s.k.Routines == nil {
		writeErr(w, 503, "unavailable", "routines not ready", "")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.k.Routines.List())
	case http.MethodPost:
		body, _ := io.ReadAll(r.Body)
		var req deck.CreateRoutineRequest
		_ = json.Unmarshal(body, &req)
		rt, err := s.k.Routines.Create(req)
		if err != nil {
			writeErr(w, 400, "bad_request", err.Error(), "")
			return
		}
		writeOK(w, rt)
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}

func (s *Server) apiRoutineSub(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/routines/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		writeErr(w, 404, "not_found", "use /routines/:id/fire|/pause", "")
		return
	}
	id := parts[0]
	switch parts[1] {
	case "fire":
		rt, err := s.k.Routines.Fire(r.Context(), id)
		if err != nil && rt == nil {
			writeErr(w, 400, "fire_failed", err.Error(), "")
			return
		}
		writeOK(w, rt)
	case "pause":
		_ = s.k.Routines.SetPaused(id, true)
		writeOK(w, map[string]any{"id": id, "paused": true})
	case "resume":
		_ = s.k.Routines.SetPaused(id, false)
		writeOK(w, map[string]any{"id": id, "paused": false})
	default:
		writeErr(w, 404, "not_found", "fire|pause|resume", "")
	}
}

func (s *Server) apiTools(w http.ResponseWriter, r *http.Request) {
	if s.k.Tools() == nil {
		writeOK(w, []any{})
		return
	}
	writeOK(w, s.k.Tools().List())
}

func (s *Server) apiToolInvocations(w http.ResponseWriter, r *http.Request) {
	if s.k.Tools() == nil {
		writeOK(w, []any{})
		return
	}
	writeOK(w, s.k.Tools().Invocations(50))
}

func (s *Server) apiToolInvoke(w http.ResponseWriter, r *http.Request) {
	// /api/v1/tools/:id/invoke
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/tools/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 || parts[1] != "invoke" {
		writeErr(w, 404, "not_found", "POST /api/v1/tools/:id/invoke", "")
		return
	}
	if r.Method != http.MethodPost {
		writeErr(w, 405, "method", "POST", "")
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req struct {
		Input     map[string]any `json:"input"`
		MissionID string         `json:"missionId"`
	}
	_ = json.Unmarshal(body, &req)
	inv, err := s.k.Tools().Invoke(r.Context(), parts[0], types.MissionID(req.MissionID), "", req.Input)
	if err != nil {
		writeEnv(w, 422, inv, map[string]any{"code": "invoke_failed", "message": err.Error()})
		return
	}
	writeOK(w, inv)
}
