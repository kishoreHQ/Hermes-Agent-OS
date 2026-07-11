package providercfg

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/adapters/echo"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/adapters/openaicompat"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/credentials"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Config is a UI-managed provider definition (no secrets — use credential handles).
type Config struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Driver       string           `json:"driver"` // openai-compat | echo-provider
	TemplateID   string           `json:"templateId,omitempty"`
	BaseURL      string           `json:"baseUrl,omitempty"`
	Local        bool             `json:"local"`
	CostTier     types.CostTier   `json:"costTier"`
	DefaultModel string           `json:"defaultModel,omitempty"`
	Models       []string         `json:"models,omitempty"`
	// CredentialHandle links an existing broker handle (never the secret).
	CredentialHandle string            `json:"credentialHandle,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	// Managed true = created via UI/API (deletable); false = disk/seed bootstrap.
	Managed   bool      `json:"managed"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateRequest from UI / Host API.
type CreateRequest struct {
	// FromTemplate seeds defaults (e.g. "openai", "ollama").
	FromTemplate string `json:"fromTemplate,omitempty"`
	// ID optional; auto-generated if empty (provider.ui.<slug>).
	ID           string         `json:"id,omitempty"`
	Name         string         `json:"name,omitempty"`
	Driver       string         `json:"driver,omitempty"`
	BaseURL      string         `json:"baseUrl,omitempty"`
	Local        *bool          `json:"local,omitempty"`
	CostTier     types.CostTier `json:"costTier,omitempty"`
	DefaultModel string         `json:"defaultModel,omitempty"`
	Models       []string       `json:"models,omitempty"`
	// APIKey stored via credential broker; never returned on list.
	APIKey string `json:"apiKey,omitempty"`
	// CredentialHandle reuse existing handle instead of apiKey.
	CredentialHandle string `json:"credentialHandle,omitempty"`
}

// RefreshAdapters is called after register/unregister.
type RefreshFunc func()

// Manager adds/removes provider plugins at runtime for UI management.
type Manager struct {
	mu      sync.Mutex
	reg     plugin.Registry
	creds   credentials.Broker
	configs map[types.PluginID]*Config
	refresh RefreshFunc
}

func NewManager(reg plugin.Registry, creds credentials.Broker, refresh RefreshFunc) *Manager {
	return &Manager{
		reg: reg, creds: creds, configs: map[types.PluginID]*Config{}, refresh: refresh,
	}
}

// SyncFromRegistry marks existing providers as unmanaged baseline.
func (m *Manager) SyncFromRegistry() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, man := range m.reg.List(plugin.KindProvider) {
		id := man.Metadata.ID
		if _, ok := m.configs[id]; ok {
			continue
		}
		baseURL, _ := man.Spec["baseURL"].(string)
		local, _ := man.Spec["local"].(bool)
		tier, _ := man.Spec["costTier"].(string)
		driver := "openai-compat"
		if man.Labels != nil && man.Labels["hermes.driver"] != "" {
			driver = man.Labels["hermes.driver"]
		}
		if driver == "echo-provider" || strings.Contains(string(id), "example.echo") || strings.Contains(string(id), "example.budget") {
			driver = "echo-provider"
		}
		m.configs[id] = &Config{
			ID: string(id), Name: man.Metadata.Name, Driver: driver,
			BaseURL: baseURL, Local: local, CostTier: types.CostTier(tier),
			Managed: false, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
			Labels: map[string]string{"source": "bootstrap"},
		}
	}
}

func (m *Manager) List() []Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Config, 0, len(m.configs))
	for _, c := range m.configs {
		cp := *c
		// never expose secrets — only handle
		out = append(out, cp)
	}
	return out
}

func (m *Manager) Get(id types.PluginID) (Config, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.configs[id]
	if !ok {
		return Config{}, false
	}
	return *c, true
}

// Create registers a new provider from template and/or custom fields.
func (m *Manager) Create(ctx context.Context, req CreateRequest) (*Config, error) {
	cfg := Config{
		Managed: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		Driver: "openai-compat", CostTier: types.TierStandard, Labels: map[string]string{"source": "ui"},
	}
	if req.FromTemplate != "" {
		t, ok := TemplateByID(req.FromTemplate)
		if !ok {
			return nil, fmt.Errorf("unknown template %q", req.FromTemplate)
		}
		cfg.TemplateID = t.ID
		cfg.Name = t.Name
		cfg.Driver = t.Driver
		cfg.BaseURL = t.BaseURL
		cfg.Local = t.Local
		cfg.CostTier = t.CostTier
		cfg.DefaultModel = t.DefaultModel
		cfg.Models = append([]string{}, t.SuggestedModels...)
		if len(cfg.Models) == 0 && t.DefaultModel != "" {
			cfg.Models = []string{t.DefaultModel}
		}
	}
	if req.Name != "" {
		cfg.Name = req.Name
	}
	if req.Driver != "" {
		cfg.Driver = req.Driver
	}
	if req.BaseURL != "" {
		cfg.BaseURL = req.BaseURL
	}
	if req.Local != nil {
		cfg.Local = *req.Local
	}
	if req.CostTier != "" {
		cfg.CostTier = req.CostTier
	}
	if req.DefaultModel != "" {
		cfg.DefaultModel = req.DefaultModel
	}
	if len(req.Models) > 0 {
		cfg.Models = req.Models
	}
	if cfg.Name == "" {
		cfg.Name = "Custom provider"
	}
	// ID
	if req.ID != "" {
		cfg.ID = sanitizeID(req.ID)
	} else {
		cfg.ID = "provider.ui." + slug(cfg.Name)
		if cfg.TemplateID != "" {
			cfg.ID = "provider.ui." + cfg.TemplateID
		}
	}
	if !strings.HasPrefix(cfg.ID, "provider.") {
		cfg.ID = "provider." + cfg.ID
	}
	// uniqueness
	if _, _, exists := m.reg.Get(types.PluginID(cfg.ID)); exists {
		cfg.ID = fmt.Sprintf("%s.%d", cfg.ID, time.Now().Unix()%100000)
	}

	// Credential
	if req.CredentialHandle != "" {
		cfg.CredentialHandle = req.CredentialHandle
	} else if req.APIKey != "" && m.creds != nil {
		h, err := m.creds.Put(ctx, cfg.ID, "ui-api-key", types.PluginID(cfg.ID), req.APIKey)
		if err != nil {
			return nil, err
		}
		cfg.CredentialHandle = string(h)
	}

	if err := m.materialize(cfg); err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.configs[types.PluginID(cfg.ID)] = &cfg
	m.mu.Unlock()
	if m.refresh != nil {
		m.refresh()
	}
	cp := cfg
	return &cp, nil
}

