// Package service содержит бизнес-логику: хранение задач и операции над ними.
// Никакого HTTP здесь нет — этот пакет ничего не знает о Gin или JSON.
// Это и есть смысл архитектуры: handler работает с веб-запросами,
// service работает с самими задачами.
package service

import (
	"errors"
	"strings"
	"sync"
	"time"

	"todo-api/models"
)

// TodoService хранит задачи в памяти и предоставляет методы для работы с ними.
//
// Поля:
//   - items  — сами задачи, ключ map это ID
//   - nextID — следующий свободный ID (растёт с каждой новой задачей)
//   - mu     — мьютекс. Защищает items и nextID, если несколько запросов
//     придут одновременно (Gin обрабатывает запросы в разных горутинах).
type TodoService struct {
	mu     sync.Mutex
	items  map[int64]models.Todo
	nextID int64
}

// New создаёт пустой сервис, готовый к работе.
func New() *TodoService {
	return &TodoService{
		items:  make(map[int64]models.Todo),
		nextID: 1,
	}
}

// Add создаёт новую задачу.
// title — обязателен (после обрезки пробелов не должен быть пустым).
// body  — опционален.
func (s *TodoService) Add(title, body string) (models.Todo, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return models.Todo{}, errors.New("title required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	todo := models.Todo{
		ID:        s.nextID,
		Title:     title,
		Body:      strings.TrimSpace(body),
		Done:      false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.items[s.nextID] = todo
	s.nextID++

	return todo, nil
}

// Get возвращает задачу по ID или ошибку "not found".
func (s *TodoService) Get(id int64) (models.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo, ok := s.items[id]
	if !ok {
		return models.Todo{}, errors.New("not found")
	}
	return todo, nil
}

// List возвращает задачи.
// filter == nil  → вернуть все задачи
// *filter == true  → только выполненные (done = true)
// *filter == false → только невыполненные (done = false)
func (s *TodoService) List(filter *bool) []models.Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := []models.Todo{} // не nil — чтобы в JSON всегда был [], а не null
	for _, todo := range s.items {
		if filter != nil && todo.Done != *filter {
			continue
		}
		result = append(result, todo)
	}
	return result
}

// Update меняет title и/или body существующей задачи.
// nil-параметр означает "не менять это поле".
func (s *TodoService) Update(id int64, title, body *string) (models.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo, ok := s.items[id]
	if !ok {
		return models.Todo{}, errors.New("not found")
	}

	if title != nil {
		newTitle := strings.TrimSpace(*title)
		if newTitle == "" {
			return models.Todo{}, errors.New("title required")
		}
		todo.Title = newTitle
	}
	if body != nil {
		todo.Body = strings.TrimSpace(*body)
	}

	todo.UpdatedAt = time.Now().UTC()
	s.items[id] = todo

	return todo, nil
}

// Delete удаляет задачу по ID.
func (s *TodoService) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return errors.New("not found")
	}
	delete(s.items, id)
	return nil
}

// Toggle переключает статус Done: false → true → false → ...
func (s *TodoService) Toggle(id int64) (models.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo, ok := s.items[id]
	if !ok {
		return models.Todo{}, errors.New("not found")
	}

	todo.Done = !todo.Done
	todo.UpdatedAt = time.Now().UTC()
	s.items[id] = todo

	return todo, nil
}

// Clear удаляет все задачи и сбрасывает счётчик ID обратно на 1.
func (s *TodoService) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = make(map[int64]models.Todo)
	s.nextID = 1
}
