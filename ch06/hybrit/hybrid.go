package main

import (
	"errors"
	"fmt"
	"strconv"
)

// ─────────────────────────────────────────────
//  ГИБРИДНАЯ ИДЕОЛОГИЯ (самая распространённая в Go)
//
//  Cart     → МУТАБЕЛЬНАЯ  (*Cart, pointer receiver)
//             Корзина живёт как указатель и меняет себя напрямую.
//             Это оправдано: корзина — объект с последовательными
//             изменениями в рамках одного запроса.
//
//  Item     → ИММУТАБЕЛЬНАЯ (Item, value receiver)
//  Coupon   → ИММУТАБЕЛЬНАЯ (Coupon, value receiver)
//  Shipping → ИММУТАБЕЛЬНАЯ (Shipping, value receiver)
//             Эти типы — чистые данные. Они не меняются сами по себе.
//             Хочешь изменить — создай новый экземпляр.
// ─────────────────────────────────────────────

// ══════════════════════════════════════════════
//  Item — ИММУТАБЕЛЬНАЯ
//  Поля приватные. Только конструктор + геттеры + value receivers.
// ══════════════════════════════════════════════

type Item struct {
	id             int
	article        string
	name           string
	isProfessional bool
	price          float64
	discountPrice  *float64
	quantity       int
}

// NewItem — единственный способ создать Item корректно
func NewItem(id int, article, name string, isPro bool, price float64, qty int) Item {
	return Item{
		id:             id,
		article:        article,
		name:           name,
		isProfessional: isPro,
		price:          price,
		quantity:       qty,
	}
}

// Value receivers — Item никогда не меняет себя

func (i Item) ID() int        { return i.id }
func (i Item) Name() string   { return i.name }
func (i Item) Price() float64 { return i.price }
func (i Item) Quantity() int  { return i.quantity }
func (i Item) IsPro() bool    { return i.isProfessional }

func (i Item) Total() float64 {
	price := i.price
	if i.discountPrice != nil {
		price = *i.discountPrice
	}
	return round2(price * float64(i.quantity))
}

// WithQuantity возвращает НОВЫЙ Item с другим количеством
// Оригинал не тронут — иммутабельность в действии
func (i Item) WithQuantity(qty int) Item {
	i.quantity = qty // меняем копию
	return i         // возвращаем новый Item
}

// ══════════════════════════════════════════════
//  Coupon — ИММУТАБЕЛЬНАЯ
//  Создаётся один раз через конструктор, поля не меняются.
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
		discount: round2(base * float64(percent) / 100),
	}
}

func (c Coupon) Code() string      { return c.code }
func (c Coupon) Percent() int      { return c.percent }
func (c Coupon) Discount() float64 { return c.discount }

// ══════════════════════════════════════════════
//  Shipping — ИММУТАБЕЛЬНАЯ
//  Условия доставки фиксированы на момент оформления.
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

// ══════════════════════════════════════════════
//  Cart — МУТАБЕЛЬНАЯ
//
//  Используем *Cart (указатель) — это явный сигнал в Go:
//  "этот объект будет изменяться".
//
//  Все методы с pointer receiver (*Cart) меняют оригинал напрямую,
//  без создания копий. Это правильно для корзины, потому что:
//  - она живёт в рамках одного запроса
//  - изменения последовательные, история не нужна
//  - при лимите 15 товаров — никакого смысла копировать
// ══════════════════════════════════════════════

const MaxCartItems = 15

type Cart struct {
	pricingTier bool // false = BASIC, true = PRO
	items       []Item
	coupon      *Coupon // указатель — купон может отсутствовать (nil)
	shipping    Shipping
	currency    string
}

// NewCart — конструктор. Возвращает *Cart (указатель),
// потому что корзина МУТАБЕЛЬНАЯ и будет изменяться.
func NewCart(pricingTier bool, currency string, shipping Shipping) *Cart {
	return &Cart{
		pricingTier: pricingTier,
		currency:    currency,
		shipping:    shipping,
		items:       make([]Item, 0, MaxCartItems),
	}
}

// ── Pointer receivers (*Cart) — мутируют корзину напрямую ──

// AddItem добавляет товар или увеличивает количество если уже есть.
// МУТИРУЕТ корзину — c.items меняется у оригинала.
func (c *Cart) AddItem(item Item) error {
	// Проверка лимита корзины
	if c.CartCount() >= MaxCartItems {
		return errors.New("cart is full: maximum 15 items reached")
	}

	for i, existing := range c.items {
		if existing.ID() == item.ID() {
			// WithQuantity возвращает новый иммутабельный Item,
			// но мы записываем его в мутабельный слайс корзины
			c.items[i] = existing.WithQuantity(existing.Quantity() + item.Quantity())
			return nil
		}
	}

	c.items = append(c.items, item) // МУТАЦИЯ: меняем слайс оригинала
	return nil
}

