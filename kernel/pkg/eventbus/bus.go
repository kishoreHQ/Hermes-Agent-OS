// Package eventbus provides a monotonic event journal for host clients (INV-10).
// Seq is global and increasing — required for WS reconnect catch-up (UI-RT-01).
package eventbus

import (
	"context"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Event is a journaled control-plane event.
type Event struct {
	Seq       int64            `json:"seq"`
	Type      string           `json:"type"`
	ID        string           `json:"id,omitempty"`
	MissionID types.MissionID  `json:"missionId,omitempty"`
	Time      time.Time        `json:"ts"`
	Data      map[string]any   `json:"data,omitempty"`
}

// Bus is the publish/subscribe journal surface.
type Bus interface {
	Publish(ctx context.Context, e Event) error
	Subscribe(ctx context.Context, missionFilter string) (<-chan Event, error)
	Replay(ctx context.Context, missionID types.MissionID) ([]Event, error)
	Since(ctx context.Context, since int64) ([]Event, error)
	Seq() int64
}

// MemoryBus is an in-process Bus implementation.
type MemoryBus struct {
	mu   sync.Mutex
	log  []Event
	subs map[string][]chan Event
	seq  int64
}

func NewMemoryBus() *MemoryBus {
	return &MemoryBus{subs: make(map[string][]chan Event)}
}

func (b *MemoryBus) Publish(ctx context.Context, e Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seq++
	e.Seq = b.seq
	if e.ID == "" {
		e.ID = time.Now().UTC().Format("20060102T150405.000000000")
	}
	if e.Time.IsZero() {
		e.Time = time.Now().UTC()
	}
	b.log = append(b.log, e)
	key := string(e.MissionID)
	for _, ch := range b.subs[key] {
		select {
		case ch <- e:
		default:
		}
	}
	for _, ch := range b.subs[""] {
		select {
		case ch <- e:
		default:
		}
	}
	return nil
}

func (b *MemoryBus) Subscribe(ctx context.Context, missionFilter string) (<-chan Event, error) {
	ch := make(chan Event, 128)
	b.mu.Lock()
	b.subs[missionFilter] = append(b.subs[missionFilter], ch)
	b.mu.Unlock()
	go func() {
		<-ctx.Done()
		b.mu.Lock()
		defer b.mu.Unlock()
		subs := b.subs[missionFilter]
		for i, c := range subs {
			if c == ch {
				b.subs[missionFilter] = append(subs[:i], subs[i+1:]...)
				close(ch)
				break
			}
		}
	}()
	return ch, nil
}

func (b *MemoryBus) Replay(ctx context.Context, missionID types.MissionID) ([]Event, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []Event
	for _, e := range b.log {
		if e.MissionID == missionID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (b *MemoryBus) Since(ctx context.Context, since int64) ([]Event, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []Event
	for _, e := range b.log {
		if e.Seq > since {
			out = append(out, e)
		}
	}
	return out, nil
}

func (b *MemoryBus) Seq() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.seq
}
