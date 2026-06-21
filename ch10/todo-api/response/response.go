// Package response описывает, что именно API возвращает клиенту.
// Каждый JSON-ответ имеет здесь свой тип — так сразу видно,
// какую форму данных ожидать от каждого эндпоинта.
package response

import "todo-api/models"

// Error — ответ при любой ошибке.
//
//	{ "error": "title required" }
type Error struct {
	Error string `json:"error"`
}

// Message — простое текстовое подтверждение (например, после Clear).
//
//	{ "message": "cleared" }
type Message struct {
	Message string `json:"message"`
}

// TodoList — ответ для списка задач (GET /todos).
//
//	{
//	  "count": 2,
//	  "items": [ {...}, {...} ]
//	}
type TodoList struct {
	Count int           `json:"count"`
	Items []models.Todo `json:"items"`
}
