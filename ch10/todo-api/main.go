package main

import (
	"log"
	"todo-api/handler"
	"todo-api/service"

	"github.com/gin-gonic/gin"
)

func main() {
	svc := service.New(0) // 0 = без ограничений
	h := handler.NewTodoHandler(svc)

	r := gin.Default()

	todos := r.Group("/todos")
	h.RegisterRoutes(todos)

	log.Fatal(r.Run(":8080"))
}
