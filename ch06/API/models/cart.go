package models

import (
	"errors"
	"fmt"
	"strconv"
)

// ─────────────────────────────────────────────
//  ГИБРИДНАЯ ИДЕОЛОГИЯ
//
//  Cart     → МУТАБЕЛЬНАЯ  (*Cart, pointer receiver)
//  Item     → ИММУТАБЕЛЬНАЯ (Item, value receiver)
//  Coupon   → ИММУТАБЕЛЬНАЯ (Coupon, value receiver)
//  Shipping → ИММУТАБЕЛЬНАЯ (Shipping, value receiver)
// ─────────────────────────────────────────────

const MaxCartItems = 15

// ══════════════════════════════════════════════
//  Item — ИММУТАБЕЛЬНАЯ
// ══════════════════════════════════════════════

type Item struct {
	id             int
	article        string
	name           string
	category       string
	country        string
	image          string
	isProfessional bool
	isActive       bool
	price          float64
	discountPrice  *float64
	quantity       int
}

func NewItem(id int, article, name, category, country, image string, isPro, isActive bool, price float64, discountPrice *float64, qty int) Item {
	return Item{
		id:             id,
		article:        article,
		name:           name,
		category:       category,
		country:        country,
		image:          image,
		isProfessional: isPro,
		isActive:       isActive,
		price:          price,
		discountPrice:  discountPrice,
		quantity:       qty,
	}
}

// Геттеры — value receivers, Item не меняется
func (i Item) ID() int                 { return i.id }
func (i Item) Article() string         { return i.article }
func (i Item) Name() string            { return i.name }
func (i Item) Category() string        { return i.category }
func (i Item) Country() string         { return i.country }
func (i Item) Image() string           { return i.image }
func (i Item) IsPro() bool             { return i.isProfessional }
func (i Item) IsActive() bool          { return i.isActive }
func (i Item) Price() float64          { return i.price }
func (i Item) DiscountPrice() *float64 { return i.discountPrice }
func (i Item) Quantity() int           { return i.quantity }

func (i Item) Total() float64 {
	price := i.price
	if i.discountPrice != nil {
		price = *i.discountPrice
	}
	return Round2(price * float64(i.quantity))
}

// WithQuantity — возвращает НОВЫЙ Item, оригинал не тронут
func (i Item) WithQuantity(qty int) Item {
	i.quantity = qty
	return i
}

// ItemResponse — структура для JSON ответа
type ItemResponse struct {
	ID             int     `json:"id"`
	Article        string  `json:"article"`
	Name           string  `json:"name"`
	Category       string  `json:"category"`
	Country        string  `json:"country"`
	Image          string  `json:"image"`
	IsProfessional bool    `json:"is_professional"`
	IsActive       bool    `json:"is_active"`
	Price          string  `json:"price"`
	DiscountPrice  *string `json:"discount_price"`
	Quantity       int     `json:"quantity"`
	Total          string  `json:"total"`
}

func (i Item) ToResponse() ItemResponse {
	var dp *string
	if i.discountPrice != nil {
		s := fmt.Sprintf("%.2f", *i.discountPrice)
		dp = &s
	}
	return ItemResponse{
		ID:             i.id,
		Article:        i.article,
		Name:           i.name,
		Category:       i.category,
		Country:        i.country,
		Image:          i.image,
		IsProfessional: i.isProfessional,
		IsActive:       i.isActive,
		Price:          fmt.Sprintf("%.2f", i.price),
		DiscountPrice:  dp,
		Quantity:       i.quantity,
		Total:          fmt.Sprintf("%.2f", i.Total()),
	}
}

// ══════════════════════════════════════════════
//  Coupon — ИММУТАБЕЛЬНАЯ
// ══════════════════════════════════════════════

type Coupon struct {
	code     string
	percent  int
	discount float64
}

func NewCoupon(code string, percent int, base float64) Coupon {
	return Coupon{
		code:     code,
		percent:  percent,
		discount: Round2(base * float64(percent) / 100),
	}
}

func (c Coupon) Code() string      { return c.code }
func (c Coupon) Percent() int      { return c.percent }
func (c Coupon) Discount() float64 { return c.discount }

type CouponResponse struct {
	Code     string `json:"code"`
	Percent  int    `json:"percent"`
	Discount string `json:"discount"`
}

func (c Coupon) ToResponse() CouponResponse {
	return CouponResponse{
		Code:     c.code,
		Percent:  c.percent,
		Discount: fmt.Sprintf("%.2f", c.discount),
	}
}

// ══════════════════════════════════════════════
//  Shipping — ИММУТАБЕЛЬНАЯ
// ══════════════════════════════════════════════

type Shipping struct {
	method string
	price  float64
}

func NewShipping(method string, price float64) Shipping {
	return Shipping{method: method, price: price}
}

func (s Shipping) Method() string { return s.method }
func (s Shipping) Price() float64 { return s.price }

type ShippingResponse struct {
	Method string `json:"method"`
	Price  string `json:"price"`
}

func (s Shipping) ToResponse() ShippingResponse {
	return ShippingResponse{
		Method: s.method,
		Price:  fmt.Sprintf("%.2f", s.price),
	}
}

