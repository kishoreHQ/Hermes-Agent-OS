package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/compare"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/workspace"
)

func (s *Server) registerWorkspaceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/notes", s.apiNotes)
	mux.HandleFunc("/api/v1/notes/", s.apiNoteSub)
	mux.HandleFunc("/api/v1/todos", s.apiTodos)
	mux.HandleFunc("/api/v1/todos/", s.apiTodoSub)
	mux.HandleFunc("/api/v1/documents", s.apiDocuments)
	mux.HandleFunc("/api/v1/documents/", s.apiDocumentSub)
	mux.HandleFunc("/api/v1/vault", s.apiVault)
	mux.HandleFunc("/api/v1/uploads", s.apiUploads)
	mux.HandleFunc("/api/v1/presets", s.apiPresets)
	mux.HandleFunc("/api/v1/webhooks", s.apiWebhooks)
	mux.HandleFunc("/api/v1/webhooks/", s.apiWebhookSub)
	mux.HandleFunc("/api/v1/compare", s.apiCompare)
	mux.HandleFunc("/api/v1/backup", s.apiBackup)
	mux.HandleFunc("/api/v1/diagnostics", s.apiDiagnostics)
}

func (s *Server) ws() *workspace.Store {
	if s.k.Workspace == nil {
		s.k.Workspace = workspace.New("")
	}
	return s.k.Workspace
}

func (s *Server) apiNotes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query().Get("q")
		if q != "" {
			writeOK(w, s.ws().SearchNotes(q))
			return
		}
		writeOK(w, s.ws().ListNotes())
	case http.MethodPost:
		var n workspace.Note
		_ = json.NewDecoder(r.Body).Decode(&n)
		writeOK(w, s.ws().PutNote(n))
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}

func (s *Server) apiNoteSub(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/notes/"), "/")
	if r.Method == http.MethodGet {
		n, ok := s.ws().GetNote(id)
		if !ok {
			writeErr(w, 404, "not_found", "note", "")
			return
		}
		writeOK(w, n)
		return
	}
	if r.Method == http.MethodDelete {
		s.ws().DeleteNote(id)
		writeOK(w, map[string]any{"deleted": id})
		return
	}
	writeErr(w, 405, "method", "GET or DELETE", "")
}

func (s *Server) apiTodos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.ws().ListTodos())
	case http.MethodPost:
		var t workspace.Todo
		_ = json.NewDecoder(r.Body).Decode(&t)
		writeOK(w, s.ws().PutTodo(t))
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}

func (s *Server) apiTodoSub(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/todos/"), "/")
	if r.Method == http.MethodDelete {
		s.ws().DeleteTodo(id)
		writeOK(w, map[string]any{"deleted": id})
		return
	}
	writeErr(w, 405, "method", "DELETE", "")
}

func (s *Server) apiDocuments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.ws().ListDocs())
	case http.MethodPost:
		var d workspace.Doc
		_ = json.NewDecoder(r.Body).Decode(&d)
		writeOK(w, s.ws().PutDoc(d))
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}

func (s *Server) apiDocumentSub(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/documents/"), "/")
	if r.Method == http.MethodGet {
		d, ok := s.ws().GetDoc(id)
		if !ok {
			writeErr(w, 404, "not_found", "document", "")
			return
		}
		writeOK(w, d)
		return
	}
	if r.Method == http.MethodDelete {
		s.ws().DeleteDoc(id)
		writeOK(w, map[string]any{"deleted": id})
		return
	}
	writeErr(w, 405, "method", "GET or DELETE", "")
}

func (s *Server) apiVault(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.ws().ListVault())
	case http.MethodPost:
		var v workspace.VaultEntry
		_ = json.NewDecoder(r.Body).Decode(&v)
		writeOK(w, s.ws().PutVault(v))
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}

func (s *Server) apiUploads(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.ws().ListUploads())
	case http.MethodPost:
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			// JSON fallback
			var req struct {
				Name      string `json:"name"`
				Content   string `json:"content"`
				MediaType string `json:"mediaType"`
			}
			if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
				writeErr(w, 400, "bad_request", err.Error(), "")
				return
			}
			u, e := s.ws().SaveUpload(req.Name, req.MediaType, []byte(req.Content))
			if e != nil {
				writeErr(w, 400, "upload_failed", e.Error(), "")
				return
			}
			writeOK(w, u)
			return
		}
		file, hdr, err := r.FormFile("file")
		if err != nil {
			writeErr(w, 400, "bad_request", "file field required", "")
			return
		}
		defer file.Close()
		raw, _ := io.ReadAll(io.LimitReader(file, 32<<20))
		u, err := s.ws().SaveUpload(hdr.Filename, hdr.Header.Get("Content-Type"), raw)
		if err != nil {
			writeErr(w, 400, "upload_failed", err.Error(), "")
			return
		}
		writeOK(w, u)
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}

func (s *Server) apiPresets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.ws().ListPresets())
	case http.MethodPost:
		var p workspace.Preset
		_ = json.NewDecoder(r.Body).Decode(&p)
		writeOK(w, s.ws().PutPreset(p))
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}

func (s *Server) apiWebhooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeOK(w, s.ws().ListWebhooks())
	case http.MethodPost:
		var h workspace.Webhook
		_ = json.NewDecoder(r.Body).Decode(&h)
		writeOK(w, s.ws().PutWebhook(h))
	default:
		writeErr(w, 405, "method", "GET or POST", "")
	}
}

func (s *Server) apiWebhookSub(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/webhooks/"), "/")
	if r.Method == http.MethodDelete {
		s.ws().DeleteWebhook(id)
		writeOK(w, map[string]any{"deleted": id})
		return
	}
	writeErr(w, 405, "method", "DELETE", "")
}

func (s *Server) apiCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, 405, "method", "POST", "")
		return
	}
	var req compare.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "bad_json", err.Error(), "")
		return
	}
	res, err := compare.Run(r.Context(), s.k, req)
	if err != nil {
		writeErr(w, 400, "compare_failed", err.Error(), "")
		return
	}
	writeOK(w, res)
}

func (s *Server) apiBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, 405, "method", "GET", "")
		return
	}
	snap := map[string]any{
		"version":   Version,
		"exportedAt": time.Now().UTC(),
		"workspace": s.ws().Snapshot(),
		"skills":    nil,
		"chat":      nil,
	}
	if s.k.Skills != nil {
		snap["skills"] = s.k.Skills.List()
	}
	if s.k.Chat != nil {
		snap["chat"] = s.k.Chat.List()
	}
	if s.k.Scheduler != nil {
		snap["jobs"] = s.k.Scheduler.List()
	}
	writeOK(w, snap)
}

func (s *Server) apiDiagnostics(w http.ResponseWriter, r *http.Request) {
	toolsN := 0
	if s.k.Tools() != nil {
		toolsN = len(s.k.Tools().List())
	}
	mcpN := 0
	if s.k.MCPClient != nil {
		mcpN = len(s.k.MCPClient.List())
	}
	writeOK(w, map[string]any{
		"status":    "ok",
		"version":   Version,
		"time":      time.Now().UTC(),
		"tools":     toolsN,
		"mcpServers": mcpN,
		"workspace": s.ws().DataDir(),
		"ready":     true,
	})
}
