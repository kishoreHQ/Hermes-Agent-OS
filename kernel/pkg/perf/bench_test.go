package perf

import (
	"context"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/bootstrap"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

func BenchmarkSubmitMission(b *testing.B) {
	res, err := bootstrap.New(bootstrap.Options{SeedBuiltins: true, PluginRoots: []string{"/none"}})
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := res.Kernel.SubmitMission(ctx, host.Mission{
			Goal:         "bench",
			RequiredCaps: []types.Capability{"coding"},
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
