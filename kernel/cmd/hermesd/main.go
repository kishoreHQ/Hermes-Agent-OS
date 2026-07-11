package main

import (
	"fmt"
	"os"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/kernel"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
)

func main() {
	reg := plugin.NewMemoryRegistry()
	k := kernel.New(reg)
	if err := k.Health(nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("hermesd — Hermes Agent OS kernel")
	fmt.Println("status: foundation skeleton (product platform)")
	fmt.Println("protocol: AESP (upstream, vendor-neutral)")
	fmt.Println("plugins registered:", len(reg.List("")))
	fmt.Println("principle: providers ≠ runtimes · everything is a plugin")
}
