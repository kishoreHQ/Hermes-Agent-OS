// Package artifact is content-addressed artifact storage (AESP-0007 / CG-ARTIFACT).
package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Meta describes a stored blob.
type Meta struct {
	Digest    types.ArtifactDigest `json:"digest"`
	MediaType string               `json:"mediaType,omitempty"`
	Size      int64                `json:"size"`
	MissionID types.MissionID      `json:"missionId,omitempty"`
	Labels    map[string]string    `json:"labels,omitempty"`
	CreatedAt time.Time            `json:"createdAt"`
}

type entry struct {
	meta Meta
	data []byte
}

// Store is an in-process content-addressed store.
type Store struct {
	mu   sync.RWMutex
	byD  map[types.ArtifactDigest]entry
}

func New() *Store {
	return &Store{byD: map[types.ArtifactDigest]entry{}}
}

// Put stores bytes; digest is sha256 hex of content (sha256:<hex>).
func (s *Store) Put(ctx context.Context, data []byte, mediaType string, mission types.MissionID, labels map[string]string) (Meta, error) {
	if len(data) == 0 {
		return Meta{}, fmt.Errorf("empty artifact")
	}
	sum := sha256.Sum256(data)
	digest := types.ArtifactDigest("sha256:" + hex.EncodeToString(sum[:]))
	m := Meta{
		Digest: digest, MediaType: mediaType, Size: int64(len(data)),
		MissionID: mission, Labels: labels, CreatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	s.byD[digest] = entry{meta: m, data: append([]byte(nil), data...)}
	s.mu.Unlock()
	return m, nil
}

func (s *Store) Get(ctx context.Context, digest types.ArtifactDigest) ([]byte, Meta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.byD[digest]
	if !ok {
		return nil, Meta{}, fmt.Errorf("not found")
	}
	return append([]byte(nil), e.data...), e.meta, nil
}

func (s *Store) Meta(ctx context.Context, digest types.ArtifactDigest) (Meta, error) {
	_, m, err := s.Get(ctx, digest)
	return m, err
}

func (s *Store) List(ctx context.Context, mission types.MissionID) []Meta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Meta
	for _, e := range s.byD {
		if mission == "" || e.meta.MissionID == mission {
			out = append(out, e.meta)
		}
	}
	return out
}
