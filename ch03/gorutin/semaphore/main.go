package main

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type Car struct {
	Brand    string
	Fuel     float64 // текущий уровень топлива
	Capacity float64 // ёмкость бака
	RateLps  float64 // расход топлива (литр/сек)
	stopped  bool
	mu       sync.Mutex
}

type GasStation struct {
	mu        sync.Mutex
	fuel      float64
	semaphore chan struct{} // ограничивает количество машин у колонок
}

func (s *GasStation) TryRefuelFull(c *Car) (ok bool, refueled float64, left float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	need := c.Capacity - c.Fuel
	need = round2(need)
	if need <= 0 {
		return false, 0, s.fuel
	}
	if need <= s.fuel {
		s.fuel = round2(s.fuel - need)
		c.Fuel = round2(c.Capacity)
		return true, need, s.fuel
	}
	return false, 0, s.fuel
}

func (c *Car) IsStopped() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stopped
}

func (c *Car) setStopped() {
	c.mu.Lock()
	c.stopped = true
	c.mu.Unlock()
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func main() {
	// создаём АЗС с двумя колонками (семафор = 2)
	station := &GasStation{
		fuel:      2000,                   // общий запас
		semaphore: make(chan struct{}, 2), // 2 «разрешения» — 2 колонки
	}

	// 15 машин, разные баки и расход
	cars := []*Car{
		{Brand: "BMW", Fuel: 20, Capacity: 60, RateLps: 0.8},
		{Brand: "Audi", Fuel: 30, Capacity: 55, RateLps: 0.5},
		{Brand: "Mercedes", Fuel: 40, Capacity: 70, RateLps: 0.45},
		{Brand: "Toyota", Fuel: 25, Capacity: 50, RateLps: 0.35},
		{Brand: "Honda", Fuel: 15, Capacity: 48, RateLps: 0.7},
		{Brand: "Ford", Fuel: 35, Capacity: 62, RateLps: 0.8},
		{Brand: "Kia", Fuel: 22, Capacity: 45, RateLps: 0.55},
		{Brand: "Hyundai", Fuel: 28, Capacity: 50, RateLps: 0.65},
		{Brand: "Volvo", Fuel: 45, Capacity: 70, RateLps: 0.6},
		{Brand: "Skoda", Fuel: 18, Capacity: 55, RateLps: 0.4},
		{Brand: "Peugeot", Fuel: 25, Capacity: 50, RateLps: 0.5},
		{Brand: "Opel", Fuel: 32, Capacity: 58, RateLps: 0.7},
		{Brand: "Nissan", Fuel: 26, Capacity: 60, RateLps: 0.6},
		{Brand: "Mazda", Fuel: 27, Capacity: 52, RateLps: 0.55},
		{Brand: "Lexus", Fuel: 24, Capacity: 65, RateLps: 0.75},

		{Brand: "BMW_E", Fuel: 20, Capacity: 50, RateLps: 0.2},
		{Brand: "Audi_E", Fuel: 30, Capacity: 55, RateLps: 0.1},
		{Brand: "Mercedes_E", Fuel: 40, Capacity: 60, RateLps: 0.25},
		{Brand: "Toyota_E", Fuel: 25, Capacity: 50, RateLps: 0.15},
		{Brand: "Honda_E", Fuel: 15, Capacity: 40, RateLps: 0.4},
		{Brand: "Ford_E", Fuel: 35, Capacity: 55, RateLps: 0.6},
		{Brand: "Kia_E", Fuel: 22, Capacity: 45, RateLps: 0.3},
		{Brand: "Hyundai_E", Fuel: 28, Capacity: 50, RateLps: 0.4},
		{Brand: "Volvo_E", Fuel: 45, Capacity: 70, RateLps: 0.4},
		{Brand: "Skoda_E", Fuel: 18, Capacity: 55, RateLps: 0.15},
		{Brand: "Peugeot_E", Fuel: 25, Capacity: 50, RateLps: 0.25},
		{Brand: "Opel_E", Fuel: 32, Capacity: 58, RateLps: 0.25},
		{Brand: "Nissan_E", Fuel: 26, Capacity: 60, RateLps: 0.25},
		{Brand: "Mazda_E", Fuel: 27, Capacity: 52, RateLps: 0.3},
		{Brand: "Lexus_E", Fuel: 24, Capacity: 65, RateLps: 0.55},
	}

	fmt.Println("🚗=== НАЧАЛО СИМУЛЯЦИИ ===")
	fmt.Printf("⛽ Запас на АЗС: %.2f л | Колонок: %d\n", station.fuel, cap(station.semaphore))
	fmt.Println("------------------------------------------------------------")

	var wg sync.WaitGroup
	wg.Add(len(cars))

	for _, car := range cars {
		go func(c *Car) {
			defer wg.Done()
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			fmt.Printf("🚙 %s выехала. Бак: %.2f/%.2f (расход %.2f л/с)\n",
				c.Brand, c.Fuel, c.Capacity, c.RateLps)

			for range ticker.C {
				if c.IsStopped() {
					return
				}

				c.Fuel = round2(c.Fuel - c.RateLps)
				if c.Fuel < 0 {
					c.Fuel = 0
				}

				fmt.Printf("[⛽ ТИК] %-9s | бак: %.2f/%.2f | расход: -%.2f л/с\n",
					c.Brand, c.Fuel, c.Capacity, c.RateLps)

				// если топливо кончилось
				if c.Fuel <= 0 {
					fmt.Printf("⚠️  %-9s: бак пуст. Еду на заправку...\n", c.Brand)
					time.Sleep(1 * time.Second)

					// ожидаем "свободную колонку"
					station.semaphore <- struct{}{} // если канал полон — ждём

					fmt.Printf("🚦 %-9s подъехала к колонке (занятая колонка)\n", c.Brand)
					ok, ref, left := station.TryRefuelFull(c)
					time.Sleep(2 * time.Second) // имитация времени заправки

					if ok {
						fmt.Printf("✅ %-9s: заправилась на %.2f л. Остаток на АЗС: %.2f л\n",
							c.Brand, ref, left)
					} else {
						fmt.Printf("❌ %-9s: не хватило топлива на АЗС (%.2f л осталось). Машина остановилась.\n",
							c.Brand, left)
						c.setStopped()
						<-station.semaphore // освобождаем колонку
						return
					}

					fmt.Printf("🏁 %-9s уезжает от колонки.\n", c.Brand)
					<-station.semaphore // освободили колонку
				}
			}
		}(car)
	}

	wg.Wait()

	fmt.Println("------------------------------------------------------------")
	stopped := 0
	for _, c := range cars {
		status := "едет"
		if c.IsStopped() {
			status = "остановилась"
			stopped++
		}
		fmt.Printf("🚗 %-9s | %s | бак: %.2f/%.2f\n", c.Brand, status, c.Fuel, c.Capacity)
	}

	station.mu.Lock()
	fmt.Printf("\n⛽ Остаток топлива на АЗС: %.2f л\n", station.fuel)
	station.mu.Unlock()

	fmt.Printf("Итого машин остановилось: %d из %d\n", stopped, len(cars))
	fmt.Println("🏁 Симуляция завершена.")
}
