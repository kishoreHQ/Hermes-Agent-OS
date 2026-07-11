package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/security"
	"gopkg.in/yaml.v3"
)

// LoadResult is one successfully loaded plugin.
type LoadResult struct {
	Path     string
	Manifest Manifest
	Instance any
}

// Loader discovers plugin.yaml files and materializes instances via FactoryRegistry.
type Loader struct {
	Factories *FactoryRegistry
}

func NewLoader(factories *FactoryRegistry) *Loader {
	if factories == nil {
		factories = NewFactoryRegistry()
	}
	return &Loader{Factories: factories}
}

// Discover walks root for **/plugin.yaml (and plugin.yml).
func (l *Loader) Discover(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip unreadable dirs
			if d != nil && d.IsDir() {
				return nil
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		base := strings.ToLower(d.Name())
		if base == "plugin.yaml" || base == "plugin.yml" {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

// LoadFile parses one manifest and constructs an instance.
func (l *Loader) LoadFile(path string) (LoadResult, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return LoadResult{}, err
	}
	var m Manifest
	if err := yaml.Unmarshal(b, &m); err != nil {
		return LoadResult{}, fmt.Errorf("%s: %w", path, err)
	}
	if m.APIVersion == "" {
		return LoadResult{}, fmt.Errorf("%s: apiVersion required", path)
	}
	if m.Kind == "" {
		return LoadResult{}, fmt.Errorf("%s: kind required", path)
	}
	if m.Metadata.ID == "" {
		return LoadResult{}, fmt.Errorf("%s: metadata.id required", path)
	}
	// Optional / required HMAC signature (H5)
	sig := ""
	if m.Labels != nil {
		sig = m.Labels["hermes.signature"]
	}
	if err := security.VerifyPluginIdentity(m.APIVersion, string(m.Kind), string(m.Metadata.ID), m.Metadata.Version, sig); err != nil {
		return LoadResult{}, fmt.Errorf("%s: %w", path, err)
	}
	inst, err := l.Factories.Create(m)
	if err != nil {
		return LoadResult{}, fmt.Errorf("%s: %w", path, err)
	}
	return LoadResult{Path: path, Manifest: m, Instance: inst}, nil
}

// LoadTree discovers and loads all plugins under root into reg.
// Continues on per-plugin errors; returns count loaded and multi-error string if any.
func (l *Loader) LoadTree(root string, reg Registry) (loaded int, results []LoadResult, err error) {
	paths, err := l.Discover(root)
	if err != nil {
		return 0, nil, err
	}
	var errs []string
	for _, p := range paths {
		res, e := l.LoadFile(p)
		if e != nil {
			errs = append(errs, e.Error())
			continue
		}
		if e := reg.Register(res.Manifest, res.Instance); e != nil {
			errs = append(errs, fmt.Sprintf("%s: register: %v", p, e))
			continue
		}
		results = append(results, res)
		loaded++
	}
	if len(errs) > 0 {
		return loaded, results, fmt.Errorf("plugin load errors (%d): %s", len(errs), strings.Join(errs, "; "))
	}
	return loaded, results, nil
}

// FindPluginRoots returns candidate plugin roots relative to cwd and env HERMES_PLUGINS.
func FindPluginRoots() []string {
	var roots []string
	if e := os.Getenv("HERMES_PLUGINS"); e != "" {
		for _, p := range strings.Split(e, string(os.PathListSeparator)) {
			if p != "" {
				roots = append(roots, p)
			}
		}
	}
	// Common layouts when running from repo root or kernel/
	candidates := []string{
		"plugins",
		filepath.Join("..", "plugins"),
		filepath.Join("..", "..", "plugins"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			roots = append(roots, c)
		}
	}
	return uniq(roots)
}

func uniq(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		abs, err := filepath.Abs(s)
		if err != nil {
			abs = s
		}
		if seen[abs] {
			continue
		}
		seen[abs] = true
		out = append(out, s)
	}
	return out
}
