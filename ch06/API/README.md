# Cart API

Учебный проект. Гибридная идеология Go:
- `Cart`     → **МУТАБЕЛЬНАЯ** (`*Cart`, pointer receiver)
- `Item`     → иммутабельная (value receiver)
- `Coupon`   → иммутабельная (value receiver)
- `Shipping` → иммутабельная (value receiver)

## Структура

```
cart-api/
├── main.go              ← точка входа, роутер
├── go.mod
├── models/
│   └── cart.go          ← Cart, Item, Coupon, Shipping
├── handlers/
│   └── cart.go          ← HTTP хендлеры
└── storage/
    └── storage.go       ← хранилище в памяти (map + mutex)
```

## Запуск

```bash
# 1. Установить зависимости
go mod tidy

# 2. Запустить сервер
go run main.go

# Сервер стартует на http://localhost:8080
```

## Эндпоинты

| Метод    | URL                              | Действие           |
|----------|----------------------------------|--------------------|
| `GET`    | `/cart/:userID`                  | Получить корзину   |
| `POST`   | `/cart/:userID/items`            | Добавить товар     |
| `DELETE` | `/cart/:userID/items/:itemID`    | Удалить товар      |
| `POST`   | `/cart/:userID/coupon`           | Применить купон    |
| `DELETE` | `/cart/:userID/coupon`           | Убрать купон       |
| `DELETE` | `/cart/:userID`                  | Очистить корзину   |

---

## Примеры запросов (curl)

### Получить корзину
```bash
curl http://localhost:8080/cart/user-1
```

### Добавить товар
```bash
curl -X POST http://localhost:8080/cart/user-1/items \
  -H "Content-Type: application/json" \
  -d '{
    "id": 1,
    "article": "SHMP001",
    "name": "Shampoo",
    "category": "Hiustenhoito",
    "country": "Suomi",
    "image": "http://localhost:8000/storage/products/shampoo.jpg",
    "is_professional": false,
    "is_active": true,
    "price": 12.99,
    "discount_price": null,
    "quantity": 2
  }'
```

### Добавить ещё один товар
```bash
curl -X POST http://localhost:8080/cart/user-1/items \
  -H "Content-Type: application/json" \
  -d '{
    "id": 2,
    "article": "COND001",
    "name": "Conditioner",
    "category": "Hiustenhoito",
    "country": "Suomi",
    "image": "http://localhost:8000/storage/products/conditioner.jpg",
    "is_professional": false,
    "is_active": true,
    "price": 10.50,
    "discount_price": null,
    "quantity": 1
  }'
```

### Применить купон
```bash
curl -X POST http://localhost:8080/cart/user-1/coupon \
  -H "Content-Type: application/json" \
  -d '{
    "code": "9003006",
    "percent": 20
  }'
```

### Удалить товар
```bash
curl -X DELETE http://localhost:8080/cart/user-1/items/2
```

### Убрать купон
```bash
curl -X DELETE http://localhost:8080/cart/user-1/coupon
```

### Очистить корзину
```bash
curl -X DELETE http://localhost:8080/cart/user-1
```

---

## Пример ответа

```json
{
  "message": "cart_fetched",
  "data": {
    "pricing_tier": false,
    "items": [
      {
        "id": 1,
        "article": "SHMP001",
        "name": "Shampoo",
        "category": "Hiustenhoito",
        "country": "Suomi",
        "image": "http://localhost:8000/storage/products/shampoo.jpg",
        "is_professional": false,
        "is_active": true,
        "price": "12.99",
        "discount_price": null,
        "quantity": 2,
        "total": "25.98"
      }
    ],
    "base": "25.98",
    "cart_count": 2,
    "shipping": {
      "method": "courier",
      "price": "7.00"
    },
    "coupon": null,
    "final_total": "32.98",
    "currency": "EUR",
    "warnings": []
  }
}
```
