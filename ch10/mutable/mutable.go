package main

import (
	"fmt"
	"strconv"
)

// ─────────────────────────────────────────────
//  МУТАБЕЛЬНАЯ ИДЕОЛОГИЯ
//  Pointer receivers (*T) — методы меняют оригинал.
//  Cart живёт как *Cart и модифицирует себя напрямую.
// ─────────────────────────────────────────────

type Item struct {
	ID             int
	Article        string
	Name           string
	IsProfessional bool
	Price          float64
	DiscountPrice  *float64
	Quantity       int
}

type Coupon struct {
	Code     string
	Percent  int
	Discount float64
}

type Shipping struct {
	Method string
	Price  float64
}

type Cart struct {
	PricingTier bool // false = BASIC, true = PRO
	Items       []Item
	Coupon      *Coupon
	Shipping    Shipping
	Currency    string
}

// ── Методы с pointer receiver (*Cart) ──────────

// AddItem добавляет товар в корзину (мутирует оригинал)
func (c *Cart) AddItem(item Item) {
	for i, existing := range c.Items {
		if existing.ID == item.ID {
			c.Items[i].Quantity += item.Quantity // мутируем существующий
			return
		}
	}
	c.Items = append(c.Items, item) // мутируем слайс
}

// RemoveItem удаляет товар по ID
func (c *Cart) RemoveItem(id int) {
	filtered := c.Items[:0]
	for _, item := range c.Items {
		if item.ID != id {
			filtered = append(filtered, item)
		}
	}
	c.Items = filtered // мутируем поле
}

// ApplyCoupon применяет купон прямо к корзине
func (c *Cart) ApplyCoupon(code string, percent int) {
	base := c.Base()
	discount := base * float64(percent) / 100
	c.Coupon = &Coupon{ // мутируем поле Coupon
		Code:     code,
		Percent:  percent,
		Discount: round2(discount),
	}
}

// Base считает сумму без скидок и доставки
func (c *Cart) Base() float64 {
	total := 0.0
	for _, item := range c.Items {
		price := item.Price
		if item.DiscountPrice != nil {
			price = *item.DiscountPrice
		}
		total += price * float64(item.Quantity)
	}
	return round2(total)
}

// FinalTotal = base - coupon + shipping
func (c *Cart) FinalTotal() float64 {
	total := c.Base()
	if c.Coupon != nil {
		total -= c.Coupon.Discount
	}
	total += c.Shipping.Price
	return round2(total)
}

// CartCount суммарное кол-во единиц товара
func (c *Cart) CartCount() int {
	count := 0
	for _, item := range c.Items {
		count += item.Quantity
	}
	return count
}

// Print выводит состояние корзины
func (c *Cart) Print(label string) {
	tier := "BASIC"
	if c.PricingTier {
		tier = "PRO"
	}
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Printf("║  %s [%s — %s]\n", label, tier, c.Currency)
	fmt.Println("╠══════════════════════════════════════╣")
	for _, item := range c.Items {
		pro := ""
		if item.IsProfessional {
			pro = " [PRO]"
		}
		fmt.Printf("║  %-18s x%d  %.2f %s%s\n",
			item.Name, item.Quantity, item.Price*float64(item.Quantity), c.Currency, pro)
	}
	fmt.Println("╠══════════════════════════════════════╣")
	fmt.Printf("║  Товаров:      %d шт.\n", c.CartCount())
	fmt.Printf("║  База:         %.2f %s\n", c.Base(), c.Currency)
	if c.Coupon != nil {
		fmt.Printf("║  Купон [%s]: -%d%% = -%.2f %s\n",
			c.Coupon.Code, c.Coupon.Percent, c.Coupon.Discount, c.Currency)
	}
	fmt.Printf("║  Доставка:     +%.2f %s (%s)\n",
		c.Shipping.Price, c.Currency, c.Shipping.Method)
	fmt.Printf("║  ИТОГО:        %.2f %s\n", c.FinalTotal(), c.Currency)
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
	cart := &Cart{ // указатель — сигнал: "этот объект будет мутировать"
		PricingTier: false,
		Currency:    "EUR",
		Shipping:    Shipping{Method: "courier", Price: 7.00},
		Items: []Item{
			{ID: 1, Article: "SHMP001", Name: "Shampoo", Price: 12.99, Quantity: 2},
			{ID: 2, Article: "COND001", Name: "Conditioner", IsProfessional: true, Price: 10.50, Quantity: 1},
		},
	}

	cart.Print("Начальная корзина")

	// ── Мутация 1: добавляем товар ──
	cart.AddItem(Item{ID: 3, Article: "MASK001", Name: "Hair Mask", Price: 18.00, Quantity: 1})
	cart.Print("После добавления Hair Mask")

	// ── Мутация 2: применяем купон ──
	cart.ApplyCoupon("9003006", 20)
	cart.Print("После купона 20%")

	// ── Мутация 3: удаляем товар ──
	cart.RemoveItem(2)
	cart.Print("После удаления Conditioner")
}