// RemoveItem удаляет товар из корзины по ID.
// МУТИРУЕТ корзину — c.items меняется у оригинала.
func (c *Cart) RemoveItem(id int) error {
	for i, item := range c.items {
		if item.ID() == id {
			c.items = append(c.items[:i], c.items[i+1:]...) // МУТАЦИЯ
			// Если был купон — пересчитываем скидку от новой базы
			if c.coupon != nil {
				newCoupon := NewCoupon(c.coupon.Code(), c.coupon.Percent(), c.Base())
				c.coupon = &newCoupon
			}
			return nil
		}
	}
	return fmt.Errorf("item with id %d not found", id)
}

// ApplyCoupon применяет купон к корзине.
// МУТИРУЕТ корзину — c.coupon меняется у оригинала.
func (c *Cart) ApplyCoupon(code string, percent int) error {
	if percent <= 0 || percent > 100 {
		return errors.New("coupon percent must be between 1 and 100")
	}
	if c.Base() == 0 {
		return errors.New("cannot apply coupon to empty cart")
	}
	// Coupon создаётся как иммутабельное значение
	coupon := NewCoupon(code, percent, c.Base())
	c.coupon = &coupon // МУТАЦИЯ: записываем в корзину
	return nil
}

// RemoveCoupon убирает купон из корзины.
// МУТИРУЕТ корзину — c.coupon = nil.
func (c *Cart) RemoveCoupon() {
	c.coupon = nil // МУТАЦИЯ
}

// ── Value receivers (Cart) — только чтение, не мутируют ──

func (c *Cart) Base() float64 {
	total := 0.0
	for _, item := range c.items {
		total += item.Total() // Item.Total() — иммутабельный вызов
	}
	return round2(total)
}

func (c *Cart) FinalTotal() float64 {
	total := c.Base()
	if c.coupon != nil {
		total -= c.coupon.Discount()
	}
	total += c.shipping.Price() // Shipping.Price() — иммутабельный вызов
	return round2(total)
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

// Print выводит состояние корзины
func (c *Cart) Print(label string) {
	tier := "BASIC"
	if c.pricingTier {
		tier = "PRO"
	}
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Printf("║  %s [%s — %s]\n", label, tier, c.currency)
	fmt.Println("╠══════════════════════════════════════╣")
	for _, item := range c.items {
		pro := ""
		if item.IsPro() {
			pro = " [PRO]"
		}
		fmt.Printf("║  %-18s x%d  %.2f %s%s\n",
			item.Name(), item.Quantity(), item.Total(), c.currency, pro)
	}
	fmt.Println("╠══════════════════════════════════════╣")
	fmt.Printf("║  Товаров:   %d / %d\n", c.CartCount(), MaxCartItems)
	fmt.Printf("║  База:      %.2f %s\n", c.Base(), c.currency)
	if c.coupon != nil {
		fmt.Printf("║  Купон [%s]: -%d%% = -%.2f %s\n",
			c.coupon.Code(), c.coupon.Percent(), c.coupon.Discount(), c.currency)
	}
	fmt.Printf("║  Доставка:  +%.2f %s (%s)\n",
		c.shipping.Price(), c.currency, c.shipping.Method())
	fmt.Printf("║  ИТОГО:     %.2f %s\n", c.FinalTotal(), c.currency)
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()
}

// ─────────────────────────────────────────────

func round2(v float64) float64 {
	f, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", v), 64)
	return f
}

func main() {
	// NewCart возвращает *Cart — указатель, корзина МУТАБЕЛЬНАЯ
	cart := NewCart(
		false,
		"EUR",
		NewShipping("courier", 7.00), // Shipping — иммутабельный
	)

	// Item создаётся иммутабельным, но добавляется в мутабельную корзину
	if err := cart.AddItem(NewItem(1, "SHMP001", "Shampoo", false, 12.99, 2)); err != nil {
		fmt.Println("Error:", err)
	}
	if err := cart.AddItem(NewItem(2, "COND001", "Conditioner", true, 10.50, 1)); err != nil {
		fmt.Println("Error:", err)
	}

	cart.Print("Начальная корзина")

	// Добавляем ещё один товар — корзина мутирует
	if err := cart.AddItem(NewItem(3, "MASK001", "Hair Mask", false, 18.00, 1)); err != nil {
		fmt.Println("Error:", err)
	}
	cart.Print("После добавления Hair Mask")

	// Применяем купон — Coupon создаётся иммутабельным, записывается в корзину
	if err := cart.ApplyCoupon("9003006", 20); err != nil {
		fmt.Println("Error:", err)
	}
	cart.Print("После купона 20%")

	// Удаляем товар — корзина мутирует, купон пересчитывается
	if err := cart.RemoveItem(2); err != nil {
		fmt.Println("Error:", err)
	}
	cart.Print("После удаления Conditioner")
}
