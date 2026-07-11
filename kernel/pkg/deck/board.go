package deck

import (
	"fmt"
	"sync"
	"time"
)

type Column string

const (
	ColBacklog    Column = "backlog"
	ColQueued     Column = "queued"
	ColInProgress Column = "in_progress"
	ColReview     Column = "review"
	ColDone       Column = "done"
)

type Task struct {
	ID           string    `json:"id"`
	BoardID      string    `json:"boardId"`
	Title        string    `json:"title"`
	Column       Column    `json:"column"`
	Assignee     string    `json:"assignee,omitempty"`
	Capabilities []string  `json:"capabilities,omitempty"`
	MissionID    string    `json:"missionId,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Board struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Tasks []Task `json:"tasks,omitempty"`
}

// BoardService is K4 kanban.
type BoardService struct {
	mu     sync.Mutex
	boards map[string]*Board
	tasks  map[string]*Task
}

func NewBoard() *BoardService {
	s := &BoardService{boards: map[string]*Board{}, tasks: map[string]*Task{}}
	b := &Board{ID: "board_default", Name: "Command Deck"}
	s.boards[b.ID] = b
	s.addTask(b.ID, "Connect provider plugin", ColQueued, []string{"coding"})
	s.addTask(b.ID, "Review mission replay", ColReview, []string{"tools"})
	s.addTask(b.ID, "Triage fleet health", ColBacklog, []string{"reasoning"})
	return s
}

func (s *BoardService) addTask(boardID, title string, col Column, caps []string) *Task {
	t := &Task{
		ID: fmt.Sprintf("task_%d", time.Now().UnixNano()), BoardID: boardID, Title: title,
		Column: col, Capabilities: caps,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	s.tasks[t.ID] = t
	return t
}

func (s *BoardService) ListBoards() []Board {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Board, 0, len(s.boards))
	for _, b := range s.boards {
		bb := *b
		bb.Tasks = s.tasksFor(b.ID)
		out = append(out, bb)
	}
	return out
}

func (s *BoardService) tasksFor(boardID string) []Task {
	var out []Task
	for _, t := range s.tasks {
		if t.BoardID == boardID {
			out = append(out, *t)
		}
	}
	return out
}

func (s *BoardService) ListTasks() []Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, *t)
	}
	return out
}

type CreateTaskRequest struct {
	BoardID      string   `json:"boardId"`
	Title        string   `json:"title"`
	Column       string   `json:"column,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

func (s *BoardService) CreateTask(req CreateTaskRequest) (*Task, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title required")
	}
	if req.BoardID == "" {
		req.BoardID = "board_default"
	}
	col := Column(req.Column)
	if col == "" {
		col = ColBacklog
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.boards[req.BoardID]; !ok {
		return nil, fmt.Errorf("unknown board")
	}
	t := s.addTask(req.BoardID, req.Title, col, req.Capabilities)
	cp := *t
	return &cp, nil
}

func (s *BoardService) MoveTask(id string, col Column) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("unknown task")
	}
	t.Column = col
	t.UpdatedAt = time.Now().UTC()
	cp := *t
	return &cp, nil
}

func (s *BoardService) ClaimTask(id, assignee string) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("unknown task")
	}
	t.Assignee = assignee
	if t.Column == ColBacklog || t.Column == ColQueued {
		t.Column = ColInProgress
	}
	t.UpdatedAt = time.Now().UTC()
	cp := *t
	return &cp, nil
}
