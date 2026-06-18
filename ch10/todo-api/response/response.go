// Package response содержит все JSON-структуры,
// которые API возвращает клиенту.
// Правило: каждый endpoint → свой response-тип,
// чтобы контракт был явным и читаемым.
package response

import "todo-api/models"

// ── Общие обёртки ──────────────────────────────────────────────

// Error — стандартный ответ при ошибке.
//
//	{ "error": "title required" }
type Error struct {
	Error string `json:"error"`
}

// OK — простое подтверждение действия.
//
//	{ "message": "cleared" }
type OK struct {
	Message string `json:"message"`
}

// ── Todo-ответы ────────────────────────────────────────────────

// TodoResponse — один Todo (Create / Get / Update / Toggle).
//
//	{
//	  "id": 1,
//	  "title": "Buy milk",
//	  "body": "2 liters",
//	  "done": false,
//	  "createdAt": "...",
//	  "updatedAt": "..."
//	}
type TodoResponse struct {
	models.Todo
}

// TodoListResponse — список задач (List).
//
//	{
//	  "count": 2,
//	  "items": [ {...}, {...} ]
//	}
type TodoListResponse struct {
	Count int           `json:"count"`
	Items []models.Todo `json:"items"`
}

// NewTodoResponse оборачивает модель в response-тип.
func NewTodoResponse(t models.Todo) TodoResponse {
	return TodoResponse{t}
}

// NewTodoListResponse оборачивает срез моделей.
func NewTodoListResponse(items []models.Todo) TodoListResponse {
	if items == nil {
		items = []models.Todo{} // никогда не возвращаем null в JSON
	}
	return TodoListResponse{
		Count: len(items),
		Items: items,
	}
}
