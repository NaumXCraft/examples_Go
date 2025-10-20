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
	mu   sync.Mutex
	fuel float64
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
	station := &GasStation{fuel: 10000}

	cars := []*Car{
		{Brand: "BMW", Fuel: 20, Capacity: 60, RateLps: 0.5},
		{Brand: "Audi", Fuel: 30, Capacity: 55, RateLps: 0.3},
		{Brand: "Mercedes", Fuel: 40, Capacity: 70, RateLps: 0.4},
		{Brand: "Toyota", Fuel: 25, Capacity: 50, RateLps: 0.2},
		{Brand: "Honda", Fuel: 15, Capacity: 48, RateLps: 0.6},
		{Brand: "Ford", Fuel: 35, Capacity: 62, RateLps: 0.7},
		{Brand: "Kia", Fuel: 22, Capacity: 45, RateLps: 0.4},
		{Brand: "Hyundai", Fuel: 28, Capacity: 50, RateLps: 0.5},
		{Brand: "Volvo", Fuel: 45, Capacity: 70, RateLps: 0.55},
		{Brand: "Skoda", Fuel: 18, Capacity: 55, RateLps: 0.25},
	}

	fmt.Println("🚗=== НАЧАЛО СИМУЛЯЦИИ ===")
	fmt.Printf("⛽ Запас на АЗС: %.2f л\n", station.fuel)
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

				if c.Fuel <= 0 {
					fmt.Printf("⚠️  %-9s: бак пуст. Еду на заправку...\n", c.Brand)
					time.Sleep(1 * time.Second) // имитация пути к АЗС

					ok, ref, left := station.TryRefuelFull(c)
					if ok {
						fmt.Printf("✅ %-9s: заправилась на %.2f л. Остаток на АЗС: %.2f л\n",
							c.Brand, ref, left)
					} else {
						fmt.Printf("❌ %-9s: не хватило топлива на АЗС (%.2f л осталось). Машина остановилась.\n",
							c.Brand, left)
						c.setStopped()
						return
					}
				}
			}
		}(car)
	}

	// Ждём, пока все машины остановятся
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
