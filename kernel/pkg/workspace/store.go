// Package workspace holds notes, todos, documents, vault entries, uploads, presets.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Note is a lightweight note.
type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Tags      []string  `json:"tags,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedAt time.Time `json:"createdAt"`
}

// Todo item.
type Todo struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Done      bool       `json:"done"`
	Due       *time.Time `json:"due,omitempty"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// Doc is a markdown document artifact.
type Doc struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// VaultEntry is a named secret-free metadata vault record (content is encrypted-at-rest path).
type VaultEntry struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"` // stored in-process; operators should use credentials for secrets
	UpdatedAt time.Time `json:"updatedAt"`
}

// Upload meta.
type Upload struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	MediaType string    `json:"mediaType,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// Preset is a saved mission/chat configuration.
type Preset struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	GoalTemplate   string            `json:"goalTemplate,omitempty"`
	PreferProvider string            `json:"preferProvider,omitempty"`
	PreferModel    string            `json:"preferModel,omitempty"`
	SkillIDs       []string          `json:"skillIds,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
}

// Webhook outbound hook.
type Webhook struct {
	ID     string   `json:"id"`
	URL    string   `json:"url"`
	Events []string `json:"events"` // e.g. mission.completed
	Secret string   `json:"secret,omitempty"`
}

// Store is in-memory + optional disk dir for uploads.
type Store struct {
	mu      sync.Mutex
	notes   map[string]*Note
	todos   map[string]*Todo
	docs    map[string]*Doc
	vault   map[string]*VaultEntry
	uploads map[string]*Upload
	presets map[string]*Preset
	hooks   map[string]*Webhook
	seq     int64
	dataDir string
}

func New(dataDir string) *Store {
	if dataDir == "" {
		dataDir = filepath.Join(os.TempDir(), "hermes-workspace")
	}
	_ = os.MkdirAll(filepath.Join(dataDir, "uploads"), 0o755)
	return &Store{
		notes: map[string]*Note{}, todos: map[string]*Todo{}, docs: map[string]*Doc{},
		vault: map[string]*VaultEntry{}, uploads: map[string]*Upload{},
		presets: map[string]*Preset{}, hooks: map[string]*Webhook{},
		dataDir: dataDir,
	}
}

func (s *Store) nextID(prefix string) string {
	s.seq++
	return fmt.Sprintf("%s_%d", prefix, s.seq)
}

// —— Notes ——

func (s *Store) ListNotes() []Note {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Note, 0, len(s.notes))
	for _, n := range s.notes {
		out = append(out, *n)
	}
	return out
}

func (s *Store) PutNote(n Note) Note {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if n.ID == "" {
		n.ID = s.nextID("note")
		n.CreatedAt = now
	}
	n.UpdatedAt = now
	cp := n
	s.notes[n.ID] = &cp
	return cp
}

func (s *Store) GetNote(id string) (Note, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.notes[id]
	if !ok {
		return Note{}, false
	}
	return *n, true
}

func (s *Store) DeleteNote(id string) { s.mu.Lock(); delete(s.notes, id); s.mu.Unlock() }

// —— Todos ——

func (s *Store) ListTodos() []Todo {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Todo, 0, len(s.todos))
	for _, t := range s.todos {
		out = append(out, *t)
	}
	return out
}

func (s *Store) PutTodo(t Todo) Todo {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t.ID == "" {
		t.ID = s.nextID("todo")
	}
	t.UpdatedAt = time.Now().UTC()
	cp := t
	s.todos[t.ID] = &cp
	return cp
}

func (s *Store) DeleteTodo(id string) { s.mu.Lock(); delete(s.todos, id); s.mu.Unlock() }

// —— Docs ——

func (s *Store) ListDocs() []Doc {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Doc, 0, len(s.docs))
	for _, d := range s.docs {
		out = append(out, *d)
	}
	return out
}

func (s *Store) PutDoc(d Doc) Doc {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d.ID == "" {
		d.ID = s.nextID("doc")
	}
	d.UpdatedAt = time.Now().UTC()
	cp := d
	s.docs[d.ID] = &cp
	return cp
}

func (s *Store) GetDoc(id string) (Doc, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.docs[id]
	if !ok {
		return Doc{}, false
	}
	return *d, true
}

func (s *Store) DeleteDoc(id string) { s.mu.Lock(); delete(s.docs, id); s.mu.Unlock() }

// —— Vault ——

func (s *Store) ListVault() []VaultEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]VaultEntry, 0, len(s.vault))
	for _, v := range s.vault {
		cp := *v
		// never echo full content in list
		if len(cp.Content) > 0 {
			cp.Content = "[redacted — use vault.get]"
		}
		out = append(out, cp)
	}
	return out
}

func (s *Store) PutVault(v VaultEntry) VaultEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v.ID == "" {
		v.ID = s.nextID("vault")
	}
	v.UpdatedAt = time.Now().UTC()
	cp := v
	s.vault[v.ID] = &cp
	return VaultEntry{ID: cp.ID, Name: cp.Name, UpdatedAt: cp.UpdatedAt, Content: "[stored]"}
}

func (s *Store) GetVault(id string) (VaultEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.vault[id]
	if !ok {
		return VaultEntry{}, false
	}
	return *v, true
}

// —— Uploads ——

func (s *Store) SaveUpload(name, mediaType string, data []byte) (Upload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID("up")
	safe := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == 0 {
			return '_'
		}
		return r
	}, name)
	path := filepath.Join(s.dataDir, "uploads", id+"_"+safe)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return Upload{}, err
	}
	u := Upload{ID: id, Name: name, Path: path, Size: int64(len(data)), MediaType: mediaType, CreatedAt: time.Now().UTC()}
	s.uploads[id] = &u
	return u, nil
}

func (s *Store) ListUploads() []Upload {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Upload, 0, len(s.uploads))
	for _, u := range s.uploads {
		out = append(out, *u)
	}
	return out
}

// —— Presets / webhooks ——

func (s *Store) ListPresets() []Preset {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Preset, 0, len(s.presets))
	for _, p := range s.presets {
		out = append(out, *p)
	}
	return out
}

func (s *Store) PutPreset(p Preset) Preset {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ID == "" {
		p.ID = s.nextID("preset")
	}
	cp := p
	s.presets[p.ID] = &cp
	return cp
}

func (s *Store) ListWebhooks() []Webhook {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Webhook, 0, len(s.hooks))
	for _, h := range s.hooks {
		out = append(out, *h)
	}
	return out
}

func (s *Store) PutWebhook(h Webhook) Webhook {
	s.mu.Lock()
	defer s.mu.Unlock()
	if h.ID == "" {
		h.ID = s.nextID("hook")
	}
	cp := h
	s.hooks[h.ID] = &cp
	return cp
}

func (s *Store) DeleteWebhook(id string) { s.mu.Lock(); delete(s.hooks, id); s.mu.Unlock() }

// Snapshot for backup.
func (s *Store) Snapshot() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]any{
		"notes": s.notes, "todos": s.todos, "docs": s.docs,
		"vault": s.vault, "uploads": s.uploads, "presets": s.presets, "webhooks": s.hooks,
	}
}

// SearchNotes simple contains.
func (s *Store) SearchNotes(q string) []Note {
	q = strings.ToLower(q)
	var out []Note
	for _, n := range s.ListNotes() {
		if strings.Contains(strings.ToLower(n.Title+" "+n.Body), q) {
			out = append(out, n)
		}
	}
	return out
}

func (s *Store) DataDir() string { return s.dataDir }
