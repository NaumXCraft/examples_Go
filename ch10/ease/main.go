package main

import (
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ==================== МОДЕЛЬ ====================

// Todo — одна задача.
type Todo struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body,omitempty"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ==================== ХРАНИЛИЩЕ (in-memory) ====================

// Всё состояние приложения — здесь. Никаких отдельных пакетов:
// это маленький проект, и так гораздо проще понять, что происходит.
var (
	todos  = make(map[int64]Todo)
	nextID = int64(1)
	mu     sync.Mutex // защищает todos и nextID от гонок при параллельных запросах
)

// ---- функции работы с хранилищем ----

// addTodo создаёт новую задачу.
func addTodo(title, body string) (Todo, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Todo{}, errors.New("title required")
	}

	mu.Lock()
	defer mu.Unlock()

	now := time.Now().UTC()
	t := Todo{
		ID:        nextID,
		Title:     title,
		Body:      strings.TrimSpace(body),
		Done:      false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	todos[nextID] = t
	nextID++
	return t, nil
}

// getTodo ищет задачу по ID.
func getTodo(id int64) (Todo, error) {
	mu.Lock()
	defer mu.Unlock()

	t, ok := todos[id]
	if !ok {
		return Todo{}, errors.New("not found")
	}
	return t, nil
}

// listTodos возвращает все задачи, либо отфильтрованные по done.
// filter == nil → вернуть все.
func listTodos(filter *bool) []Todo {
	mu.Lock()
	defer mu.Unlock()

	result := []Todo{}
	for _, t := range todos {
		if filter != nil && t.Done != *filter {
			continue
		}
		result = append(result, t)
	}
	return result
}

// updateTodo меняет title и/или body. nil-поле = не трогать.
func updateTodo(id int64, title, body *string) (Todo, error) {
	mu.Lock()
	defer mu.Unlock()

	t, ok := todos[id]
	if !ok {
		return Todo{}, errors.New("not found")
	}

	if title != nil {
		newTitle := strings.TrimSpace(*title)
		if newTitle == "" {
			return Todo{}, errors.New("title required")
		}
		t.Title = newTitle
	}
	if body != nil {
		t.Body = strings.TrimSpace(*body)
	}
	t.UpdatedAt = time.Now().UTC()

	todos[id] = t
	return t, nil
}

// deleteTodo удаляет задачу по ID.
func deleteTodo(id int64) error {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := todos[id]; !ok {
		return errors.New("not found")
	}
	delete(todos, id)
	return nil
}

// toggleTodo переключает Done (true ↔ false).
func toggleTodo(id int64) (Todo, error) {
	mu.Lock()
	defer mu.Unlock()

	t, ok := todos[id]
	if !ok {
		return Todo{}, errors.New("not found")
	}
	t.Done = !t.Done
	t.UpdatedAt = time.Now().UTC()
	todos[id] = t
	return t, nil
}

// clearTodos удаляет все задачи и сбрасывает счётчик ID.
func clearTodos() {
	mu.Lock()
	defer mu.Unlock()
	todos = make(map[int64]Todo)
	nextID = 1
}

// ==================== HTTP-ХЕНДЛЕРЫ ====================

// POST /todos
// body: { "title": "...", "body": "..." }
func handleCreate(c *gin.Context) {
	var input struct {
		Title string `json:"title" binding:"required"`
		Body  string `json:"body"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	todo, err := addTodo(input.Title, input.Body)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, todo)
}

// GET /todos?done=1|0
func handleList(c *gin.Context) {
	var filter *bool

	switch c.Query("done") {
	case "1", "true":
		v := true
		filter = &v
	case "0", "false":
		v := false
		filter = &v
	case "":
		// без фильтра — показываем все
	default:
		c.JSON(400, gin.H{"error": "done must be 1|0|true|false"})
		return
	}

	items := listTodos(filter)
	c.JSON(200, gin.H{"count": len(items), "items": items})
}

// GET /todos/:id
func handleGet(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	todo, err := getTodo(id)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, todo)
}

// PUT /todos/:id
// body: { "title": "...", "body": "..." }  (оба поля опциональны)
func handleUpdate(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	var input struct {
		Title *string `json:"title"`
		Body  *string `json:"body"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	todo, err := updateTodo(id, input.Title, input.Body)
	if err != nil {
		if err.Error() == "not found" {
			c.JSON(404, gin.H{"error": err.Error()})
		} else {
			c.JSON(400, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, todo)
}

// DELETE /todos/:id
func handleDelete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	if err := deleteTodo(id); err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.Status(204)
}

// POST /todos/:id/toggle
func handleToggle(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	todo, err := toggleTodo(id)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, todo)
}

// POST /todos/clear
func handleClear(c *gin.Context) {
	clearTodos()
	c.JSON(200, gin.H{"message": "cleared"})
}

// parseID достаёт :id из URL и переводит в int64.
func parseID(c *gin.Context) (int64, error) {
	var uri struct {
		ID int64 `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		return 0, err
	}
	return uri.ID, nil
}

// ==================== MAIN ====================

func main() {
	r := gin.Default()

	r.POST("/todos", handleCreate)
	r.GET("/todos", handleList)
	r.GET("/todos/:id", handleGet)
	r.PUT("/todos/:id", handleUpdate)
	r.DELETE("/todos/:id", handleDelete)
	r.POST("/todos/:id/toggle", handleToggle)
	r.POST("/todos/clear", handleClear)

	log.Fatal(r.Run(":8080"))
}
