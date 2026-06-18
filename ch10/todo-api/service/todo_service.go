package service

import (
	"errors"
	"strings"
	"sync"
	"time"
	"todo-api/models"
)

// Sentinel-ошибки — удобно проверять через errors.Is().
var (
	ErrNotFound     = errors.New("not found")
	ErrTitleEmpty   = errors.New("title required")
	ErrLimitReached = errors.New("limit reached")
)

// CreateInput — входные данные для создания задачи.
type CreateInput struct {
	Title string
	Body  string
}

// UpdateInput — поля обновления (nil = не трогать поле).
type UpdateInput struct {
	Title *string
	Body  *string
}

// TodoService хранит задачи в памяти и управляет ими.
type TodoService struct {
	mu     sync.Mutex
	items  map[int64]models.Todo
	nextID int64
	limit  int // 0 = без ограничений
}

func New(limit int) *TodoService {
	return &TodoService{
		items:  make(map[int64]models.Todo),
		nextID: 1,
		limit:  limit,
	}
}

// Add создаёт новую задачу и возвращает её.
func (s *TodoService) Add(in CreateInput) (models.Todo, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return models.Todo{}, ErrTitleEmpty
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.limit > 0 && len(s.items) >= s.limit {
		return models.Todo{}, ErrLimitReached
	}

	now := time.Now().UTC()
	t := models.Todo{
		ID:        s.nextID,
		Title:     title,
		Body:      strings.TrimSpace(in.Body),
		Done:      false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.items[s.nextID] = t
	s.nextID++
	return t, nil
}

// Get возвращает задачу по ID.
func (s *TodoService) Get(id int64) (models.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.items[id]
	if !ok {
		return models.Todo{}, ErrNotFound
	}
	return t, nil
}

// List возвращает все задачи; doneFilter == nil → все.
func (s *TodoService) List(doneFilter *bool) []models.Todo {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]models.Todo, 0, len(s.items))
	for _, t := range s.items {
		if doneFilter != nil && t.Done != *doneFilter {
			continue
		}
		out = append(out, t)
	}
	return out
}

// Update изменяет title/body задачи (nil = не трогать).
func (s *TodoService) Update(id int64, in UpdateInput) (models.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.items[id]
	if !ok {
		return models.Todo{}, ErrNotFound
	}

	changed := false
	if in.Title != nil {
		nt := strings.TrimSpace(*in.Title)
		if nt == "" {
			return models.Todo{}, ErrTitleEmpty
		}
		t.Title = nt
		changed = true
	}
	if in.Body != nil {
		t.Body = strings.TrimSpace(*in.Body)
		changed = true
	}
	if changed {
		t.UpdatedAt = time.Now().UTC()
		s.items[id] = t
	}
	return t, nil
}

// Delete удаляет задачу по ID.
func (s *TodoService) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return ErrNotFound
	}
	delete(s.items, id)
	return nil
}

// Toggle переключает Done.
func (s *TodoService) Toggle(id int64) (models.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.items[id]
	if !ok {
		return models.Todo{}, ErrNotFound
	}
	t.Done = !t.Done
	t.UpdatedAt = time.Now().UTC()
	s.items[id] = t
	return t, nil
}

// Clear удаляет все задачи и сбрасывает счётчик ID.
func (s *TodoService) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make(map[int64]models.Todo)
	s.nextID = 1
}
