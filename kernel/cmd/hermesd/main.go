package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/httpapi"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/kernel"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 2 {
		printBanner()
		fmt.Println("usage: hermesd <command>")
		fmt.Println("  status              print foundation status")
		fmt.Println("  serve [addr]        Host API (default :8080)")
		os.Exit(0)
	}
	switch os.Args[1] {
	case "status":
		runStatus()
	case "serve":
		addr := ":8080"
		if len(os.Args) > 2 {
			addr = os.Args[2]
		}
		if err := runServe(addr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "version", "--version", "-v":
		fmt.Println(httpapi.Version)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(2)
	}
}

func printBanner() {
	fmt.Println("hermesd — Hermes Agent OS kernel")
	fmt.Println("protocol: AESP (upstream, vendor-neutral)")
	fmt.Println("principle: providers ≠ runtimes · everything is a plugin")
}

func newKernel() *kernel.Kernel {
	reg := plugin.NewMemoryRegistry()
	loadExamplePlugins(reg)
	return kernel.New(reg)
}

func runStatus() {
	k := newKernel()
	printBanner()
	fmt.Println("status: H1 Host API")
	fmt.Println("plugins registered:", len(k.Plugins().List("")))
	fmt.Println("endpoints: /api/v1/health /api/v1/missions /api/v1/events /api/v1/registry/*")
}

func runServe(addr string) error {
	k := newKernel()
	if err := k.Health(context.Background()); err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.New(k).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	printBanner()
	fmt.Printf("serving Host API on %s\n", addr)
	fmt.Printf("  GET  /api/v1/health\n")
	fmt.Printf("  GET  /api/v1/missions\n")
	fmt.Printf("  POST /api/v1/missions\n")
	fmt.Printf("  GET  /api/v1/events?since=0&format=json\n")
	fmt.Printf("  WS   /api/v1/events\n")
	fmt.Printf("  GET  /api/v1/registry/{providers|runtimes|tools}\n")
	fmt.Printf("plugins: %d\n", len(k.Plugins().List("")))

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	case <-sig:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		fmt.Println("shutdown")
		return nil
	}
}

// loadExamplePlugins registers example manifests from plugins/ if present,
// otherwise seeds in-process example provider + runtime.
func loadExamplePlugins(reg plugin.Registry) {
	roots := []string{
		"plugins",
		filepath.Join("..", "plugins"),
		filepath.Join("..", "..", "plugins"),
	}
	loaded := 0
	for _, root := range roots {
		n, _ := loadPluginDir(reg, filepath.Join(root, "providers"))
		loaded += n
		n, _ = loadPluginDir(reg, filepath.Join(root, "runtimes"))
		loaded += n
		if loaded > 0 {
			return
		}
	}
	// Seed when no files found (tests / bare binary)
	_ = reg.Register(plugin.Manifest{
		APIVersion: "hermes.plugin/v1",
		Kind:       plugin.KindProvider,
		Metadata:   plugin.Metadata{ID: "provider.example.echo", Version: "0.0.1", Name: "Example Echo Provider"},
		Spec: map[string]any{
			"capabilities": []any{"coding", "tools"},
			"local":        true,
			"costTier":     "free-local",
		},
		Labels: map[string]string{"hermes.example": "true"},
	}, nil)
	_ = reg.Register(plugin.Manifest{
		APIVersion: "hermes.plugin/v1",
		Kind:       plugin.KindRuntime,
		Metadata:   plugin.Metadata{ID: "runtime.example.echo", Version: "0.0.1", Name: "Example Echo Runtime"},
		Spec: map[string]any{
			"sandboxTier": "process-pty",
		},
		Labels: map[string]string{"hermes.example": "true"},
	}, nil)
}

func loadPluginDir(reg plugin.Registry, dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name(), "plugin.yaml")
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var m plugin.Manifest
		if err := yaml.Unmarshal(b, &m); err != nil {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", path, err)
			continue
		}
		if err := reg.Register(m, nil); err != nil {
			fmt.Fprintf(os.Stderr, "warn: register %s: %v\n", path, err)
			continue
		}
		n++
	}
	return n, nil
}
