package store

import (
	"errors"
	"strings"
	"sync"
	"time"

	"src/model"
)

// Package store отвечает за хранение и получение данных.
// Сейчас реализация in-memory (в памяти), но через интерфейс
// TodoRepository её легко заменить на базу данных.

// ErrNotFound возвращается когда задача с таким ID не найдена.
// Sentinel error — позволяет точно проверять тип ошибки через errors.Is().
var ErrNotFound = errors.New("not found")

// TodoRepository — интерфейс хранилища.
// Все методы работы с задачами описаны здесь.
// Благодаря интерфейсу можно подменить реализацию (например на PostgreSQL)
// не трогая service и handler.
type TodoRepository interface {
	Add(title, body string) (model.Todo, error)
	Get(id int64) (model.Todo, error)
	List(filter *bool) []model.Todo
	Update(id int64, title, body *string) (model.Todo, error)
	Delete(id int64) error
	Toggle(id int64) (model.Todo, error)
	Clear()
}

// InMemoryTodoRepository — хранилище задач в памяти (map).
// Данные живут пока работает сервер, после перезапуска — сбрасываются.
type InMemoryTodoRepository struct {
	todos  map[int64]model.Todo
	nextID int64
	mu     sync.RWMutex // RWMutex: много читателей одновременно, но только один писатель
}

func NewInMemoryTodoRepository() *InMemoryTodoRepository {
	return &InMemoryTodoRepository{
		todos:  make(map[int64]model.Todo),
		nextID: 1,
	}
}

// Add создаёт новую задачу и сохраняет в map.
func (r *InMemoryTodoRepository) Add(title, body string) (model.Todo, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return model.Todo{}, errors.New("title required")
	}

	// Lock/Unlock — запись, только один горутин за раз.
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	t := model.Todo{
		ID:        r.nextID,
		Title:     title,
		Body:      strings.TrimSpace(body),
		Done:      false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	r.todos[r.nextID] = t
	r.nextID++
	return t, nil
}

// Get ищет задачу по ID. Возвращает ErrNotFound если не существует.
func (r *InMemoryTodoRepository) Get(id int64) (model.Todo, error) {
	// RLock/RUnlock — только чтение, несколько горутинов могут читать одновременно.
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.todos[id]
	if !ok {
		return model.Todo{}, ErrNotFound
	}
	return t, nil
}

// List возвращает все задачи или фильтрует по done.
// filter == nil → вернуть все.
// filter == &true → только выполненные.
// filter == &false → только невыполненные.
func (r *InMemoryTodoRepository) List(filter *bool) []model.Todo {
	// RLock — только чтение.
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := []model.Todo{}
	for _, t := range r.todos {
		if filter != nil && t.Done != *filter {
			continue // пропустить если не подходит по фильтру
		}
		result = append(result, t)
	}
	return result
}

// Update обновляет title и/или body задачи.
// Принимает указатели — nil значит "не менять это поле".
func (r *InMemoryTodoRepository) Update(id int64, title, body *string) (model.Todo, error) {
	// Lock — запись.
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.todos[id]
	if !ok {
		return model.Todo{}, ErrNotFound
	}

	if title != nil {
		newTitle := strings.TrimSpace(*title)
		if newTitle == "" {
			return model.Todo{}, errors.New("title required")
		}
		t.Title = newTitle
	}
	if body != nil {
		t.Body = strings.TrimSpace(*body)
	}
	t.UpdatedAt = time.Now().UTC()

	r.todos[id] = t
	return t, nil
}

// Delete удаляет задачу по ID.
func (r *InMemoryTodoRepository) Delete(id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.todos[id]; !ok {
		return ErrNotFound
	}
	delete(r.todos, id)
	return nil
}

// Toggle переключает Done: true→false, false→true.
func (r *InMemoryTodoRepository) Toggle(id int64) (model.Todo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.todos[id]
	if !ok {
		return model.Todo{}, ErrNotFound
	}
	t.Done = !t.Done
	t.UpdatedAt = time.Now().UTC()
	r.todos[id] = t
	return t, nil
}

// Clear удаляет все задачи.
// nextID не сбрасываем — чтобы ID никогда не повторялись.
// Это учебное решение: в продакшене nextID тоже обычно не сбрасывают.
func (r *InMemoryTodoRepository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.todos = make(map[int64]model.Todo)
}
