// Package persist provides durable JSON file storage for missions, memory, MCP, chat.
package persist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Store is a simple JSON document store under a data directory.
type Store struct {
	mu  sync.Mutex
	dir string
}

func New(dir string) (*Store, error) {
	if dir == "" {
		dir = os.Getenv("HERMES_DATA_DIR")
	}
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "hermes-data")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Dir() string { return s.dir }

func (s *Store) path(name string) string {
	return filepath.Join(s.dir, name+".json")
}

// Save marshals v to name.json.
func (s *Store) Save(name string, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path(name) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path(name))
}

// Load unmarshals name.json into v. Missing file is not an error.
func (s *Store) Load(name string, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(s.path(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(b, v)
}
