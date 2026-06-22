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

// Todo — одна задача в нашем списке.
// Теги `json:"..."` говорят Go, как называть поля в JSON-ответе.
// omitempty = не включать поле, если оно пустое.
type Todo struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body,omitempty"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ==================== ХРАНИЛИЩЕ ====================

// Всё состояние живёт в этих трёх переменных.
// В реальном проекте здесь была бы БД, но для обучения in-memory проще.
var (
	todos  = make(map[int64]Todo) // map[id]Todo — быстрый поиск по ID
	nextID = int64(1)             // автоинкремент
	mu     sync.Mutex             // мьютекс: защищает от гонок при параллельных запросах
)

// addTodo создаёт задачу и кладёт в map.
func addTodo(title, body string) (Todo, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Todo{}, errors.New("title required")
	}

	// Lock/Unlock гарантируют, что только один горутин меняет данные за раз.
	mu.Lock()
	defer mu.Unlock() // defer = выполнится при выходе из функции, даже при панике

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
// Второй возврат map-а (ok bool) показывает, нашли ли ключ.
func getTodo(id int64) (Todo, error) {
	mu.Lock()
	defer mu.Unlock()

	t, ok := todos[id]
	if !ok {
		return Todo{}, errors.New("not found")
	}
	return t, nil
}

// listTodos отдаёт все задачи или только с нужным статусом done.
// filter == nil означает «без фильтра» — указатель удобнее, чем отдельный флаг.
func listTodos(filter *bool) []Todo {
	mu.Lock()
	defer mu.Unlock()

	result := []Todo{}
	for _, t := range todos {
		if filter != nil && t.Done != *filter {
			continue // пропустить, если не подходит по фильтру
		}
		result = append(result, t)
	}
	return result
}

// updateTodo меняет title и/или body.
// Принимаем *string, а не string: nil = «не трогать это поле».
func updateTodo(id int64, title, body *string) (Todo, error) {
	mu.Lock()
	defer mu.Unlock()

	t, ok := todos[id]
	if !ok {
		return Todo{}, errors.New("not found")
	}

	if title != nil {
		newTitle := strings.TrimSpace(*title) // *title = разыменование указателя
		if newTitle == "" {
			return Todo{}, errors.New("title required")
		}
		t.Title = newTitle
	}
	if body != nil {
		t.Body = strings.TrimSpace(*body)
	}
	t.UpdatedAt = time.Now().UTC()

	todos[id] = t // map хранит копии, поэтому нужно положить обратно
	return t, nil
}

// deleteTodo удаляет задачу. Возвращает ошибку, если не нашли.
func deleteTodo(id int64) error {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := todos[id]; !ok {
		return errors.New("not found")
	}
	delete(todos, id) // встроенная функция Go для удаления из map
	return nil
}

// toggleTodo переключает Done: true → false, false → true.
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

// clearTodos сбрасывает всё хранилище — удобно для тестов.
func clearTodos() {
	mu.Lock()
	defer mu.Unlock()
	todos = make(map[int64]Todo)
	nextID = 1
}

// ==================== ХЕНДЛЕРЫ ====================

// handleCreate — POST /todos
// ShouldBindJSON читает тело запроса и кладёт в структуру.
// binding:"required" вернёт 400, если поле отсутствует.
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
	c.JSON(201, todo) // 201 Created — стандарт для успешного POST
}

// handleList — GET /todos?done=1|0
// Query-параметр done опциональный: без него вернём все задачи.
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
		// нет параметра — filter остаётся nil, покажем всё
	default:
		c.JSON(400, gin.H{"error": "done must be 1|0|true|false"})
		return
	}

	items := listTodos(filter)
	c.JSON(200, gin.H{"count": len(items), "items": items})
}

// handleGet — GET /todos/:id
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

// handleUpdate — PUT /todos/:id
// Оба поля опциональны: *string в структуре = nil, если клиент не прислал поле.
func handleUpdate(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}

	var input struct {
		Title *string `json:"title"` // *string: nil если не пришло
		Body  *string `json:"body"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	todo, err := updateTodo(id, input.Title, input.Body)
	if err != nil {
		// Разные ошибки → разные HTTP-статусы
		if err.Error() == "not found" {
			c.JSON(404, gin.H{"error": err.Error()})
		} else {
			c.JSON(400, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, todo)
}

// handleDelete — DELETE /todos/:id
// 204 No Content: успех, но тело ответа пустое (так принято для DELETE).
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

// handleToggle — POST /todos/:id/toggle
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

// handleClear — POST /todos/clear
func handleClear(c *gin.Context) {
	clearTodos()
	c.JSON(200, gin.H{"message": "cleared"})
}

// parseID достаёт :id из URL и конвертирует в int64.
// ShouldBindUri читает URL-параметры в структуру — аналог ShouldBindJSON для пути.
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
	r := gin.Default() // Default = логгер + recovery (не падает при панике)

	// Маршруты сгруппированы по ресурсу.
	// Важно: /todos/clear должен быть ВЫШЕ /todos/:id,
	// иначе Gin воспримет "clear" как id.
	r.POST("/todos", handleCreate)
	r.GET("/todos", handleList)
	r.GET("/todos/:id", handleGet)
	r.PUT("/todos/:id", handleUpdate)
	r.DELETE("/todos/:id", handleDelete)
	r.POST("/todos/:id/toggle", handleToggle)
	r.POST("/todos/clear", handleClear)

	log.Fatal(r.Run(":8080")) // log.Fatal = напечатает ошибку и завершит процесс
}
