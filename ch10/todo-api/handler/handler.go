package handler

import (
	"errors"
	"src/service"
	"src/store"

	"github.com/gin-gonic/gin"
)

// Package handler отвечает за обработку HTTP-запросов.
// Здесь только: парсинг запроса, вызов service, формирование ответа.
// Никакой бизнес-логики здесь нет — всё в service.

// TodoHandler держит ссылку на service и регистрируется на роутах в main.go.
type TodoHandler struct {
	service *service.TodoService
}

func NewTodoHandler(service *service.TodoService) *TodoHandler {
	return &TodoHandler{service: service}
}

// parseID достаёт :id из URL и конвертирует в int64.
// Вынесено отдельно чтобы не дублировать в каждом хендлере.
func parseID(c *gin.Context) (int64, error) {
	var uri struct {
		ID int64 `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		return 0, err
	}
	return uri.ID, nil
}

// respondError — хелпер: определяет статус по типу ошибки и отвечает JSON.
// ErrNotFound → 404, всё остальное → 400.
func respondError(c *gin.Context, err error) {
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(404, gin.H{"error": "not found"})
	} else {
		c.JSON(400, gin.H{"error": err.Error()})
	}
}

func (h *TodoHandler) Create(c *gin.Context) {
	var input struct {
		Title string `json:"title" binding:"required"`
		Body  string `json:"body"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	todo, err := h.service.Create(input.Title, input.Body)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(201, todo)
}

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
		// параметр не передан — показываем все задачи
	default:
		c.JSON(400, gin.H{"error": "done must be 1|0|true|false"})
		return
	}

	items := h.service.List(filter)
	c.JSON(200, gin.H{"count": len(items), "items": items})
}

func (h *TodoHandler) Get(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	todo, err := h.service.GetByID(id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, todo)
}

func (h *TodoHandler) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	// Указатели (*string) нужны чтобы различить:
	// - поле не передано (nil) → не менять
	// - поле передано пустым ("") → ошибка валидации в store
	var input struct {
		Title *string `json:"title"`
		Body  *string `json:"body"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	todo, err := h.service.Update(id, input.Title, input.Body)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, todo)
}

func (h *TodoHandler) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	if err := h.service.Delete(id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(204)
}

// Toggle переключает done ↔ not done.
// Используем POST а не PATCH — это действие (action), не обновление поля.
func (h *TodoHandler) Toggle(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	todo, err := h.service.Toggle(id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, todo)
}

func (h *TodoHandler) Clear(c *gin.Context) {
	h.service.Clear()
	c.JSON(200, gin.H{"message": "cleared"})
}
