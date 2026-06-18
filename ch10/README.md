# Todo API

REST API для управления задачами (todo-list) на Go + Gin.  
Хранилище — in-memory (перезапуск сбрасывает данные).

---

## Структура проекта

```
todo-api/
├── main.go                      ← запуск, wire-up
├── models/
│   └── todo.go                  ← struct Todo
├── service/
│   ├── todo_service.go          ← бизнес-логика, sentinel-ошибки
│   └── todo_service_test.go     ← unit-тесты
├── handler/
│   └── todo_handler.go          ← HTTP-хендлеры (Gin)
└── response/
    └── response.go              ← JSON-контракты (типы ответов)
```

---

## Запуск

```bash
go mod tidy
go run main.go
# → [GIN-debug] Listening on :8080
```

Тесты:

```bash
go test ./...
```

---

## Модель данных

### Todo

```json
{
  "id":        1,
  "title":     "Buy milk",
  "body":      "2 liters",
  "done":      false,
  "createdAt": "2024-06-18T10:00:00Z",
  "updatedAt": "2024-06-18T10:00:00Z"
}
```

| Поле        | Тип      | Описание                        |
|-------------|----------|---------------------------------|
| `id`        | integer  | Уникальный ID, автоинкремент    |
| `title`     | string   | Название задачи (обязательное)  |
| `body`      | string   | Описание (опционально)          |
| `done`      | boolean  | Статус выполнения               |
| `createdAt` | ISO 8601 | Время создания (UTC)            |
| `updatedAt` | ISO 8601 | Время последнего изменения      |

> `body` отсутствует в JSON если пустой (`omitempty`).

---

## Эндпоинты

| Метод    | URL                  | Действие              |
|----------|----------------------|-----------------------|
| `POST`   | `/todos`             | Создать задачу        |
| `GET`    | `/todos`             | Список всех задач     |
| `GET`    | `/todos/:id`         | Получить одну задачу  |
| `PUT`    | `/todos/:id`         | Обновить задачу       |
| `DELETE` | `/todos/:id`         | Удалить задачу        |
| `POST`   | `/todos/:id/toggle`  | Переключить done      |
| `POST`   | `/todos/clear`       | Удалить все задачи    |

---

## JSON-контракты

### POST `/todos` — создать задачу

**Request**
```json
{
  "title": "Buy milk",
  "body": "2 liters"
}
```

| Поле    | Обязательное | Описание       |
|---------|:------------:|----------------|
| `title` | ✅           | Название       |
| `body`  | ❌           | Описание       |

**Response `201 Created`**
```json
{
  "id":        1,
  "title":     "Buy milk",
  "body":      "2 liters",
  "done":      false,
  "createdAt": "2024-06-18T10:00:00Z",
  "updatedAt": "2024-06-18T10:00:00Z"
}
```

**Response `400 Bad Request`** (пустой title)
```json
{ "error": "title required" }
```

---

### GET `/todos` — список задач

**Query-параметры**

| Параметр | Значения          | Описание              |
|----------|-------------------|-----------------------|
| `done`   | `1`, `true`       | Только выполненные    |
| `done`   | `0`, `false`      | Только невыполненные  |
| —        | (без параметра)   | Все задачи            |

**Response `200 OK`**
```json
{
  "count": 2,
  "items": [
    {
      "id":        1,
      "title":     "Buy milk",
      "body":      "2 liters",
      "done":      false,
      "createdAt": "2024-06-18T10:00:00Z",
      "updatedAt": "2024-06-18T10:00:00Z"
    },
    {
      "id":        2,
      "title":     "Call dentist",
      "done":      true,
      "createdAt": "2024-06-18T11:00:00Z",
      "updatedAt": "2024-06-18T12:30:00Z"
    }
  ]
}
```

> Если задач нет — `"items": []`, никогда не `null`.

**Response `400 Bad Request`** (неверный фильтр)
```json
{ "error": "done must be 1|0|true|false" }
```

---

### GET `/todos/:id` — получить задачу

**Response `200 OK`**
```json
{
  "id":        1,
  "title":     "Buy milk",
  "body":      "2 liters",
  "done":      false,
  "createdAt": "2024-06-18T10:00:00Z",
  "updatedAt": "2024-06-18T10:00:00Z"
}
```

**Response `404 Not Found`**
```json
{ "error": "not found" }
```

---

### PUT `/todos/:id` — обновить задачу

Оба поля опциональны. Отсутствующее поле не изменяется.

**Request** (только title)
```json
{ "title": "Buy bread" }
```

**Request** (оба поля)
```json
{
  "title": "Buy bread",
  "body":  "rye bread, 1 loaf"
}
```

**Response `200 OK`**
```json
{
  "id":        1,
  "title":     "Buy bread",
  "body":      "rye bread, 1 loaf",
  "done":      false,
  "createdAt": "2024-06-18T10:00:00Z",
  "updatedAt": "2024-06-18T13:00:00Z"
}
```

**Response `400 Bad Request`** (title передан, но пустой)
```json
{ "error": "title required" }
```

**Response `404 Not Found`**
```json
{ "error": "not found" }
```

---

### DELETE `/todos/:id` — удалить задачу

**Response `204 No Content`** — нет тела

**Response `404 Not Found`**
```json
{ "error": "not found" }
```

---

### POST `/todos/:id/toggle` — переключить статус

Меняет `done`: `false → true → false → ...`

**Response `200 OK`**
```json
{
  "id":        1,
  "title":     "Buy milk",
  "done":      true,
  "createdAt": "2024-06-18T10:00:00Z",
  "updatedAt": "2024-06-18T14:00:00Z"
}
```

**Response `404 Not Found`**
```json
{ "error": "not found" }
```

---

### POST `/todos/clear` — удалить все задачи

Сбрасывает и данные, и счётчик ID (следующий ID снова будет `1`).

**Response `200 OK`**
```json
{ "message": "cleared" }
```

---

## Примеры curl

```bash
# Создать задачу
curl -s -X POST http://localhost:8080/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"Buy milk","body":"2 liters"}' | jq

# Список всех
curl -s http://localhost:8080/todos | jq

# Только выполненные
curl -s "http://localhost:8080/todos?done=1" | jq

# Только невыполненные
curl -s "http://localhost:8080/todos?done=0" | jq

# Получить одну
curl -s http://localhost:8080/todos/1 | jq

# Обновить title
curl -s -X PUT http://localhost:8080/todos/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Buy bread"}' | jq

# Переключить done
curl -s -X POST http://localhost:8080/todos/1/toggle | jq

# Удалить
curl -s -X DELETE http://localhost:8080/todos/1

# Очистить всё
curl -s -X POST http://localhost:8080/todos/clear | jq
```

---

## Коды ответов

| Код | Когда                                   |
|-----|-----------------------------------------|
| 200 | Успешный GET / Toggle / Clear / Update  |
| 201 | Задача создана (POST /todos)            |
| 204 | Задача удалена (DELETE) — нет тела      |
| 400 | Неверные данные запроса                 |
| 404 | Задача не найдена                       |
| 405 | Неверный HTTP-метод                     |
