package handler

import (
	"errors"
	"net/http"
	"todo-api/response"
	"todo-api/service"

	"github.com/gin-gonic/gin"
)

// TodoHandler держит зависимость на сервис.
type TodoHandler struct {
	svc *service.TodoService
}

func NewTodoHandler(svc *service.TodoService) *TodoHandler {
	return &TodoHandler{svc: svc}
}

// RegisterRoutes регистрирует все маршруты на router-группе.
//
//	POST   /todos
//	GET    /todos
//	GET    /todos/:id
//	PUT    /todos/:id
//	DELETE /todos/:id
//	POST   /todos/:id/toggle
//	POST   /todos/clear
func (h *TodoHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("", h.Create)
	rg.GET("", h.List)
	rg.GET("/:id", h.Get)
	rg.PUT("/:id", h.Update)
	rg.DELETE("/:id", h.Delete)
	rg.POST("/:id/toggle", h.Toggle)
	rg.POST("/clear", h.Clear)
}

// ── Request-структуры (входящий JSON) ─────────────────────────

// createRequest — тело POST /todos.
//
//	{ "title": "Buy milk", "body": "2 liters" }
type createRequest struct {
	Title string `json:"title" binding:"required"`
	Body  string `json:"body"`
}

// updateRequest — тело PUT /todos/:id.
// Оба поля опциональны: nil = не менять.
//
//	{ "title": "New title" }
//	{ "body": "new body" }
//	{ "title": "X", "body": "Y" }
type updateRequest struct {
	Title *string `json:"title"`
	Body  *string `json:"body"`
}

// ── Handlers ──────────────────────────────────────────────────

// Create godoc
// POST /todos
// Request:  createRequest
// Response 201: response.TodoResponse
// Response 400: response.Error
func (h *TodoHandler) Create(c *gin.Context) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error{Error: err.Error()})
		return
	}

	todo, err := h.svc.Add(service.CreateInput{
		Title: req.Title,
		Body:  req.Body,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response.NewTodoResponse(todo))
}

// List godoc
// GET /todos?done=1|0   (done — опционально)
// Response 200: response.TodoListResponse
// Response 400: response.Error
func (h *TodoHandler) List(c *gin.Context) {
	var filter *bool

	if q := c.Query("done"); q != "" {
		switch q {
		case "1", "true":
			v := true
			filter = &v
		case "0", "false":
			v := false
			filter = &v
		default:
			c.JSON(http.StatusBadRequest, response.Error{Error: "done must be 1|0|true|false"})
			return
		}
	}

	c.JSON(http.StatusOK, response.NewTodoListResponse(h.svc.List(filter)))
}

// Get godoc
// GET /todos/:id
// Response 200: response.TodoResponse
// Response 404: response.Error
func (h *TodoHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	todo, err := h.svc.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, response.Error{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response.NewTodoResponse(todo))
}

// Update godoc
// PUT /todos/:id
// Request:  updateRequest
// Response 200: response.TodoResponse
// Response 400: response.Error
// Response 404: response.Error
func (h *TodoHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error{Error: err.Error()})
		return
	}

	todo, err := h.svc.Update(id, service.UpdateInput{
		Title: req.Title,
		Body:  req.Body,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, response.Error{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response.NewTodoResponse(todo))
}

// Delete godoc
// DELETE /todos/:id
// Response 204: (нет тела)
// Response 404: response.Error
func (h *TodoHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	if err := h.svc.Delete(id); err != nil {
		c.JSON(http.StatusNotFound, response.Error{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// Toggle godoc
// POST /todos/:id/toggle
// Response 200: response.TodoResponse
// Response 404: response.Error
func (h *TodoHandler) Toggle(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	todo, err := h.svc.Toggle(id)
	if err != nil {
		c.JSON(http.StatusNotFound, response.Error{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response.NewTodoResponse(todo))
}

// Clear godoc
// POST /todos/clear
// Response 200: response.OK
func (h *TodoHandler) Clear(c *gin.Context) {
	h.svc.Clear()
	c.JSON(http.StatusOK, response.OK{Message: "cleared"})
}

// ── вспомогательные ───────────────────────────────────────────

// parseID читает :id из пути и пишет 400, если невалидно.
func parseID(c *gin.Context) (int64, bool) {
	var uri struct {
		ID int64 `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, response.Error{Error: "invalid id"})
		return 0, false
	}
	return uri.ID, true
}
