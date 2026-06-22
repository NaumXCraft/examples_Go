package main

import (
	"log"
	"src/handler"
	"src/service"
	"src/store"

	"github.com/gin-gonic/gin"
)

func main() {
	// Инициализация слоёв: store → service → handler.
	// Каждый слой знает только о слое ниже себя.
	repo := store.NewInMemoryTodoRepository()
	todoService := service.NewTodoService(repo)
	h := handler.NewTodoHandler(todoService)

	r := gin.Default()

	// Маршруты сгруппированы по ресурсу /todos.
	// ВАЖНО: /todos/clear регистрируем до /todos/:id/toggle —
	// Gin даёт приоритет статическому сегменту "clear" над параметром ":id",
	// но только если оба маршрута одного HTTP-метода (здесь оба POST).
	r.POST("/todos", h.Create)
	r.GET("/todos", h.List)
	r.GET("/todos/:id", h.Get)
	r.PUT("/todos/:id", h.Update)
	r.DELETE("/todos/:id", h.Delete)
	r.POST("/todos/clear", h.Clear)
	r.POST("/todos/:id/toggle", h.Toggle)

	log.Fatal(r.Run(":8080"))
}
