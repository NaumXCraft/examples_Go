// Точка входа: создаём сервис, подключаем хендлеры, запускаем сервер.
package main

import (
	"log"

	"todo-api/handler"
	"todo-api/service"

	"github.com/gin-gonic/gin"
)

func main() {
	svc := service.New()
	h := handler.New(svc)

	r := gin.Default()

	todos := r.Group("/todos")
	h.RegisterRoutes(todos)

	log.Fatal(r.Run(":8080"))
}
