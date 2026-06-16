package main

import (
	"fmt"
	"strconv"
)

// ─────────────────────────────────────────────
//  ИММУТАБЕЛЬНАЯ ИДЕОЛОГИЯ
//  Value receivers (T) — методы работают с копией.
//  Каждая "мутация" возвращает НОВУЮ корзину.
//  Оригинал никогда не меняется.
// ─────────────────────────────────────────────

type Item struct {
	id             int
	article        string
	name           string
	isProfessional bool
	price          float64
	discountPrice  *float64
	quantity       int
}

// Конструктор — единственный способ создать Item
func NewItem(id int, article, name string, isPro bool, price float64, qty int) Item {
	return Item{ // возвращаем значение, не указатель
		id:             id,
		article:        article,
		name:           name,
		isProfessional: isPro,
		price:          price,
		quantity:       qty,
	}
}

// Value receiver — читаем, не меняем
func (i Item) Total() float64 {
	price := i.price
	if i.discountPrice != nil {
		price = *i.discountPrice
	}
	return round2(price * float64(i.quantity))
}

func (i Item) WithQuantity(qty int) Item {
	i.quantity = qty // меняем КОПИЮ
	return i         // возвращаем новую
}

// ─────────────────────────────────────────────

type Coupon struct {
	code     string
	percent  int
	discount float64
}

type Shipping struct {
	method string
	price  float64
}

type Cart struct {
	pricingTier bool
	items       []Item
	coupon      *Coupon
	shipping    Shipping
	currency    string
}

// Конструктор корзины
func NewCart(pricingTier bool, currency string, shipping Shipping) Cart {
	return Cart{
		pricingTier: pricingTier,
		currency:    currency,
		shipping:    shipping,
	}
}

// ── Value receivers — только чтение ────────────

func (c Cart) Base() float64 {
	total := 0.0
	for _, item := range c.items {
		total += item.Total()
	}
	return round2(total)
}

func (c Cart) FinalTotal() float64 {
	total := c.Base()
	if c.coupon != nil {
		total -= c.coupon.discount
	}
	total += c.shipping.price
	return round2(total)
}

func (c Cart) CartCount() int {
	count := 0
	for _, item := range c.items {
		count += item.quantity
	}
	return count
}

// ── "Мутирующие" методы — возвращают НОВУЮ корзину ──

// WithItem возвращает новую корзину с добавленным/обновлённым товаром
func (c Cart) WithItem(newItem Item) Cart {
	newItems := make([]Item, len(c.items))
	copy(newItems, c.items) // копируем слайс

	for i, existing := range newItems {
		if existing.id == newItem.id {
			newItems[i] = existing.WithQuantity(existing.quantity + newItem.quantity)
			c.items = newItems
			return c // возвращаем изменённую КОПИЮ Cart
		}
	}

	c.items = append(newItems, newItem)
	return c
}

// WithoutItem возвращает новую корзину без указанного товара
func (c Cart) WithoutItem(id int) Cart {
	newItems := make([]Item, 0, len(c.items))
	for _, item := range c.items {
		if item.id != id {
			newItems = append(newItems, item)
		}
	}
	c.items = newItems
	return c
}

// WithCoupon возвращает новую корзину с применённым купоном
func (c Cart) WithCoupon(code string, percent int) Cart {
	discount := c.Base() * float64(percent) / 100
	coupon := &Coupon{
		code:     code,
		percent:  percent,
		discount: round2(discount),
	}
	c.coupon = coupon
	return c
}

// Print выводит состояние корзины
func (c Cart) Print(label string) {
	tier := "BASIC"
	if c.pricingTier {
		tier = "PRO"
	}
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Printf("║  %s [%s — %s]\n", label, tier, c.currency)
	fmt.Println("╠══════════════════════════════════════╣")
	for _, item := range c.items {
		pro := ""
		if item.isProfessional {
			pro = " [PRO]"
		}
		fmt.Printf("║  %-18s x%d  %.2f %s%s\n",
			item.name, item.quantity, item.Total(), c.currency, pro)
	}
	fmt.Println("╠══════════════════════════════════════╣")
	fmt.Printf("║  Товаров:      %d шт.\n", c.CartCount())
	fmt.Printf("║  База:         %.2f %s\n", c.Base(), c.currency)
	if c.coupon != nil {
		fmt.Printf("║  Купон [%s]: -%d%% = -%.2f %s\n",
			c.coupon.code, c.coupon.percent, c.coupon.discount, c.currency)
	}
	fmt.Printf("║  Доставка:     +%.2f %s (%s)\n",
		c.shipping.price, c.currency, c.shipping.method)
	fmt.Printf("║  ИТОГО:        %.2f %s\n", c.FinalTotal(), c.currency)
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()
}

// ─────────────────────────────────────────────

func round2(v float64) float64 {
	f, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", v), 64)
	return f
}

func main() {
	// Начальная корзина из JSON
	original := NewCart(
		false,
		"EUR",
		Shipping{method: "courier", price: 7.00},
	).
		WithItem(NewItem(1, "SHMP001", "Shampoo", false, 12.99, 2)).
		WithItem(NewItem(2, "COND001", "Conditioner", true, 10.50, 1))

	original.Print("Начальная корзина (original)")

	// ── Каждый шаг — новая корзина, original не тронут ──

	withMask := original.WithItem(NewItem(3, "MASK001", "Hair Mask", false, 18.00, 1))
	withMask.Print("После добавления Hair Mask (withMask)")

	withCoupon := withMask.WithCoupon("9003006", 20)
	withCoupon.Print("После купона 20% (withCoupon)")

	withoutCond := withCoupon.WithoutItem(2)
	withoutCond.Print("После удаления Conditioner (withoutCond)")

	// Доказательство: original не изменился
	fmt.Println("── Доказательство иммутабельности ──────")
	original.Print("original — всё ещё нетронутый")
}
