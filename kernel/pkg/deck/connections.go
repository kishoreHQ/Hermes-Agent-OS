// Package deck implements Command Deck platform services (H3.1).
package deck

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/runtime"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

type ConnKind string

const (
	ConnProvider ConnKind = "provider"
	ConnRuntime  ConnKind = "runtime"
)

// Candidate from probe (plugins + optional local endpoints).
type Candidate struct {
	ID           string   `json:"id"`
	Kind         ConnKind `json:"kind"`
	Name         string   `json:"name"`
	Detected     bool     `json:"detected"`
	Version      string   `json:"version,omitempty"`
	Detail       string   `json:"detail,omitempty"`
	NeedsCred    bool     `json:"needsCredential"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// Connection is a registered live binding.
type Connection struct {
	ID           string    `json:"id"`
	Kind         ConnKind  `json:"kind"`
	PluginID     string    `json:"pluginId"`
	Name         string    `json:"name"`
	Status       string    `json:"status"` // connected|error|disconnected
	Version      string    `json:"version,omitempty"`
	Capabilities []string  `json:"capabilities,omitempty"`
	LastError    string    `json:"lastError,omitempty"`
	ConnectedAt  time.Time `json:"connectedAt"`
	CredentialID string    `json:"credentialId,omitempty"` // handle, never secret
}

type RegisterRequest struct {
	PluginID     string `json:"pluginId"`
	Kind         string `json:"kind"`
	Name         string `json:"name,omitempty"`
	CredentialID string `json:"credentialId,omitempty"`
}

// ConnectionsService is K1 probe + register.
type ConnectionsService struct {
	mu      sync.Mutex
	conns   map[string]*Connection
	plugins plugin.Registry
}

func NewConnections(reg plugin.Registry) *ConnectionsService {
	return &ConnectionsService{conns: map[string]*Connection{}, plugins: reg}
}

func (s *ConnectionsService) Probe(ctx context.Context) []Candidate {
	var out []Candidate
	if s.plugins != nil {
		for _, m := range s.plugins.List(plugin.KindProvider) {
			_, inst, ok := s.plugins.Get(m.Metadata.ID)
			det := "registered"
			detected := true
			var caps []string
			if ok {
				if p, ok := inst.(provider.Provider); ok {
					if err := p.Health(ctx); err != nil {
						detected = false
						det = err.Error()
					} else if d, err := p.Describe(ctx); err == nil {
						for _, c := range d.Capabilities {
							caps = append(caps, string(c))
						}
					}
				}
			}
			out = append(out, Candidate{
				ID: string(m.Metadata.ID), Kind: ConnProvider, Name: m.Metadata.Name,
				Detected: detected, Version: m.Metadata.Version, Detail: det,
				NeedsCred: true, Capabilities: caps,
			})
		}
		for _, m := range s.plugins.List(plugin.KindRuntime) {
			_, inst, ok := s.plugins.Get(m.Metadata.ID)
			det := "registered"
			detected := true
			if ok {
				if rt, ok := inst.(runtime.Runtime); ok {
					if err := rt.Health(ctx); err != nil {
						detected = false
						det = err.Error()
					}
				}
			}
			out = append(out, Candidate{
				ID: string(m.Metadata.ID), Kind: ConnRuntime, Name: m.Metadata.Name,
				Detected: detected, Version: m.Metadata.Version, Detail: det,
				NeedsCred: false, Capabilities: []string{"coding", "tools"},
			})
		}
	}
	// Local OpenAI-compatible endpoints (no vendor names in kernel logic)
	client := &http.Client{Timeout: 600 * time.Millisecond}
	for _, ep := range []struct {
		id, name, url string
	}{
		{"endpoint.local.11434", "openai-compat :11434", "http://127.0.0.1:11434/api/tags"},
		{"endpoint.local.1234", "openai-compat :1234", "http://127.0.0.1:1234/v1/models"},
	} {
		ok := false
		detail := "unreachable"
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ep.url, nil)
		if resp, err := client.Do(req); err == nil {
			_ = resp.Body.Close()
			ok = resp.StatusCode < 500
			if ok {
				detail = "reachable"
			} else {
				detail = fmt.Sprintf("http %d", resp.StatusCode)
			}
		}
		out = append(out, Candidate{
			ID: ep.id, Kind: ConnProvider, Name: ep.name, Detected: ok, Detail: detail,
			NeedsCred: false, Capabilities: []string{"coding"},
		})
	}
	return out
}

func (s *ConnectionsService) List() []Connection {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Connection, 0, len(s.conns))
	for _, c := range s.conns {
		out = append(out, *c)
	}
	return out
}

func (s *ConnectionsService) Register(ctx context.Context, req RegisterRequest) (*Connection, error) {
	if req.PluginID == "" {
		return nil, fmt.Errorf("pluginId required")
	}
	kind := ConnKind(req.Kind)
	if kind == "" {
		kind = ConnProvider
	}
	name := req.Name
	if name == "" {
		name = req.PluginID
	}
	status := "connected"
	lastErr := ""
	version := ""
	var caps []string
	if s.plugins != nil {
		if m, inst, ok := s.plugins.Get(types.PluginID(req.PluginID)); ok {
			version = m.Metadata.Version
			if name == req.PluginID && m.Metadata.Name != "" {
				name = m.Metadata.Name
			}
			switch kind {
			case ConnProvider:
				if p, ok := inst.(provider.Provider); ok {
					if err := p.Health(ctx); err != nil {
						status = "error"
						lastErr = err.Error()
					} else if d, err := p.Describe(ctx); err == nil {
						for _, c := range d.Capabilities {
							caps = append(caps, string(c))
						}
					}
				}
			case ConnRuntime:
				if rt, ok := inst.(runtime.Runtime); ok {
					if err := rt.Health(ctx); err != nil {
						status = "error"
						lastErr = err.Error()
					}
				}
			}
		} else {
			// endpoint-only connections
			status = "connected"
			lastErr = ""
		}
	}
	c := &Connection{
		ID: fmt.Sprintf("conn_%d", time.Now().UnixNano()), Kind: kind, PluginID: req.PluginID,
		Name: name, Status: status, Version: version, Capabilities: caps, LastError: lastErr,
		ConnectedAt: time.Now().UTC(), CredentialID: req.CredentialID,
	}
	s.mu.Lock()
	s.conns[c.ID] = c
	s.mu.Unlock()
	if status == "error" {
		return c, fmt.Errorf("%s", lastErr)
	}
	return c, nil
}
