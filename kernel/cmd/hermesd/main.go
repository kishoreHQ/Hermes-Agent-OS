package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/bootstrap"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/hardening"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/httpapi"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/interchange"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
)

func main() {
	if len(os.Args) < 2 {
		printBanner()
		fmt.Println("usage: hermesd <command>")
		fmt.Println("  status              print platform status")
		fmt.Println("  serve [addr]        Host API (default :8080)")
		fmt.Println("  prove-h4            interchangeability proof (H4)")
		fmt.Println("  prove-h5            production hardening proof (H5)")
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
	case "prove-h4":
		if err := runProveH4(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "prove-h5":
		if err := runProveH5(); err != nil {
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

func boot() (*bootstrap.Result, error) {
	return bootstrap.New(bootstrap.Options{SeedBuiltins: true})
}

func runStatus() {
	res, err := boot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	printBanner()
	fmt.Println("status: H5 production hardening")
	fmt.Println("plugins registered:", len(res.Registry.List("")))
	fmt.Printf("  providers: %d  runtimes: %d\n",
		len(res.Registry.List(plugin.KindProvider)),
		len(res.Registry.List(plugin.KindRuntime)))
	fmt.Println("loaded from disk/seed:", res.Loaded)
	if res.LoadWarnings != "" {
		fmt.Println("load notes:", res.LoadWarnings)
	}
	fmt.Println("policy:", res.Kernel.Policy().ID)
	fmt.Println("endpoints: /api/v1/health /api/v1/missions /api/v1/events")
	fmt.Println("           /api/v1/registry/* /api/v1/memory/search /api/v1/credentials")
	fmt.Println("           /api/v1/security/posture /api/v1/policies")
	fmt.Println("proof: hermesd prove-h4 | prove-h5")
}

func runProveH4() error {
	printBanner()
	rep, err := interchange.Run(context.Background())
	if err != nil {
		return err
	}
	fmt.Print(interchange.Format(rep))
	if rep.Failed > 0 || rep.Passed == 0 {
		return fmt.Errorf("H4 proof failed")
	}
	return nil
}

func runProveH5() error {
	printBanner()
	rep, err := hardening.Run(context.Background())
	if err != nil {
		return err
	}
	fmt.Print(hardening.Format(rep))
	if !rep.Passed {
		return fmt.Errorf("H5 proof failed")
	}
	return nil
}

func runServe(addr string) error {
	res, err := boot()
	if err != nil {
		return err
	}
	k := res.Kernel
	if err := k.Health(context.Background()); err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.New(k).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	printBanner()
	fmt.Printf("serving Host API on %s (H5)\n", addr)
	fmt.Printf("  plugins: %d (disk/seed loaded=%d)\n", len(k.Plugins().List("")), res.Loaded)
	if res.LoadWarnings != "" {
		fmt.Printf("  load notes: %s\n", res.LoadWarnings)
	}
	fmt.Printf("  policy: %s (modes full|assist|observe)\n", k.Policy().ID)
	fmt.Printf("  GET  /api/v1/health\n")
	fmt.Printf("  POST /api/v1/missions   → security → route → runtime → memory\n")
	fmt.Printf("  GET  /api/v1/events?since=0&format=json\n")
	fmt.Printf("  WS   /api/v1/events\n")
	fmt.Printf("  GET  /api/v1/registry/{providers|runtimes|tools}\n")
	fmt.Printf("  GET  /api/v1/memory/search?q=\n")
	fmt.Printf("  GET  /api/v1/credentials  (handles only)\n")
	fmt.Printf("  GET  /api/v1/security/posture\n")
	fmt.Printf("  GET  /api/v1/policies\n")

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