// ══════════════════════════════════════════════
//  Cart — МУТАБЕЛЬНАЯ
//
//  *Cart — указатель, явный сигнал: "этот объект меняется".
//  Все методы с pointer receiver (*Cart) мутируют оригинал.
// ══════════════════════════════════════════════

type Cart struct {
	pricingTier bool
	items       []Item
	coupon      *Coupon
	shipping    Shipping
	currency    string
	warnings    []string
}

// NewCart возвращает *Cart — МУТАБЕЛЬНАЯ корзина
func NewCart(pricingTier bool, currency string, shipping Shipping) *Cart {
	return &Cart{
		pricingTier: pricingTier,
		currency:    currency,
		shipping:    shipping,
		items:       make([]Item, 0, MaxCartItems),
		warnings:    make([]string, 0),
	}
}

// ── Pointer receivers (*Cart) — мутируют корзину ──

// AddItem добавляет товар или увеличивает количество
func (c *Cart) AddItem(item Item) error {
	if !item.IsActive() {
		return errors.New("item is not active")
	}
	if item.IsPro() && !c.pricingTier {
		return errors.New("professional item requires PRO pricing tier")
	}
	if c.CartCount() >= MaxCartItems {
		return fmt.Errorf("cart is full: maximum %d items reached", MaxCartItems)
	}

	for i, existing := range c.items {
		if existing.ID() == item.ID() {
			c.items[i] = existing.WithQuantity(existing.Quantity() + item.Quantity()) // МУТАЦИЯ
			c.recalcCoupon()
			return nil
		}
	}

	c.items = append(c.items, item) // МУТАЦИЯ
	c.recalcCoupon()
	return nil
}

// RemoveItem удаляет товар по ID
func (c *Cart) RemoveItem(id int) error {
	for i, item := range c.items {
		if item.ID() == id {
			c.items = append(c.items[:i], c.items[i+1:]...) // МУТАЦИЯ
			c.recalcCoupon()
			return nil
		}
	}
	return fmt.Errorf("item with id %d not found", id)
}

// ApplyCoupon применяет купон
func (c *Cart) ApplyCoupon(code string, percent int) error {
	if percent <= 0 || percent > 100 {
		return errors.New("coupon percent must be between 1 and 100")
	}
	if c.Base() == 0 {
		return errors.New("cannot apply coupon to empty cart")
	}
	coupon := NewCoupon(code, percent, c.Base()) // Coupon — иммутабельный
	c.coupon = &coupon                           // МУТАЦИЯ: записываем в корзину
	return nil
}

// RemoveCoupon убирает купон
func (c *Cart) RemoveCoupon() {
	c.coupon = nil // МУТАЦИЯ
}

// recalcCoupon пересчитывает скидку при изменении состава корзины
func (c *Cart) recalcCoupon() {
	if c.coupon != nil {
		newCoupon := NewCoupon(c.coupon.Code(), c.coupon.Percent(), c.Base())
		c.coupon = &newCoupon // МУТАЦИЯ
	}
}

// ── Чтение — не мутируют ──

func (c *Cart) Base() float64 {
	total := 0.0
	for _, item := range c.items {
		total += item.Total()
	}
	return Round2(total)
}

func (c *Cart) FinalTotal() float64 {
	total := c.Base()
	if c.coupon != nil {
		total -= c.coupon.Discount()
	}
	total += c.shipping.Price()
	return Round2(total)
}

func (c *Cart) CartCount() int {
	count := 0
	for _, item := range c.items {
		count += item.Quantity()
	}
	return count
}

func (c *Cart) IsFull() bool {
	return c.CartCount() >= MaxCartItems
}

// ── JSON Response ──

type CartResponse struct {
	PricingTier bool             `json:"pricing_tier"`
	Items       []ItemResponse   `json:"items"`
	Base        string           `json:"base"`
	CartCount   int              `json:"cart_count"`
	Shipping    ShippingResponse `json:"shipping"`
	Coupon      *CouponResponse  `json:"coupon"`
	FinalTotal  string           `json:"final_total"`
	Currency    string           `json:"currency"`
	Warnings    []string         `json:"warnings"`
}

func (c *Cart) ToResponse() CartResponse {
	items := make([]ItemResponse, len(c.items))
	for i, item := range c.items {
		items[i] = item.ToResponse()
	}

	var coupon *CouponResponse
	if c.coupon != nil {
		cr := c.coupon.ToResponse()
		coupon = &cr
	}

	warnings := c.warnings
	if warnings == nil {
		warnings = []string{}
	}

	return CartResponse{
		PricingTier: c.pricingTier,
		Items:       items,
		Base:        fmt.Sprintf("%.2f", c.Base()),
		CartCount:   c.CartCount(),
		Shipping:    c.shipping.ToResponse(),
		Coupon:      coupon,
		FinalTotal:  fmt.Sprintf("%.2f", c.FinalTotal()),
		Currency:    c.currency,
		Warnings:    warnings,
	}
}

// ─────────────────────────────────────────────

func Round2(v float64) float64 {
	f, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", v), 64)
	return f
}
