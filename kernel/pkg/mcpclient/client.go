// Package mcpclient is a real multi-server MCP client (stdio + HTTP JSON-RPC lite).
// Tools are mirrored into Hermes toolrouter as mcp.<server>.<tool>.
package mcpclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/toolrouter"
)

// Transport kind.
type Transport string

const (
	TransportStdio Transport = "stdio"
	TransportHTTP  Transport = "http"
)

// ServerConfig is an MCP server definition (UI / env).
type ServerConfig struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Transport Transport       `json:"transport"` // stdio|http
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Enabled bool              `json:"enabled"`
}

// Status of a server connection.
type Status struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	State   string   `json:"state"` // connected|disconnected|error
	Error   string   `json:"error,omitempty"`
	Tools   []string `json:"tools,omitempty"`
	Transport string `json:"transport"`
}

// Manager holds MCP servers and registers tools into toolrouter.
type Manager struct {
	mu      sync.Mutex
	tools   *toolrouter.Router
	servers map[string]*ServerConfig
	status  map[string]*Status
	// stdio sessions
	procs map[string]*stdioSession
	http  *http.Client
	seq   atomic.Int64
}

type stdioSession struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
}

func New(tools *toolrouter.Router) *Manager {
	return &Manager{
		tools: tools,
		servers: map[string]*ServerConfig{},
		status:  map[string]*Status{},
		procs:   map[string]*stdioSession{},
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (m *Manager) List() []ServerConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]ServerConfig, 0, len(m.servers))
	for _, s := range m.servers {
		out = append(out, *s)
	}
	return out
}

func (m *Manager) Statuses() []Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Status, 0, len(m.status))
	for _, s := range m.status {
		out = append(out, *s)
	}
	return out
}

// Upsert registers config (does not auto-connect unless Connect called).
func (m *Manager) Upsert(cfg ServerConfig) error {
	if cfg.ID == "" {
		return fmt.Errorf("id required")
	}
	if cfg.Name == "" {
		cfg.Name = cfg.ID
	}
	if cfg.Transport == "" {
		if cfg.URL != "" {
			cfg.Transport = TransportHTTP
		} else {
			cfg.Transport = TransportStdio
		}
	}
	m.mu.Lock()
	cp := cfg
	m.servers[cfg.ID] = &cp
	if m.status[cfg.ID] == nil {
		m.status[cfg.ID] = &Status{ID: cfg.ID, Name: cfg.Name, State: "disconnected", Transport: string(cfg.Transport)}
	}
	m.mu.Unlock()
	return nil
}

func (m *Manager) Delete(id string) error {
	_ = m.Disconnect(id)
	m.mu.Lock()
	delete(m.servers, id)
	delete(m.status, id)
	m.mu.Unlock()
	return nil
}

// Connect starts the server and registers tools.
func (m *Manager) Connect(ctx context.Context, id string) error {
	m.mu.Lock()
	cfg, ok := m.servers[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("unknown server %s", id)
	}
	cp := *cfg
	m.mu.Unlock()

	switch cp.Transport {
	case TransportHTTP:
		return m.connectHTTP(ctx, cp)
	default:
		return m.connectStdio(ctx, cp)
	}
}

func (m *Manager) Disconnect(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sess, ok := m.procs[id]; ok {
		if sess.stdin != nil {
			_ = sess.stdin.Close()
		}
		if sess.cmd != nil && sess.cmd.Process != nil {
			_ = sess.cmd.Process.Kill()
		}
		delete(m.procs, id)
	}
	if st, ok := m.status[id]; ok {
		st.State = "disconnected"
		st.Tools = nil
		st.Error = ""
	}
	// unregister tools with prefix
	if m.tools != nil {
		prefix := "mcp." + id + "."
		for _, t := range m.tools.List() {
			if strings.HasPrefix(t.ID, prefix) {
				_ = m.tools.Unregister(t.ID)
			}
		}
	}
	return nil
}

func (m *Manager) connectStdio(ctx context.Context, cfg ServerConfig) error {
	if cfg.Command == "" {
		return fmt.Errorf("command required for stdio MCP")
	}
	cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)
	env := os.Environ()
	for k, v := range cfg.Env {
		env = append(env, k+"="+v)
	}
	cmd.Env = env
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	sess := &stdioSession{cmd: cmd, stdin: stdin, stdout: bufio.NewReader(stdout)}
	m.mu.Lock()
	m.procs[cfg.ID] = sess
	m.mu.Unlock()

	// initialize
	_, err = m.rpcStdio(sess, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "hermes", "version": "1.0"},
	})
	if err != nil {
		m.setErr(cfg.ID, err)
		return err
	}
	_, _ = m.rpcStdio(sess, "notifications/initialized", nil)

	tools, err := m.listToolsStdio(sess)
	if err != nil {
		m.setErr(cfg.ID, err)
		return err
	}
	m.registerTools(cfg.ID, tools, func(toolName string, args map[string]any) (string, error) {
		res, err := m.rpcStdio(sess, "tools/call", map[string]any{
			"name": toolName, "arguments": args,
		})
		if err != nil {
			return "", err
		}
		return formatToolResult(res), nil
	})
	m.setOK(cfg.ID, cfg.Name, string(cfg.Transport), toolNames(tools))
	return nil
}

