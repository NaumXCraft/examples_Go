package storage

import (
	"fmt"
	"sync"

	"cart-api/models"
)

// ─────────────────────────────────────────────
//  Storage — хранилище корзин в памяти
//
//  Корзины живут в map: userID → *Cart
//  sync.RWMutex защищает map от race condition,
//  потому что HTTP сервер обрабатывает запросы
//  в разных горутинах одновременно.
// ─────────────────────────────────────────────

type Storage struct {
	mu    sync.RWMutex
	carts map[string]*models.Cart
}

func New() *Storage {
	return &Storage{
		carts: make(map[string]*models.Cart),
	}
}

// GetCart возвращает корзину пользователя.
// Если корзины нет — создаёт новую с дефолтными настройками.
func (s *Storage) GetCart(userID string) *models.Cart {
	s.mu.RLock()
	cart, exists := s.carts[userID]
	s.mu.RUnlock()

	if exists {
		return cart
	}

	// Корзины нет — создаём новую
	return s.createCart(userID)
}

// createCart создаёт новую корзину и сохраняет её
func (s *Storage) createCart(userID string) *models.Cart {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check: вдруг другая горутина уже создала пока мы ждали Lock
	if cart, exists := s.carts[userID]; exists {
		return cart
	}

	cart := models.NewCart(
		false,
		"EUR",
		models.NewShipping("courier", 7.00),
	)
	s.carts[userID] = cart
	return cart
}

// SaveCart явно сохраняет корзину (нужно после изменений)
func (s *Storage) SaveCart(userID string, cart *models.Cart) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.carts[userID] = cart
}

// DeleteCart удаляет корзину пользователя
func (s *Storage) DeleteCart(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.carts[userID]; !exists {
		return fmt.Errorf("cart for user %s not found", userID)
	}

	delete(s.carts, userID)
	return nil
}

// Count возвращает количество активных корзин (для отладки)
func (s *Storage) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.carts)
}
