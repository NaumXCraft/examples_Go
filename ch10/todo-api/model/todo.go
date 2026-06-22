package model

import "time"

// Todo — одна задача в списке.
// Теги `json:"..."` говорят Go как называть поля в JSON-ответе.
// omitempty = не включать поле если оно пустое (Body).
type Todo struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body,omitempty"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