func (m *Manager) connectHTTP(ctx context.Context, cfg ServerConfig) error {
	if cfg.URL == "" {
		return fmt.Errorf("url required for http MCP")
	}
	// Minimal JSON-RPC over HTTP POST
	tools, err := m.listToolsHTTP(ctx, cfg.URL)
	if err != nil {
		m.setErr(cfg.ID, err)
		return err
	}
	url := cfg.URL
	m.registerTools(cfg.ID, tools, func(toolName string, args map[string]any) (string, error) {
		res, err := m.rpcHTTP(context.Background(), url, "tools/call", map[string]any{
			"name": toolName, "arguments": args,
		})
		if err != nil {
			return "", err
		}
		return formatToolResult(res), nil
	})
	m.setOK(cfg.ID, cfg.Name, string(cfg.Transport), toolNames(tools))
	return nil
}

type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func (m *Manager) listToolsStdio(sess *stdioSession) ([]mcpTool, error) {
	res, err := m.rpcStdio(sess, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	return parseTools(res)
}

func (m *Manager) listToolsHTTP(ctx context.Context, url string) ([]mcpTool, error) {
	res, err := m.rpcHTTP(ctx, url, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	return parseTools(res)
}

func parseTools(res any) ([]mcpTool, error) {
	b, _ := json.Marshal(res)
	var wrap struct {
		Tools []mcpTool `json:"tools"`
	}
	if err := json.Unmarshal(b, &wrap); err != nil {
		// maybe res is already {tools:…} nested in result
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		if t, ok := m["tools"]; ok {
			bb, _ := json.Marshal(t)
			var tools []mcpTool
			if err := json.Unmarshal(bb, &tools); err != nil {
				return nil, err
			}
			return tools, nil
		}
		return nil, err
	}
	return wrap.Tools, nil
}

func (m *Manager) registerTools(serverID string, tools []mcpTool, call func(name string, args map[string]any) (string, error)) {
	if m.tools == nil {
		return
	}
	for _, t := range tools {
		id := "mcp." + serverID + "." + t.Name
		name := t.Name
		_ = m.tools.Register(toolrouter.Tool{
			ID: id, Name: id, Description: t.Description, Enabled: true,
			Parameters: t.InputSchema,
			Labels:     map[string]string{"mcp": serverID, "category": "mcp"},
		}, func(ctx context.Context, input map[string]any) (string, error) {
			return call(name, input)
		})
	}
}

func (m *Manager) rpcStdio(sess *stdioSession, method string, params any) (any, error) {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	id := m.seq.Add(1)
	msg := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	// notifications have no id response expectation for initialized — still ok
	b, _ := json.Marshal(msg)
	b = append(b, '\n')
	if _, err := sess.stdin.Write(b); err != nil {
		return nil, err
	}
	if strings.HasPrefix(method, "notifications/") {
		return nil, nil
	}
	// read until matching id
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		line, err := sess.stdout.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		var resp map[string]any
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}
		// skip notifications
		if _, ok := resp["id"]; !ok {
			continue
		}
		if errObj, ok := resp["error"]; ok && errObj != nil {
			return nil, fmt.Errorf("mcp error: %v", errObj)
		}
		return resp["result"], nil
	}
	return nil, fmt.Errorf("mcp timeout")
}

func (m *Manager) rpcHTTP(ctx context.Context, url, method string, params any) (any, error) {
	id := m.seq.Add(1)
	msg := map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}
	b, _ := json.Marshal(msg)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("mcp http %d: %s", resp.StatusCode, string(raw))
	}
	if e, ok := out["error"]; ok && e != nil {
		return nil, fmt.Errorf("mcp error: %v", e)
	}
	return out["result"], nil
}

func (m *Manager) setOK(id, name, transport string, tools []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status[id] = &Status{ID: id, Name: name, State: "connected", Tools: tools, Transport: transport}
}

func (m *Manager) setErr(id string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.status[id]
	if st == nil {
		st = &Status{ID: id}
		m.status[id] = st
	}
	st.State = "error"
	st.Error = err.Error()
}

func toolNames(tools []mcpTool) []string {
	out := make([]string, 0, len(tools))
	for _, t := range tools {
		out = append(out, t.Name)
	}
	return out
}

func formatToolResult(res any) string {
	if res == nil {
		return ""
	}
	b, err := json.Marshal(res)
	if err != nil {
		return fmt.Sprintf("%v", res)
	}
	// Prefer content[].text if MCP shape
	var wrap struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if json.Unmarshal(b, &wrap) == nil && len(wrap.Content) > 0 {
		var parts []string
		for _, c := range wrap.Content {
			if c.Text != "" {
				parts = append(parts, c.Text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	return string(b)
}
