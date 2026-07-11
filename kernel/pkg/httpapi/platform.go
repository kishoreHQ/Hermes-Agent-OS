package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/a2a"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/agentregistry"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/deploy"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/docgen"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/knowledge"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/planner"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

func (s *Server) registerPlatformRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/artifacts", s.apiArtifacts)
	mux.HandleFunc("/api/v1/artifacts/", s.apiArtifactSub)
	mux.HandleFunc("/api/v1/agents", s.apiAgents)
	mux.HandleFunc("/api/v1/plans", s.apiPlans)
	mux.HandleFunc("/api/v1/plans/", s.apiPlanSub)
	mux.HandleFunc("/api/v1/workflows/run", s.apiWorkflowRun)
	mux.HandleFunc("/api/v1/workflows/history", s.apiWorkflowHistory)
	mux.HandleFunc("/api/v1/knowledge/nodes", s.apiKGNodes)
	mux.HandleFunc("/api/v1/knowledge/edges", s.apiKGEdges)
	mux.HandleFunc("/api/v1/knowledge/query", s.apiKGQuery)
	mux.HandleFunc("/api/v1/mcp/tools", s.apiMCPTools)
	mux.HandleFunc("/api/v1/mcp/call", s.apiMCPCall)
	mux.HandleFunc("/api/v1/a2a/peers", s.apiA2APeers)
	mux.HandleFunc("/api/v1/a2a/tasks", s.apiA2ATasks)
	mux.HandleFunc("/api/v1/remediation/playbooks", s.apiRemPlaybooks)
	mux.HandleFunc("/api/v1/remediation/run", s.apiRemRun)
	mux.HandleFunc("/api/v1/deploy/sessions", s.apiDeploySessions)
	mux.HandleFunc("/api/v1/deploy/sessions/", s.apiDeploySub)
	mux.HandleFunc("/api/v1/docs", s.apiDocs)
	mux.HandleFunc("/api/v1/docs/generate", s.apiDocsGenerate)
}

func (s *Server) apiArtifacts(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		mission := types.MissionID(r.URL.Query().Get("mission"))
		writeOK(w, s.k.Artifacts.List(r.Context(), mission))
		return
	}
	if r.Method != http.MethodPost {
		writeErr(w, 405, "method", "GET or POST", "")
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req struct {
		Content   string `json:"content"`
		MediaType string `json:"mediaType"`
		MissionID string `json:"missionId"`
	}
	_ = json.Unmarshal(body, &req)
	meta, err := s.k.Artifacts.Put(r.Context(), []byte(req.Content), req.MediaType, types.MissionID(req.MissionID), nil)
	if err != nil {
		writeErr(w, 400, "put_failed", err.Error(), "")
		return
	}
	writeOK(w, meta)
}

func (s *Server) apiArtifactSub(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/artifacts/")
	id = strings.Trim(id, "/")
	if strings.HasSuffix(id, "/meta") {
		id = strings.TrimSuffix(id, "/meta")
		m, err := s.k.Artifacts.Meta(r.Context(), types.ArtifactDigest(id))
		if err != nil {
			writeErr(w, 404, "not_found", err.Error(), "")
			return
		}
		writeOK(w, m)
		return
	}
	data, meta, err := s.k.Artifacts.Get(r.Context(), types.ArtifactDigest(id))
	if err != nil {
		writeErr(w, 404, "not_found", err.Error(), "")
		return
	}
	writeOK(w, map[string]any{"meta": meta, "content": string(data)})
}

func (s *Server) apiAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeOK(w, s.k.Agents.List())
		return
	}
	if r.Method != http.MethodPost {
		writeErr(w, 405, "method", "GET or POST", "")
		return
	}
	body, _ := io.ReadAll(r.Body)
	var a agentregistry.Agent
	_ = json.Unmarshal(body, &a)
	if err := s.k.Agents.Register(a); err != nil {
		writeErr(w, 400, "bad_request", err.Error(), "")
		return
	}
	writeOK(w, a)
}

func (s *Server) apiPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeOK(w, s.k.Plans.List(types.MissionID(r.URL.Query().Get("mission"))))
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req struct {
		MissionID string         `json:"missionId"`
		Goal      string         `json:"goal"`
		Steps     []planner.Step `json:"steps"`
	}
	_ = json.Unmarshal(body, &req)
	p, err := s.k.Plans.Create(types.MissionID(req.MissionID), req.Goal, req.Steps)
	if err != nil {
		writeErr(w, 400, "bad_request", err.Error(), "")
		return
	}
	writeOK(w, p)
}

func (s *Server) apiPlanSub(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/plans/"), "/")
	p, ok := s.k.Plans.Get(id)
	if !ok {
		writeErr(w, 404, "not_found", "plan not found", "")
		return
	}
	writeOK(w, p)
}

func (s *Server) apiWorkflowRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, 405, "method", "POST", "")
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req struct {
		PlanID string `json:"planId"`
		Goal   string `json:"goal"`
	}
	_ = json.Unmarshal(body, &req)
	planID := req.PlanID
	if planID == "" {
		p, err := s.k.Plans.Create("", req.Goal, nil)
		if err != nil {
			writeErr(w, 400, "plan_failed", err.Error(), "")
			return
		}
		planID = p.ID
	}
	res, err := s.k.Workflow.RunPlan(r.Context(), planID)
	if err != nil {
		writeEnv(w, 422, res, map[string]any{"code": "workflow_failed", "message": err.Error()})
		return
	}
	writeOK(w, res)
}

