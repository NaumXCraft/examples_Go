package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"cart-api/models"
	"cart-api/storage"
)

// ─────────────────────────────────────────────
//  CartHandler — все хендлеры корзины
//
//  Хендлер получает запрос, достаёт корзину из Storage,
//  вызывает методы модели, возвращает JSON.
//  Бизнес-логика — в models, хендлер только связывает.
// ─────────────────────────────────────────────

type CartHandler struct {
	storage *storage.Storage
}

func NewCartHandler(s *storage.Storage) *CartHandler {
	return &CartHandler{storage: s}
}

// ── Helpers ──────────────────────────────────

func success(c *gin.Context, message string, cart *models.Cart) {
	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"data":    cart.ToResponse(),
	})
}

func fail(c *gin.Context, status int, err string) {
	c.JSON(status, gin.H{
		"message": "error",
		"error":   err,
	})
}

// ── GET /cart/:userID ─────────────────────────
//
// Возвращает текущее состояние корзины.
// Если корзины нет — создаёт пустую.

func (h *CartHandler) GetCart(c *gin.Context) {
	userID := c.Param("userID")
	cart := h.storage.GetCart(userID)
	success(c, "cart_fetched", cart)
}

// ── POST /cart/:userID/items ──────────────────
//
// Добавляет товар в корзину.
//
// Body:
// {
//   "id": 1,
//   "article": "SHMP001",
//   "name": "Shampoo",
//   "category": "Hiustenhoito",
//   "country": "Suomi",
//   "image": "http://...",
//   "is_professional": false,
//   "is_active": true,
//   "price": 12.99,
//   "discount_price": null,
//   "quantity": 2
// }

type AddItemRequest struct {
	ID             int      `json:"id"              binding:"required"`
	Article        string   `json:"article"         binding:"required"`
	Name           string   `json:"name"            binding:"required"`
	Category       string   `json:"category"`
	Country        string   `json:"country"`
	Image          string   `json:"image"`
	IsProfessional bool     `json:"is_professional"`
	IsActive       bool     `json:"is_active"`
	Price          float64  `json:"price"           binding:"required,gt=0"`
	DiscountPrice  *float64 `json:"discount_price"`
	Quantity       int      `json:"quantity"        binding:"required,gt=0"`
}

func (h *CartHandler) AddItem(c *gin.Context) {
	userID := c.Param("userID")

	var req AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	item := models.NewItem(
		req.ID,
		req.Article,
		req.Name,
		req.Category,
		req.Country,
		req.Image,
		req.IsProfessional,
		req.IsActive,
		req.Price,
		req.DiscountPrice,
		req.Quantity,
	)

	cart := h.storage.GetCart(userID)

	if err := cart.AddItem(item); err != nil {
		fail(c, http.StatusUnprocessableEntity, err.Error())
		return
	}

	h.storage.SaveCart(userID, cart)
	success(c, "item_added", cart)
}

// ── DELETE /cart/:userID/items/:itemID ────────
//
// Удаляет товар из корзины по ID.

func (h *CartHandler) RemoveItem(c *gin.Context) {
	userID := c.Param("userID")
	itemIDStr := c.Param("itemID")

	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid item id")
		return
	}

	cart := h.storage.GetCart(userID)

	if err := cart.RemoveItem(itemID); err != nil {
		fail(c, http.StatusNotFound, err.Error())
		return
	}

	h.storage.SaveCart(userID, cart)
	success(c, "item_removed", cart)
}

// ── POST /cart/:userID/coupon ─────────────────
//
// Применяет купон к корзине.
//
// Body:
// {
//   "code": "9003006",
//   "percent": 20
// }

type ApplyCouponRequest struct {
	Code    string `json:"code"    binding:"required"`
	Percent int    `json:"percent" binding:"required,gt=0,lte=100"`
}

func (h *CartHandler) ApplyCoupon(c *gin.Context) {
	userID := c.Param("userID")

	var req ApplyCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	cart := h.storage.GetCart(userID)

	if err := cart.ApplyCoupon(req.Code, req.Percent); err != nil {
		fail(c, http.StatusUnprocessableEntity, err.Error())
		return
	}

	h.storage.SaveCart(userID, cart)
	success(c, "coupon_applied", cart)
}

// ── DELETE /cart/:userID/coupon ───────────────
//
// Убирает купон из корзины.

func (h *CartHandler) RemoveCoupon(c *gin.Context) {
	userID := c.Param("userID")

	cart := h.storage.GetCart(userID)
	cart.RemoveCoupon()

	h.storage.SaveCart(userID, cart)
	success(c, "coupon_removed", cart)
}

// ── DELETE /cart/:userID ──────────────────────
//
// Полностью очищает корзину пользователя.

func (h *CartHandler) ClearCart(c *gin.Context) {
	userID := c.Param("userID")

	if err := h.storage.DeleteCart(userID); err != nil {
		fail(c, http.StatusNotFound, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "cart_cleared",
	})
}
