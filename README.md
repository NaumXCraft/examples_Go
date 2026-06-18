### Program examples (prototypes)

- ch00 "Go json-srv **Проект JSON API Server ( Go 1.25.1 )"
- ch01 учебный, но продакшен-готовый boilerplate-проект на **Go 1.25.1 за NGINX**."
- ch02 "Games"
- ch03 "Examples"
- ch04 File Box
- ch05 WebCMD
- ch06 shopping cart (console + API)
- ch07 Mini Chat
- ch08 Computer Networks (Go + Wireshark)
- ch09 goPuTTY



## INFO:

##  Инициализируй модуль Go:

Командой:

```bash
go mod init myapp
```

* `myapp` — это имя твоего модуля (можно указать и путь, например `github.com/vladimir/myapp` если планируешь заливать на GitHub).
* В результате создаётся файл `go.mod`, где хранится имя модуля и версия Go.

Пример `go.mod`:

```go
module myapp

go 1.25
```

---

### 2. Создай файл `main.go`

Создай в корне проекта файл:

```bash
nano main.go
```

И напиши туда минимальный код:

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, Go!")
}
```
### Явная установка пакета
```
go get github.com/gorilla/mux
```
---

### 3. Запусти проект

Выполни:

```bash
go run .
```
или (эквивалентно)

```bash
go run main.go
```

Результат:

```
Hello, Go!
```

---

### 4. (Опционально) Собери бинарник

Если хочешь получить исполняемый файл:

```bash
go build
```

После этого в папке появится файл `myapp.exe` (в Windows) или просто `myapp` (в Linux/Mac).


### Go позволяет довольно просто увидеть, **какой ассемблерный код генерирует компилятор**.
Ниже — все основные способы, от простого до продвинутого.

---

###  1. Самый простой способ — через `go tool compile -S`

Допустим, у тебя есть файл `main.go`:

```go
package main

func add(a, b int) int {
    return a + b
}

func main() {
    _ = add(2, 3)
}
```

Теперь выполни в терминале:

```bash
go tool compile -S main.go
```

🔹 Это выведет **ассемблерный код**, который Go создаёт для архитектуры по умолчанию (например, `amd64`).

Пример фрагмента вывода:

```
"".add STEXT nosplit size=15 args=0x10 locals=0x0
	0x0000 00000 (main.go:3)	TEXT	"".add(SB), NOSPLIT|ABIInternal, $0-16
	0x0000 00000 (main.go:3)	MOVQ	"".b+8(SP), AX
	0x0005 00005 (main.go:3)	ADDQ	"".a+0(SP), AX
	0x0009 00009 (main.go:3)	MOVQ	AX, "".~r2+16(SP)
	0x000e 00014 (main.go:3)	RET
```

Здесь:

* `MOVQ` — перенос 64-битного значения
* `ADDQ` — сложение
* `SP` — указатель стека
* `SB` — "static base" (глобальные символы)

---

###  2. Через `go build -gcflags`

Если хочешь собрать бинарник, но при этом вывести ассемблер:

```bash
go build -gcflags="-S" main.go
```

или, чтобы подавить кучу лишних предупреждений, можно добавить:

```bash
go build -gcflags="-S -N -l" main.go
```

🔸 Флаги:

* `-N` — **отключает оптимизации**
* `-l` — **отключает инлайнинг**

Это удобно, если ты хочешь увидеть «чистый» ассемблер без оптимизаций.

---

###  3. Если хочешь смотреть ассемблер по конкретной функции

Можно использовать `go tool objdump`, чтобы дизассемблировать уже скомпилированный бинарник:

```bash
go build -o main.exe
go tool objdump -s "main.add" main.exe
```

🔹 Покажет только код функции `main.add`.

---

###  4. Альтернатива: через `go build` + `objdump` Unix
 
для Windows (через mingw64):

```bash
objdump -d main.exe
```

---

### Пример: сравнение с и без оптимизаций

### Без оптимизаций

```bash
go tool compile -S -N -l main.go
```

Ассемблер будет длинный, шаги очевидны (переменные реально копируются на стек).

### С оптимизациями

```bash
go tool compile -S main.go
```
Go может убрать стек, заинлайнить `add()` прямо в `main()`, и ты увидишь меньше кода.


## 💡 Краткое резюме
```
| Цель                       | Команда                                            |
| -------------------------- | -------------------------------------------------- |
| Посмотреть ассемблер сразу | `go tool compile -S main.go`                       |
| Отключить оптимизации      | `go tool compile -S -N -l main.go`                 |
| Дизассемблировать бинарник | `go tool objdump -s "main.func" main.exe`          |
| Через визуальный интерфейс | [https://go.godbolt.org/](https://go.godbolt.org/) |

плюс запись в файл:  go tool objdump -s "main.func" calculator.exe > dump.txt 

```



