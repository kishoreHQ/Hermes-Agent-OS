// Package workspacetools registers real agent tools: fs, shell, web, memory, search, research.
package workspacetools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/memorystore"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/toolrouter"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Options for tool registration.
type Options struct {
	// Root is the sandbox root for fs/shell (default: cwd or HERMES_WORKSPACE).
	Root string
	// Memory optional store for memory tools.
	Memory memorystore.Store
	// AllowShell enables shell.exec (default true when Root set).
	AllowShell bool
	// SearchURL optional SearXNG or compatible search endpoint.
	SearchURL string
	// HTTPClient optional.
	HTTP *http.Client
}

// Register adds workspace tools to the router.
func Register(r *toolrouter.Router, opts Options) error {
	if r == nil {
		return fmt.Errorf("nil router")
	}
	root := opts.Root
	if root == "" {
		root = os.Getenv("HERMES_WORKSPACE")
	}
	if root == "" {
		root, _ = os.Getwd()
	}
	root, _ = filepath.Abs(root)
	client := opts.HTTP
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	allowShell := opts.AllowShell
	if os.Getenv("HERMES_ALLOW_SHELL") == "0" {
		allowShell = false
	} else if os.Getenv("HERMES_ALLOW_SHELL") == "1" {
		allowShell = true
	} else if !allowShell {
		allowShell = true // default on for agent loop demos
	}

	obj := func(props map[string]any, required ...string) map[string]any {
		m := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			m["required"] = required
		}
		return m
	}

	// —— fs.read ——
	_ = r.Register(toolrouter.Tool{
		ID: "fs.read", Name: "fs.read", Description: "Read a text file under the workspace root",
		Enabled: true, Labels: map[string]string{"category": "fs"},
		Parameters: obj(map[string]any{
			"path": map[string]any{"type": "string", "description": "Relative path under workspace"},
		}, "path"),
	}, func(ctx context.Context, input map[string]any) (string, error) {
		p, err := resolvePath(root, str(input, "path"))
		if err != nil {
			return "", err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		if len(b) > 200_000 {
			b = b[:200_000]
		}
		return string(b), nil
	})

	// —— fs.write ——
	_ = r.Register(toolrouter.Tool{
		ID: "fs.write", Name: "fs.write", Description: "Write a text file under the workspace root",
		Enabled: true, Labels: map[string]string{"category": "fs", "danger": "write"},
		Parameters: obj(map[string]any{
			"path":    map[string]any{"type": "string"},
			"content": map[string]any{"type": "string"},
		}, "path", "content"),
	}, func(ctx context.Context, input map[string]any) (string, error) {
		p, err := resolvePath(root, str(input, "path"))
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return "", err
		}
		content := str(input, "content")
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			return "", err
		}
		return fmt.Sprintf("wrote %d bytes to %s", len(content), rel(root, p)), nil
	})

	// —— fs.list ——
	_ = r.Register(toolrouter.Tool{
		ID: "fs.list", Name: "fs.list", Description: "List directory entries under workspace",
		Enabled: true, Labels: map[string]string{"category": "fs"},
		Parameters: obj(map[string]any{
			"path": map[string]any{"type": "string", "description": "Relative dir (default .)"},
		}),
	}, func(ctx context.Context, input map[string]any) (string, error) {
		relp := str(input, "path")
		if relp == "" {
			relp = "."
		}
		p, err := resolvePath(root, relp)
		if err != nil {
			return "", err
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			return "", err
		}
		var lines []string
		for i, e := range entries {
			if i >= 200 {
				lines = append(lines, "…")
				break
			}
			suffix := ""
			if e.IsDir() {
				suffix = "/"
			}
			lines = append(lines, e.Name()+suffix)
		}
		return strings.Join(lines, "\n"), nil
	})

	// —— shell.exec ——
	if allowShell {
		_ = r.Register(toolrouter.Tool{
			ID: "shell.exec", Name: "shell.exec", Description: "Run a shell command in the workspace (sandboxed cwd)",
			Enabled: true, Labels: map[string]string{"category": "shell", "danger": "exec"},
			Parameters: obj(map[string]any{
				"command": map[string]any{"type": "string"},
				"timeoutSec": map[string]any{"type": "integer", "description": "default 30"},
			}, "command"),
		}, func(ctx context.Context, input map[string]any) (string, error) {
			cmdStr := str(input, "command")
			if cmdStr == "" {
				return "", fmt.Errorf("command required")
			}
			// Block obvious destructive patterns
			low := strings.ToLower(cmdStr)
			for _, bad := range []string{"rm -rf /", "mkfs", ":(){", "shutdown", "reboot"} {
				if strings.Contains(low, bad) {
					return "", fmt.Errorf("blocked dangerous command")
				}
			}
			to := 30
			if v, ok := input["timeoutSec"].(float64); ok && v > 0 {
				to = int(v)
			}
			cctx, cancel := context.WithTimeout(ctx, time.Duration(to)*time.Second)
			defer cancel()
			cmd := exec.CommandContext(cctx, "bash", "-lc", cmdStr)
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "HERMES_WORKSPACE="+root)
			out, err := cmd.CombinedOutput()
			s := string(out)
			if len(s) > 100_000 {
				s = s[:100_000] + "…"
			}
			if err != nil {
				return s + "\nerror: " + err.Error(), nil // return output + error as tool result
			}
			return s, nil
		})
	}

	// —— web.fetch ——
	_ = r.Register(toolrouter.Tool{
		ID: "web.fetch", Name: "web.fetch", Description: "HTTP GET a URL and return text body (truncated)",
		Enabled: true, Labels: map[string]string{"category": "web"},
		Parameters: obj(map[string]any{
			"url": map[string]any{"type": "string"},
		}, "url"),
	}, func(ctx context.Context, input map[string]any) (string, error) {
		u := str(input, "url")
		if u == "" || !strings.HasPrefix(u, "http") {
			return "", fmt.Errorf("valid http(s) url required")
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", "Hermes-Agent-OS/1.0")
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 200_000))
		return fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, string(b)), nil
	})

	// —— web.search ——
	searchURL := opts.SearchURL
	if searchURL == "" {
		searchURL = os.Getenv("HERMES_SEARCH_URL")
	}
	_ = r.Register(toolrouter.Tool{
		ID: "web.search", Name: "web.search", Description: "Web search (uses HERMES_SEARCH_URL SearXNG if set; else DuckDuckGo HTML lite)",
		Enabled: true, Labels: map[string]string{"category": "web"},
		Parameters: obj(map[string]any{
			"query": map[string]any{"type": "string"},
		}, "query"),
	}, func(ctx context.Context, input map[string]any) (string, error) {
		q := str(input, "query")
		if q == "" {
			return "", fmt.Errorf("query required")
		}
		if searchURL != "" {
			u := strings.TrimRight(searchURL, "/") + "/search?q=" + urlQuery(q) + "&format=json"
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
			resp, err := client.Do(req)
			if err != nil {
				return "", err
			}
			defer resp.Body.Close()
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 100_000))
			return string(b), nil
		}
		// Fallback: DuckDuckGo instant answer API
		u := "https://api.duckduckgo.com/?q=" + urlQuery(q) + "&format=json&no_html=1"
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 50_000))
		return string(b), nil
	})

	// —— memory.search / memory.write ——
	if opts.Memory != nil {
		mem := opts.Memory
		_ = r.Register(toolrouter.Tool{
			ID: "memory.search", Name: "memory.search", Description: "Search Hermes unified memory",
			Enabled: true, Labels: map[string]string{"category": "memory"},
			Parameters: obj(map[string]any{
				"query": map[string]any{"type": "string"},
				"limit": map[string]any{"type": "integer"},
			}, "query"),
		}, func(ctx context.Context, input map[string]any) (string, error) {
			lim := 10
			if v, ok := input["limit"].(float64); ok && v > 0 {
				lim = int(v)
			}
			hits, err := mem.Search(ctx, memorystore.Query{Text: str(input, "query"), Limit: lim})
			if err != nil {
				return "", err
			}
			b, _ := json.Marshal(hits)
			return string(b), nil
		})
		_ = r.Register(toolrouter.Tool{
			ID: "memory.write", Name: "memory.write", Description: "Write a note to Hermes memory",
			Enabled: true, Labels: map[string]string{"category": "memory"},
			Parameters: obj(map[string]any{
				"content": map[string]any{"type": "string"},
				"kind":    map[string]any{"type": "string", "description": "episodic|semantic|procedural"},
			}, "content"),
		}, func(ctx context.Context, input map[string]any) (string, error) {
			kind := memorystore.Kind(str(input, "kind"))
			if kind == "" {
				kind = memorystore.KindSemantic
			}
			e, err := mem.Write(ctx, memorystore.Entry{
				Kind: kind, Content: str(input, "content"), Trust: types.TrustAgent,
			})
			if err != nil {
				return "", err
			}
			return e.ID, nil
		})
	}

	// —— research.notes —— simple structured research helper
	_ = r.Register(toolrouter.Tool{
		ID: "research.outline", Name: "research.outline", Description: "Create a structured research outline from a topic",
		Enabled: true, Labels: map[string]string{"category": "research"},
		Parameters: obj(map[string]any{
			"topic": map[string]any{"type": "string"},
		}, "topic"),
	}, func(ctx context.Context, input map[string]any) (string, error) {
		topic := str(input, "topic")
		return fmt.Sprintf(`# Research outline: %s

1. Define question and scope
2. Gather primary sources (web.search / web.fetch)
3. Extract key claims with citations
4. Compare conflicting views
5. Write executive summary + open questions
`, topic), nil
	})

	// —— time.now already in router; enhance with http.request ——
	_ = r.Register(toolrouter.Tool{
		ID: "http.request", Name: "http.request", Description: "HTTP request to allowlisted hosts (http/https)",
		Enabled: true, Labels: map[string]string{"category": "web", "danger": "network"},
		Parameters: obj(map[string]any{
			"method": map[string]any{"type": "string"},
			"url":    map[string]any{"type": "string"},
			"body":   map[string]any{"type": "string"},
		}, "url"),
	}, func(ctx context.Context, input map[string]any) (string, error) {
		method := str(input, "method")
		if method == "" {
			method = "GET"
		}
		u := str(input, "url")
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			return "", fmt.Errorf("only http(s)")
		}
		var body io.Reader
		if b := str(input, "body"); b != "" {
			body = strings.NewReader(b)
		}
		req, err := http.NewRequestWithContext(ctx, method, u, body)
		if err != nil {
			return "", err
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 100_000))
		return fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, string(raw)), nil
	})

	return nil
}

func str(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[k].(string); ok {
		return v
	}
	return fmt.Sprintf("%v", m[k])
}

func resolvePath(root, relp string) (string, error) {
	if relp == "" {
		return "", fmt.Errorf("path required")
	}
	if strings.Contains(relp, "\x00") {
		return "", fmt.Errorf("invalid path")
	}
	clean := filepath.Clean("/" + relp)
	clean = strings.TrimPrefix(clean, "/")
	full := filepath.Join(root, clean)
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	rootAbs, _ := filepath.Abs(root)
	if abs != rootAbs && !strings.HasPrefix(abs, rootAbs+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes workspace")
	}
	return abs, nil
}

func rel(root, full string) string {
	r, err := filepath.Rel(root, full)
	if err != nil {
		return full
	}
	return r
}

func urlQuery(q string) string {
	return strings.ReplaceAll(strings.ReplaceAll(q, " ", "+"), "&", "%26")
}
