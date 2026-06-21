// Package models содержит структуры данных приложения.
package models

import "time"

// Todo — одна задача.
//
// JSON-вид:
//
//	{
//	  "id": 1,
//	  "title": "Buy milk",
//	  "body": "2 liters",
//	  "done": false,
//	  "createdAt": "2024-06-18T10:00:00Z",
//	  "updatedAt": "2024-06-18T10:00:00Z"
//	}
type Todo struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body,omitempty"` // не показываем поле в JSON, если оно пустое
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
