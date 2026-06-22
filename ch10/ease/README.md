# Todo API (простая версия)

REST API для управления задачами (todo-list) на Go + Gin.  
Весь код — **в одном файле `main.go`**, без разделения на пакеты.  
Хранилище — в памяти (in-memory), данные сбрасываются при перезапуске сервера.

---

## Структура файла

Файл состоит из 4 секций, расположенных одна за другой:

```
main.go
├── МОДЕЛЬ          ← структура Todo
├── ХРАНИЛИЩЕ        ← map с задачами + функции addTodo/getTodo/updateTodo...
├── HTTP-ХЕНДЛЕРЫ    ← функции handleCreate/handleList/handleUpdate...
└── MAIN             ← запуск сервера, регистрация маршрутов
```

Поток запроса:

```
HTTP-запрос → handleXxx() → xxxTodo() → map todos → JSON-ответ
```

- **Хендлеры** (`handleCreate`, `handleGet`...) разбирают HTTP-запрос и формируют ответ.
- **Функции хранилища** (`addTodo`, `getTodo`...) делают саму работу с задачами.
- Между ними — обычные Go-структуры и `error`, никакого Gin внутри хранилища нет.

---

## Запуск

```bash
go mod init todo-api
go get github.com/gin-gonic/gin
go run main.go
# → [GIN-debug] Listening on :8080
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
|-------------|----------|--------------------------------------|
| `id`        | integer  | Уникальный ID, растёт автоматически  |
| `title`     | string   | Название задачи (обязательное)       |
| `body`      | string   | Описание (опционально)               |
| `done`      | boolean  | Выполнена ли задача                  |
| `createdAt` | ISO 8601 | Время создания (UTC)                 |
| `updatedAt` | ISO 8601 | Время последнего изменения           |

> Поле `body` не появится в JSON, если оно пустое (`omitempty`).

---

## Эндпоинты

| Метод    | URL                  | Действие              |
|----------|----------------------|------------------------|
| `POST`   | `/todos`             | Создать задачу         |
| `GET`    | `/todos`              | Список всех задач      |
| `GET`    | `/todos/:id`          | Получить одну задачу   |
| `PUT`    | `/todos/:id`          | Обновить задачу        |
| `DELETE` | `/todos/:id`          | Удалить задачу         |
| `POST`   | `/todos/:id/toggle`   | Переключить done       |
| `POST`   | `/todos/clear`        | Удалить все задачи     |

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

| Параметр | Значение      | Что вернёт            |
|----------|---------------|------------------------|
| `done`   | `1` / `true`  | Только выполненные     |
| `done`   | `0` / `false` | Только невыполненные   |
| —        | без параметра | Все задачи             |

**Ответ `200 OK`**
```json
{
  "count": 2,
  "items": [
    {
      "id": 1,
      "title": "Buy milk",
      "body": "2 liters",
      "done": false,
      "createdAt": "2024-06-18T10:00:00Z",
      "updatedAt": "2024-06-18T10:00:00Z"
    },
    {
      "id": 2,
      "title": "Call dentist",
      "done": true,
      "createdAt": "2024-06-18T11:00:00Z",
      "updatedAt": "2024-06-18T12:30:00Z"
    }
  ]
}
```

> Если задач нет — `"items": []`, никогда не `null`.

**Ответ `400 Bad Request`** (неверный фильтр)
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

**Ответ `400 Bad Request`** (title прислан, но пустой)
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

### POST `/todos/:id/toggle` — переключить статус

`done` меняется: `false → true → false → ...`

**Ответ `200 OK`** — обновлённый объект Todo

**Ответ `404 Not Found`**
```json
{ "error": "not found" }
```

---

### POST `/todos/clear` — удалить всё

Сбрасывает список задач и счётчик ID (следующая задача снова получит `id: 1`).

**Ответ `200 OK`**
```json
{ "message": "cleared" }
```

---

## Примеры curl

```
# Создать
$body = '{"title":"Buy milk","body":"2 liters"}'
curl.exe -s -X POST http://localhost:8080/todos -H "Content-Type: application/json" -d $body | ConvertFrom-Json

# Список всех
curl.exe -s http://localhost:8080/todos | ConvertFrom-Json

# Только невыполненные
curl.exe -s "http://localhost:8080/todos?done=0" | ConvertFrom-Json

# Одна задача
curl.exe -s http://localhost:8080/todos/1 | ConvertFrom-Json

# Обновить
$body = '{"title":"Buy bread"}'
curl.exe -s -X PUT http://localhost:8080/todos/1 -H "Content-Type: application/json" -d $body | ConvertFrom-Json

# Переключить done
curl.exe -s -X POST http://localhost:8080/todos/1/toggle | ConvertFrom-Json

# Удалить
curl.exe -s -X DELETE http://localhost:8080/todos/1

# Очистить всё
curl.exe -s -X POST http://localhost:8080/todos/clear | ConvertFrom-Json
```

### запросы в Postman:

---

**Создать**
- Method: `POST`
- URL: `http://localhost:8080/todos`
- Body → raw → JSON:
```json
{"title":"Buy milk","body":"2 liters"}
```

---

**Список всех**
- Method: `GET`
- URL: `http://localhost:8080/todos`

---

**Только невыполненные**
- Method: `GET`
- URL: `http://localhost:8080/todos?done=0`

---

**Одна задача**
- Method: `GET`
- URL: `http://localhost:8080/todos/1`

---

**Обновить**
- Method: `PUT`
- URL: `http://localhost:8080/todos/1`
- Body → raw → JSON:
```json
{"title":"Buy bread"}
```

---

**Переключить done**
- Method: `POST`
- URL: `http://localhost:8080/todos/1/toggle`

---

**Удалить**
- Method: `DELETE`
- URL: `http://localhost:8080/todos/1`

---

**Очистить всё**
- Method: `POST`
- URL: `http://localhost:8080/todos/clear`

---

Для всех запросов с Body не забудь выставить заголовок `Content-Type: application/json` — или просто выбери **raw → JSON** в Postman, он добавит его автоматически.


---

## Коды ответов

| Код | Когда                                          |
|-----|--------------------------------------------------|
| 200 | Успешный GET / Update / Toggle / Clear           |
| 201 | Задача создана                                   |
| 204 | Задача удалена — тела нет                        |
| 400 | Неверные данные (пустой title, плохой `id`/`done`) |
| 404 | Задача с таким ID не найдена                     |

---

## Важно про `mu sync.Mutex`

Gin обрабатывает каждый HTTP-запрос в своей горутине (Go-потоке).  
Если два запроса одновременно попытаются изменить `map todos`,
это вызовет панику или повреждение данных.

`mu.Lock()` / `mu.Unlock()` в начале каждой функции хранилища
гарантируют, что в любой момент времени с `todos` работает только
один запрос — остальные ждут своей очереди.
