// Package handler связывает HTTP-запросы (Gin) с сервисом.
// Здесь и только здесь мы знаем про JSON, HTTP-коды и роуты.
// Вся реальная логика (создание, поиск, удаление задач) — в пакете service.
package handler

import (
	"todo-api/response"
	"todo-api/service"

	"github.com/gin-gonic/gin"
)

// TodoHandler хранит ссылку на сервис, чтобы вызывать его методы из хендлеров.
type TodoHandler struct {
	svc *service.TodoService
}

// New создаёт хендлер поверх готового сервиса.
func New(svc *service.TodoService) *TodoHandler {
	return &TodoHandler{svc: svc}
}

// RegisterRoutes регистрирует все маршруты Todo API на переданной группе.
//
//	POST   /todos            — создать задачу
//	GET    /todos             — список задач
//	GET    /todos/:id         — одна задача
//	PUT    /todos/:id         — обновить задачу
//	DELETE /todos/:id         — удалить задачу
//	POST   /todos/:id/toggle  — переключить done
//	POST   /todos/clear       — удалить все задачи
func (h *TodoHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("", h.Create)
	rg.GET("", h.List)
	rg.GET("/:id", h.Get)
	rg.PUT("/:id", h.Update)
	rg.DELETE("/:id", h.Delete)
	rg.POST("/:id/toggle", h.Toggle)
	rg.POST("/clear", h.Clear)
}

// ---- структуры входящего JSON (то, что присылает клиент) ----

// createRequest — тело запроса POST /todos.
//
//	{ "title": "Buy milk", "body": "2 liters" }
type createRequest struct {
	Title string `json:"title" binding:"required"`
	Body  string `json:"body"`
}

// updateRequest — тело запроса PUT /todos/:id.
// Оба поля — указатели: nil означает "не менять это поле".
//
//	{ "title": "New title" }
type updateRequest struct {
	Title *string `json:"title"`
	Body  *string `json:"body"`
}

// ---- сами хендлеры ----

// Create обрабатывает POST /todos.
func (h *TodoHandler) Create(c *gin.Context) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error{Error: err.Error()})
		return
	}

	todo, err := h.svc.Add(req.Title, req.Body)
	if err != nil {
		c.JSON(400, response.Error{Error: err.Error()})
		return
	}

	c.JSON(201, todo)
}

// List обрабатывает GET /todos?done=1|0.
func (h *TodoHandler) List(c *gin.Context) {
	var filter *bool

	switch c.Query("done") {
	case "1", "true":
		v := true
		filter = &v
	case "0", "false":
		v := false
		filter = &v
	case "":
		// без параметра done — показываем все задачи
	default:
		c.JSON(400, response.Error{Error: "done must be 1|0|true|false"})
		return
	}

	items := h.svc.List(filter)
	c.JSON(200, response.TodoList{
		Count: len(items),
		Items: items,
	})
}

// Get обрабатывает GET /todos/:id.
func (h *TodoHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return // parseID уже отправил ответ с ошибкой
	}

	todo, err := h.svc.Get(id)
	if err != nil {
		c.JSON(404, response.Error{Error: err.Error()})
		return
	}

	c.JSON(200, todo)
}

// Update обрабатывает PUT /todos/:id.
func (h *TodoHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error{Error: err.Error()})
		return
	}

	todo, err := h.svc.Update(id, req.Title, req.Body)
	if err != nil {
		// "not found" → 404, всё остальное (например, "title required") → 400
		if err.Error() == "not found" {
			c.JSON(404, response.Error{Error: err.Error()})
		} else {
			c.JSON(400, response.Error{Error: err.Error()})
		}
		return
	}

	c.JSON(200, todo)
}

// Delete обрабатывает DELETE /todos/:id.
func (h *TodoHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	if err := h.svc.Delete(id); err != nil {
		c.JSON(404, response.Error{Error: err.Error()})
		return
	}

	c.Status(204) // No Content — без тела ответа
}

// Toggle обрабатывает POST /todos/:id/toggle.
func (h *TodoHandler) Toggle(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	todo, err := h.svc.Toggle(id)
	if err != nil {
		c.JSON(404, response.Error{Error: err.Error()})
		return
	}

	c.JSON(200, todo)
}

// Clear обрабатывает POST /todos/clear.
func (h *TodoHandler) Clear(c *gin.Context) {
	h.svc.Clear()
	c.JSON(200, response.Message{Message: "cleared"})
}

// ---- вспомогательная функция ----

// parseID достаёт параметр :id из URL и переводит в int64.
// Если id некорректный — сам отправляет 400 и возвращает ok = false,
// чтобы хендлер мог сразу прерваться через "return".
func parseID(c *gin.Context) (id int64, ok bool) {
	var uri struct {
		ID int64 `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(400, response.Error{Error: "invalid id"})
		return 0, false
	}
	return uri.ID, true
}
