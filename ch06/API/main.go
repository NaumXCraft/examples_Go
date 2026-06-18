package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"cart-api/handlers"
	"cart-api/storage"
)

func main() {
	// Хранилище корзин в памяти
	store := storage.New()

	// Хендлеры
	cartHandler := handlers.NewCartHandler(store)

	// Роутер
	r := gin.Default()

	// ── Маршруты ─────────────────────────────
	//
	//  GET    /cart/:userID                — получить корзину
	//  POST   /cart/:userID/items          — добавить товар
	//  DELETE /cart/:userID/items/:itemID  — удалить товар
	//  POST   /cart/:userID/coupon         — применить купон
	//  DELETE /cart/:userID/coupon         — убрать купон
	//  DELETE /cart/:userID               — очистить корзину

	cart := r.Group("/cart")
	{
		cart.GET("/:userID", cartHandler.GetCart)
		cart.DELETE("/:userID", cartHandler.ClearCart)

		cart.POST("/:userID/items", cartHandler.AddItem)
		cart.DELETE("/:userID/items/:itemID", cartHandler.RemoveItem)

		cart.POST("/:userID/coupon", cartHandler.ApplyCoupon)
		cart.DELETE("/:userID/coupon", cartHandler.RemoveCoupon)
	}

	log.Println("Cart API running on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
