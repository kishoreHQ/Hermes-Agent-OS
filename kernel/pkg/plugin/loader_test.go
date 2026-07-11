package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTree(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "providers", "echo")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `
apiVersion: hermes.plugin/v1
kind: provider
metadata:
  id: provider.test.echo
  version: 0.0.1
  name: Test
labels:
  hermes.driver: test-driver
spec:
  capabilities: [coding]
`
	if err := os.WriteFile(filepath.Join(sub, "plugin.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	factories := NewFactoryRegistry()
	factories.Register("test-driver", func(m Manifest) (any, error) {
		return "instance:" + string(m.Metadata.ID), nil
	})
	loader := NewLoader(factories)
	reg := NewMemoryRegistry()
	n, results, err := loader.LoadTree(dir, reg)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || len(results) != 1 {
		t.Fatalf("n=%d results=%d", n, len(results))
	}
	_, inst, ok := reg.Get("provider.test.echo")
	if !ok || inst != "instance:provider.test.echo" {
		t.Fatalf("%v %v", ok, inst)
	}
}

func TestLoadFile_MissingFactory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.yaml")
	_ = os.WriteFile(path, []byte(`
apiVersion: hermes.plugin/v1
kind: provider
metadata:
  id: provider.orphan
  version: 0.0.1
`), 0o644)
	loader := NewLoader(NewFactoryRegistry())
	_, err := loader.LoadFile(path)
	if err == nil {
		t.Fatal("expected error")
	}
}
