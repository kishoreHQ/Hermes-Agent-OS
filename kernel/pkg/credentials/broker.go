// Package credentials is the unified credential broker (INV-07).
// Secrets never leave the broker; runtimes receive opaque handles only.
package credentials

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Handle is an opaque reference to a stored secret.
type Handle string

// Record is metadata about a stored credential (never includes secret material in List).
type Record struct {
	Handle    Handle         `json:"handle"`
	Scope     string         `json:"scope"` // e.g. provider id or "platform"
	Label     string         `json:"label"`
	PluginID  types.PluginID `json:"pluginId,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

// Broker issues and resolves credential handles.
type Broker interface {
	// Put stores secret material and returns a handle.
	Put(ctx context.Context, scope, label string, pluginID types.PluginID, secret string) (Handle, error)
	// Resolve returns secret material for an authorized internal caller only.
	// Host/API layers must never expose this.
	Resolve(ctx context.Context, h Handle) (secret string, rec Record, err error)
	// List returns metadata without secrets.
	List(ctx context.Context) ([]Record, error)
	// FindByPlugin returns a preferred handle for a plugin id.
	FindByPlugin(ctx context.Context, pluginID types.PluginID) (Handle, bool)
	// Revoke invalidates a handle.
	Revoke(ctx context.Context, h Handle) error
}

type entry struct {
	rec    Record
	secret string
}

// MemoryBroker is an in-process broker (dev/test). Production will use storage/credential plugins.
type MemoryBroker struct {
	mu   sync.Mutex
	byH  map[Handle]entry
}

func NewMemoryBroker() *MemoryBroker {
	return &MemoryBroker{byH: map[Handle]entry{}}
}

func (b *MemoryBroker) Put(ctx context.Context, scope, label string, pluginID types.PluginID, secret string) (Handle, error) {
	if secret == "" {
		return "", fmt.Errorf("secret required")
	}
	h, err := newHandle()
	if err != nil {
		return "", err
	}
	rec := Record{
		Handle: h, Scope: scope, Label: label, PluginID: pluginID, CreatedAt: time.Now().UTC(),
	}
	b.mu.Lock()
	b.byH[h] = entry{rec: rec, secret: secret}
	b.mu.Unlock()
	return h, nil
}

func (b *MemoryBroker) Resolve(ctx context.Context, h Handle) (string, Record, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	e, ok := b.byH[h]
	if !ok {
		return "", Record{}, fmt.Errorf("unknown handle")
	}
	return e.secret, e.rec, nil
}

func (b *MemoryBroker) List(ctx context.Context) ([]Record, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Record, 0, len(b.byH))
	for _, e := range b.byH {
		out = append(out, e.rec)
	}
	return out, nil
}

// FindByPlugin returns the first handle for a plugin (prefer env/operator keys over demos).
func (b *MemoryBroker) FindByPlugin(ctx context.Context, pluginID types.PluginID) (Handle, bool) {
	list, _ := b.List(ctx)
	var demo Handle
	for _, rec := range list {
		if rec.PluginID != pluginID {
			continue
		}
		if rec.Label != "mission" && rec.Label != "demo" {
			return rec.Handle, true
		}
		demo = rec.Handle
	}
	if demo != "" {
		return demo, true
	}
	return "", false
}

func (b *MemoryBroker) Revoke(ctx context.Context, h Handle) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.byH[h]; !ok {
		return fmt.Errorf("unknown handle")
	}
	delete(b.byH, h)
	return nil
}

func newHandle() (Handle, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return Handle("cred_" + hex.EncodeToString(b[:])), nil
}
