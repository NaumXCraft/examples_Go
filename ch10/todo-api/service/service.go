package service

import (
	"src/model"
	"src/store"
)

// Package service содержит бизнес-логику приложения.
// Сейчас слой тонкий — просто делегирует в store.
// Сюда добавляют сложные правила: валидация, уведомления, транзакции и т.д.

// TodoService — слой бизнес-логики между handler и store.
type TodoService struct {
	repo store.TodoRepository
}

func NewTodoService(repo store.TodoRepository) *TodoService {
	return &TodoService{repo: repo}
}

func (s *TodoService) Create(title, body string) (model.Todo, error) {
	return s.repo.Add(title, body)
}

func (s *TodoService) GetByID(id int64) (model.Todo, error) {
	return s.repo.Get(id)
}

func (s *TodoService) List(done *bool) []model.Todo {
	return s.repo.List(done)
}

func (s *TodoService) Update(id int64, title, body *string) (model.Todo, error) {
	return s.repo.Update(id, title, body)
}

func (s *TodoService) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *TodoService) Toggle(id int64) (model.Todo, error) {
	return s.repo.Toggle(id)
}

func (s *TodoService) Clear() {
	s.repo.Clear()
}
