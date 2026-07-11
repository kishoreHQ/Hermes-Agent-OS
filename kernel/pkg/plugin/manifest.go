// Package plugin is the universal discovery contract (INV-02).
// Everything is a plugin. Kernel never hardcodes vendors.
package plugin

import "github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"

// Kind enumerates first-class plugin classes.
type Kind string

const (
	KindProvider   Kind = "provider"
	KindRuntime    Kind = "runtime"
	KindTool       Kind = "tool"
	KindChannel    Kind = "channel"
	KindMemory     Kind = "memory"
	KindKnowledge  Kind = "knowledge"
	KindWorkflow   Kind = "workflow"
	KindPolicy     Kind = "policy"
	KindSecurity   Kind = "security"
	KindEvaluation Kind = "evaluation"
	KindStorage    Kind = "storage"
	KindCredential Kind = "credential"
)

// Manifest is the common header for all plugins (YAML/JSON on disk).
type Manifest struct {
	APIVersion string            `yaml:"apiVersion" json:"apiVersion"`
	Kind       Kind              `yaml:"kind" json:"kind"`
	Metadata   Metadata          `yaml:"metadata" json:"metadata"`
	Spec       map[string]any    `yaml:"spec" json:"spec"`
	Labels     map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

type Metadata struct {
	ID      types.PluginID `yaml:"id" json:"id"`
	Version string         `yaml:"version" json:"version"`
	Name    string         `yaml:"name,omitempty" json:"name,omitempty"`
}

// Registry is a dynamic plugin registry. Zero vendor names in core.
type Registry interface {
	Register(m Manifest, instance any) error
	Get(id types.PluginID) (Manifest, any, bool)
	List(kind Kind) []Manifest
}