// Delete removes a UI-managed provider (refuses unmanaged bootstrap plugins).
func (m *Manager) Delete(id types.PluginID) error {
	m.mu.Lock()
	cfg, ok := m.configs[id]
	if !ok {
		m.mu.Unlock()
		// allow delete if registered but not tracked as managed? only managed
		return fmt.Errorf("unknown managed provider")
	}
	if !cfg.Managed {
		m.mu.Unlock()
		return fmt.Errorf("cannot delete bootstrap provider %s (managed=false)", id)
	}
	delete(m.configs, id)
	m.mu.Unlock()
	if err := m.reg.Unregister(id); err != nil {
		return err
	}
	if m.refresh != nil {
		m.refresh()
	}
	return nil
}

// Update patches a managed provider (baseURL, models, name, key).
func (m *Manager) Update(ctx context.Context, id types.PluginID, req CreateRequest) (*Config, error) {
	m.mu.Lock()
	cfg, ok := m.configs[id]
	if !ok || !cfg.Managed {
		m.mu.Unlock()
		return nil, fmt.Errorf("not a managed provider")
	}
	cp := *cfg
	m.mu.Unlock()

	if req.Name != "" {
		cp.Name = req.Name
	}
	if req.BaseURL != "" {
		cp.BaseURL = req.BaseURL
	}
	if req.Local != nil {
		cp.Local = *req.Local
	}
	if req.CostTier != "" {
		cp.CostTier = req.CostTier
	}
	if req.DefaultModel != "" {
		cp.DefaultModel = req.DefaultModel
	}
	if len(req.Models) > 0 {
		cp.Models = req.Models
	}
	if req.APIKey != "" && m.creds != nil {
		h, err := m.creds.Put(ctx, cp.ID, "ui-api-key", types.PluginID(cp.ID), req.APIKey)
		if err != nil {
			return nil, err
		}
		cp.CredentialHandle = string(h)
	}
	if req.CredentialHandle != "" {
		cp.CredentialHandle = req.CredentialHandle
	}
	cp.UpdatedAt = time.Now().UTC()

	_ = m.reg.Unregister(id)
	if err := m.materialize(cp); err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.configs[id] = &cp
	m.mu.Unlock()
	if m.refresh != nil {
		m.refresh()
	}
	out := cp
	return &out, nil
}

func (m *Manager) materialize(cfg Config) error {
	models := cfg.Models
	if len(models) == 0 && cfg.DefaultModel != "" {
		models = []string{cfg.DefaultModel}
	}
	modelAny := make([]any, 0, len(models))
	for _, mid := range models {
		modelAny = append(modelAny, map[string]any{"id": mid, "costTier": string(cfg.CostTier)})
	}
	if len(modelAny) == 0 {
		modelAny = []any{map[string]any{"id": "default", "costTier": string(cfg.CostTier)}}
	}

	man := plugin.Manifest{
		APIVersion: "hermes.plugin/v1",
		Kind:       plugin.KindProvider,
		Metadata: plugin.Metadata{
			ID: types.PluginID(cfg.ID), Version: "1.0.0", Name: cfg.Name,
		},
		Spec: map[string]any{
			"baseURL":      cfg.BaseURL,
			"local":        cfg.Local,
			"costTier":     string(cfg.CostTier),
			"capabilities": []any{"coding", "tools"},
			"models":       modelAny,
		},
		Labels: map[string]string{
			"hermes.driver":  cfg.Driver,
			"hermes.managed": "true",
			"hermes.template": cfg.TemplateID,
		},
	}

	var inst any
	var err error
	switch cfg.Driver {
	case "echo-provider", "echo":
		inst, err = echo.NewProvider(man)
	case "openai-compat", "":
		inst, err = openaicompat.NewProvider(man)
		if err == nil {
			if p, ok := inst.(*openaicompat.Provider); ok && m.creds != nil {
				p.Resolve = func(ctx context.Context, handle string) (string, error) {
					// Prefer explicit handle from complete; else config handle
					h := handle
					if h == "" {
						h = cfg.CredentialHandle
					}
					if h == "" {
						return "", fmt.Errorf("no credential handle")
					}
					sec, _, e := m.creds.Resolve(ctx, credentials.Handle(h))
					return sec, e
				}
			}
		}
	default:
		return fmt.Errorf("unsupported driver %q (use openai-compat or echo-provider)", cfg.Driver)
	}
	if err != nil {
		return err
	}
	return m.reg.Register(man, inst)
}

var nonAlpha = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonAlpha.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "provider"
	}
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}

func sanitizeID(id string) string {
	id = strings.TrimSpace(id)
	id = strings.ReplaceAll(id, " ", ".")
	return id
}
