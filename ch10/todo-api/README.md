# Todo API

REST API для управления задачами (todo-list) на Go + Gin.
Хранилище — в памяти (in-memory), данные сбрасываются при перезапуске сервера.

---

## Архитектура

Проект разбит на 4 пакета — у каждого своя ответственность:

```
todo-api/
├── main.go              ← запуск: собирает все слои вместе
├── model/
│   └── todo.go          ← структура Todo
├── service/
│   └── service.go       ← бизнес-логика (создание, поиск, удаление...)
├── handler/
│   └── handler.go       ← HTTP-слой (Gin): принимает запрос, отдаёт JSON
└── store/
    └── store.go         ← хранилище: интерфейс + реализация in-memory
```

**Как это работает вместе** — запрос идёт сверху вниз:

```
HTTP-запрос → handler → service → store → model.Todo
                ↓
          JSON-ответ клиенту
```

- **`handler`** ничего не знает о том, *как* хранятся задачи — он просто
  вызывает `service.Create(...)`, `service.GetByID(...)` и т.д.
- **`service`** ничего не знает про HTTP, Gin или JSON — работает
  только с задачами (`model.Todo`) и обычными Go-ошибками.
- **`store`** содержит интерфейс `TodoRepository` и его реализацию в памяти.
  Если завтра нужна база данных — меняем только этот файл, `handler` не трогаем.
- **`model`** — просто структура данных, ни от чего не зависит.

---

## Запуск

```bash
go mod tidy
go run main.go
# → [GIN-debug] Listening and serving HTTP on :8080
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

| Поле        | Тип      | Описание                            |
|-------------|----------|-------------------------------------|
| `id`        | integer  | Уникальный ID, растёт автоматически |
| `title`     | string   | Название задачи (обязательное)      |
| `body`      | string   | Описание (опциональное)             |
| `done`      | boolean  | Выполнена ли задача                 |
| `createdAt` | ISO 8601 | Время создания (UTC)                |
| `updatedAt` | ISO 8601 | Время последнего изменения (UTC)    |

> Поле `body` не появится в JSON если оно пустое (`omitempty`).

---

## Эндпоинты

| Метод    | URL                 | Действие             |
|----------|---------------------|----------------------|
| `POST`   | `/todos`            | Создать задачу       |
| `GET`    | `/todos`            | Список всех задач    |
| `GET`    | `/todos/:id`        | Получить одну задачу |
| `PUT`    | `/todos/:id`        | Обновить задачу      |
| `DELETE` | `/todos/:id`        | Удалить задачу       |
| `POST`   | `/todos/clear`      | Удалить все задачи   |
| `POST`   | `/todos/:id/toggle` | Переключить done     |

---

## JSON по каждому эндпоинту

### POST `/todos` — создать задачу

**Запрос**
```json
{
  "title": "Buy milk",
  "body": "2 liters"
}
```
`title` обязателен, `body` — нет.

**Ответ `201 Created`**
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

**Ответ `400 Bad Request`** (пустой title)
```json
{ "error": "title required" }
```

---

### GET `/todos` — список задач

**Query-параметры**

| Параметр | Значение      | Что вернёт           |
|----------|---------------|----------------------|
| `done`   | `1` / `true`  | Только выполненные   |
| `done`   | `0` / `false` | Только невыполненные |
| —        | без параметра | Все задачи           |

**Ответ `200 OK`**
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

**Ответ `400 Bad Request`** (неверное значение `done`)
```json
{ "error": "done must be 1|0|true|false" }
```

---

### GET `/todos/:id` — получить одну задачу

**Ответ `200 OK`** — объект Todo (см. выше)

**Ответ `400 Bad Request`** (id не число)
```json
{ "error": "invalid id" }
```

**Ответ `404 Not Found`**
```json
{ "error": "not found" }
```

---

### PUT `/todos/:id` — обновить задачу

Оба поля опциональны — что не прислано, то не меняется.

**Запрос**
```json
{ "title": "Buy bread" }
```

**Ответ `200 OK`** — обновлённый объект Todo

**Ответ `400 Bad Request`** (title прислан но пустой)
```json
{ "error": "title required" }
```

**Ответ `404 Not Found`**
```json
{ "error": "not found" }
```

---

### DELETE `/todos/:id` — удалить задачу

**Ответ `204 No Content`** — без тела

**Ответ `404 Not Found`**
```json
{ "error": "not found" }
```

---

### POST `/todos/clear` — удалить все задачи

Сбрасывает список задач. ID продолжает расти с последнего значения —
повторений не будет даже после очистки.

**Ответ `200 OK`**
```json
{ "message": "cleared" }
```

---

### POST `/todos/:id/toggle` — переключить статус

`done` меняется: `false → true → false → ...`

**Ответ `200 OK`** — обновлённый объект Todo

**Ответ `404 Not Found`**
```json
{ "error": "not found" }
```

---

## Примеры curl

```bash
# Создать
curl -s -X POST http://localhost:8080/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"Buy milk","body":"2 liters"}' | jq

# Список всех
curl -s http://localhost:8080/todos | jq

# Только невыполненные
curl -s "http://localhost:8080/todos?done=0" | jq

# Только выполненные
curl -s "http://localhost:8080/todos?done=1" | jq

# Одна задача
curl -s http://localhost:8080/todos/1 | jq

# Обновить (частично — только title)
curl -s -X PUT http://localhost:8080/todos/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Buy bread"}' | jq

# Переключить done
curl -s -X POST http://localhost:8080/todos/1/toggle | jq

# Удалить одну
curl -s -X DELETE http://localhost:8080/todos/1

# Очистить всё
curl -s -X POST http://localhost:8080/todos/clear | jq
```

---

## Коды ответов

| Код | Когда                                              |
|-----|----------------------------------------------------|
| 200 | Успешный GET / PUT / Toggle / Clear                |
| 201 | Задача создана (POST `/todos`)                     |
| 204 | Задача удалена — тела нет (DELETE)                 |
| 400 | Неверные данные (пустой title, невалидный `done`)  |
| 404 | Задача с таким ID не найдена                       |