func (s *Server) apiWorkflowHistory(w http.ResponseWriter, r *http.Request) {
	writeOK(w, s.k.Workflow.History())
}

func (s *Server) apiKGNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		body, _ := io.ReadAll(r.Body)
		var n knowledge.Node
		_ = json.Unmarshal(body, &n)
		writeOK(w, s.k.Knowledge.UpsertNode(n))
		return
	}
	writeOK(w, s.k.Knowledge.Stats())
}

func (s *Server) apiKGEdges(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, 405, "method", "POST", "")
		return
	}
	body, _ := io.ReadAll(r.Body)
	var e knowledge.Edge
	_ = json.Unmarshal(body, &e)
	out, err := s.k.Knowledge.UpsertEdge(e)
	if err != nil {
		writeErr(w, 400, "bad_request", err.Error(), "")
		return
	}
	writeOK(w, out)
}

func (s *Server) apiKGQuery(w http.ResponseWriter, r *http.Request) {
	writeOK(w, s.k.Knowledge.Query(r.URL.Query().Get("type"), r.URL.Query().Get("q")))
}

func (s *Server) apiMCPTools(w http.ResponseWriter, r *http.Request) {
	writeOK(w, s.k.MCP.ListTools())
}

func (s *Server) apiMCPCall(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req struct {
		Name string         `json:"name"`
		Args map[string]any `json:"arguments"`
	}
	_ = json.Unmarshal(body, &req)
	out, err := s.k.MCP.CallTool(r.Context(), req.Name, req.Args)
	if err != nil {
		writeErr(w, 422, "mcp_call_failed", err.Error(), "")
		return
	}
	writeOK(w, map[string]any{"content": out})
}

func (s *Server) apiA2APeers(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		body, _ := io.ReadAll(r.Body)
		var p a2a.Peer
		_ = json.Unmarshal(body, &p)
		if err := s.k.A2A.Register(p); err != nil {
			writeErr(w, 400, "bad_request", err.Error(), "")
			return
		}
		writeOK(w, p)
		return
	}
	writeOK(w, s.k.A2A.List())
}

func (s *Server) apiA2ATasks(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			PeerID string `json:"peerId"`
			Goal   string `json:"goal"`
		}
		_ = json.Unmarshal(body, &req)
		t, err := s.k.A2A.OfferTaskCtx(r.Context(), types.PluginID(req.PeerID), req.Goal)
		if err != nil {
			writeErr(w, 400, "bad_request", err.Error(), "")
			return
		}
		writeOK(w, t)
		return
	}
	writeOK(w, s.k.A2A.Tasks())
}

func (s *Server) apiRemPlaybooks(w http.ResponseWriter, r *http.Request) {
	writeOK(w, s.k.Remediate.List())
}

func (s *Server) apiRemRun(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req struct {
		PlaybookID  string `json:"playbookId"`
		FreezeWindow bool  `json:"freezeWindow"`
	}
	_ = json.Unmarshal(body, &req)
	run, err := s.k.Remediate.Run(req.PlaybookID, req.FreezeWindow)
	if err != nil {
		writeEnv(w, 422, run, map[string]any{"code": "remediation_denied", "message": err.Error()})
		return
	}
	writeOK(w, run)
}

func (s *Server) apiDeploySessions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeOK(w, s.k.Deploy.List())
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req deploy.CreateReq
	_ = json.Unmarshal(body, &req)
	sess, err := s.k.Deploy.Create(req)
	if err != nil {
		writeErr(w, 400, "bad_request", err.Error(), "")
		return
	}
	writeOK(w, sess)
}

func (s *Server) apiDeploySub(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/deploy/sessions/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeErr(w, 400, "bad_request", "missing id", "")
		return
	}
	id := parts[0]
	if len(parts) == 1 {
		sess, ok := s.k.Deploy.Get(id)
		if !ok {
			writeErr(w, 404, "not_found", "not found", "")
			return
		}
		writeOK(w, sess)
		return
	}
	if parts[1] == "advance" {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Approve bool `json:"approve"`
		}
		_ = json.Unmarshal(body, &req)
		sess, err := s.k.Deploy.Advance(id, req.Approve)
		if err != nil {
			writeErr(w, 400, "advance_failed", err.Error(), "")
			return
		}
		writeOK(w, sess)
		return
	}
	writeErr(w, 404, "not_found", "use advance", "")
}

func (s *Server) apiDocs(w http.ResponseWriter, r *http.Request) {
	writeOK(w, s.k.Docs.List())
}

func (s *Server) apiDocsGenerate(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req docgen.GenerateReq
	_ = json.Unmarshal(body, &req)
	d, err := s.k.Docs.Generate(req)
	if err != nil {
		writeErr(w, 400, "bad_request", err.Error(), "")
		return
	}
	// Also store as artifact when possible
	if s.k.Artifacts != nil {
		if meta, err := s.k.Artifacts.Put(r.Context(), []byte(d.Body), "text/markdown", types.MissionID(req.MissionID), map[string]string{"kind": "doc"}); err == nil {
			d.Digest = meta.Digest
		}
	}
	writeOK(w, d)
}
