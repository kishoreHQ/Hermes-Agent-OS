// Package skills loads reusable agent skill markdown (prompt packs).
package skills

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Skill is a reusable instruction pack.
type Skill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Body        string `json:"body"`
	Source      string `json:"source,omitempty"` // path or builtin
}

// Store holds skills from disk + builtins.
type Store struct {
	mu     sync.RWMutex
	byID   map[string]Skill
	root   string
}

func New(root string) *Store {
	s := &Store{byID: map[string]Skill{}, root: root}
	s.loadBuiltins()
	if root != "" {
		_ = s.LoadDir(root)
	}
	return s
}

func (s *Store) loadBuiltins() {
	builtins := []Skill{
		{
			ID: "research", Name: "Deep Research", Description: "Multi-step web research",
			Body: `When researching:
1. Call research.outline with the topic
2. Use web.search for queries
3. Use web.fetch on best sources
4. Write findings with citations
5. End with executive summary + open questions`,
			Source: "builtin",
		},
		{
			ID: "coding", Name: "Coding", Description: "Software engineering defaults",
			Body: `Prefer reading files before editing. Use fs.list/fs.read/shell.exec.
Write minimal diffs. Run tests when possible. Explain failures clearly.`,
			Source: "builtin",
		},
		{
			ID: "ops", Name: "Ops", Description: "Safe operational procedures",
			Body: `Never run destructive commands without confirmation.
Prefer read-only diagnostics first. Capture command output for the user.`,
			Source: "builtin",
		},
	}
	for _, sk := range builtins {
		s.byID[sk.ID] = sk
	}
}

// LoadDir loads *.md skills from a directory (name from filename).
func (s *Store) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".md")
		name, body := parseSkillMD(string(b), id)
		s.byID[id] = Skill{ID: id, Name: name, Body: body, Source: p}
	}
	return nil
}

func parseSkillMD(raw, fallbackID string) (name, body string) {
	name = fallbackID
	lines := strings.Split(raw, "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "# ") {
		name = strings.TrimSpace(strings.TrimPrefix(lines[0], "# "))
		body = strings.TrimSpace(strings.Join(lines[1:], "\n"))
		return
	}
	return name, strings.TrimSpace(raw)
}

func (s *Store) List() []Skill {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Skill, 0, len(s.byID))
	for _, sk := range s.byID {
		cp := sk
		// don't dump huge bodies in list
		if len(cp.Body) > 200 {
			cp.Body = cp.Body[:200] + "…"
		}
		out = append(out, cp)
	}
	return out
}

func (s *Store) Get(id string) (Skill, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sk, ok := s.byID[id]
	return sk, ok
}

// Put adds/updates a skill.
func (s *Store) Put(sk Skill) {
	if sk.ID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[sk.ID] = sk
}

// Compose injects selected skills into a prompt block (context budget aware).
func (s *Store) Compose(ids []string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 4000
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var b strings.Builder
	for _, id := range ids {
		sk, ok := s.byID[id]
		if !ok {
			continue
		}
		chunk := "## " + sk.Name + "\n" + sk.Body + "\n\n"
		if b.Len()+len(chunk) > maxChars {
			break
		}
		b.WriteString(chunk)
	}
	// If no ids, include coding+research briefly
	if b.Len() == 0 {
		for _, id := range []string{"coding", "research"} {
			if sk, ok := s.byID[id]; ok {
				chunk := "## " + sk.Name + "\n" + sk.Body + "\n\n"
				if b.Len()+len(chunk) > maxChars {
					break
				}
				b.WriteString(chunk)
			}
		}
	}
	return strings.TrimSpace(b.String())
}
