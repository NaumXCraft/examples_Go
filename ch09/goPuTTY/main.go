// ═════════════════════════════════════════════════════════════════════════════
//                             УЧЕБНЫЙ SSH-КЛИЕНТ НА GO
//                         "GoPuTTY Demo" — тренажёр по Linux
// ═════════════════════════════════════════════════════════════════════════════
//
// Это консольная программа на Go, которая:
//   • Подключается к реальному Linux-серверу по SSH
//   • Предлагает ученику меню с заданиями (создать папку, прочитать файл и т.д.)
//   • Автоматически проверяет, правильно ли выполнено задание
//   • При ошибках доступа — подсказывает, как получить root через sudo/su
//   • Ведёт полный лог всех действий ученика в файл student_log_ИМЯ_ВРЕМЯ.txt
//   • Идеально подходит для уроков по Linux, пентесту, DevOps-тренажёров
//
// Автор: твой друг Grok + ты
// Версия: 2.0 (с заданиями и логами)
// Требования: go1.21+
// Зависимости: go get golang.org/x/crypto/ssh github.com/fatih/color
//
// Запуск:
//   go run main.go
// ═════════════════════════════════════════════════════════════════════════════

package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"golang.org/x/crypto/ssh"
)

// ═════════════════════════════════════════════════════════════════════════════
//                             ГЛОБАЛЬНЫЕ ПЕРЕМЕННЫЕ
// ═════════════════════════════════════════════════════════════════════════════

var (
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
	bold   = color.New(color.Bold).SprintFunc()
	hiBlue = color.New(color.FgHiBlue).SprintFunc()

	logFile     *os.File
	logWriter   *log.Logger
	studentName string
)

// SSHSession — обёртка над SSH-соединением с удобными методами
type SSHSession struct {
	client      *ssh.Client
	session     *ssh.Session
	stdin       io.WriteCloser // Для отправки команд в shell
	outputChan  chan string
	currentUser string
	isRoot      bool
}

// Task — описание одного задания
type Task struct {
	ID          int
	Title       string
	Description string
	Command     string                 // какую команду должен выполнить ученик
	CheckFunc   func(*SSHSession) bool // или кастомная проверка
}

// ═════════════════════════════════════════════════════════════════════════════
//                               ОСНОВНЫЕ ЗАДАНИЯ
// ═════════════════════════════════════════════════════════════════════════════

var tasks = []Task{
	{
		ID:          1,
		Title:       "Создать каталог",
		Description: "Создайте каталог /tmp/go_training_2025",
		Command:     "mkdir /tmp/go_training_2025",
		CheckFunc: func(s *SSHSession) bool {
			s.sendCommand("test -d /tmp/go_training_2025 && echo OK || echo NO")
			out := s.readUntilPrompt(3)
			return strings.Contains(out, "OK")
		},
	},
	{
		ID:          2,
		Title:       "Создать файл с содержимым",
		Description: "Создайте файл ~/hello.txt с текстом \"Hello from Go SSH Demo!\"",
		Command:     "echo 'Hello from Go SSH Demo!' > ~/hello.txt",
		CheckFunc: func(s *SSHSession) bool {
			s.sendCommand("cat ~/hello.txt")
			out := s.readUntilPrompt(3)
			return strings.Contains(out, "Hello from Go SSH Demo!")
		},
	},
	{
		ID:          3,
		Title:       "Прочитать секретный файл (нужен root)",
		Description: "Прочитайте первые 3 строки /etc/shadow",
		Command:     "head -3 /etc/shadow",
		CheckFunc: func(s *SSHSession) bool {
			s.sendCommand("head -3 /etc/shadow 2>/dev/null | wc -l")
			out := s.readUntilPrompt(3)
			return strings.TrimSpace(out) == "3"
		},
	},
	{
		ID:          4,
		Title:       "Стать root",
		Description: "Получите root-доступ (sudo -i или su)",
		Command:     "", // вручную
		CheckFunc: func(s *SSHSession) bool {
			return s.isRoot
		},
	},
}

// ═════════════════════════════════════════════════════════════════════════════
//                                   MAIN
// ═════════════════════════════════════════════════════════════════════════════

func main() {
	fmt.Println(hiBlue(bold("╔══════════════════════════════════════════════════════════╗")))
	fmt.Println(hiBlue(bold("║               УЧЕБНЫЙ SSH-КЛИЕНТ НА GO                   ║")))
	fmt.Println(hiBlue(bold("║               «GoPuTTY Demo» v2.0                        ║")))
	fmt.Println(hiBlue(bold("╚══════════════════════════════════════════════════════════╝")))

	studentName = ask("Как вас зовут ученика?", "student")

	// Создаём лог-файл
	t := time.Now().Format("2006-01-02_15-04-05")
	logFileName := fmt.Sprintf("log_%s_%s.txt", studentName, t)
	var err error
	logFile, err = os.Create(logFileName)
	if err != nil {
		log.Fatalf("Не удалось создать лог: %v", err)
	}
	defer logFile.Close()

	logWriter = log.New(logFile, "", log.LstdFlags)
	logWriter.Println("=== СЕССИЯ НАЧАТА ===")
	logWriter.Printf("Ученик: %s | Время: %s\n", studentName, time.Now().Format(time.RFC1123))

	host := ask("IP сервера", "192.168.1.100")
	port := ask("Порт", "22")
	user := ask("Логин", "student")

	fmt.Print("Пароль: ")
	password := readPassword()
	logWriter.Printf("Подключение: %s@%s:%s\n", user, host, port)

	// Подключение по SSH
	client, err := ssh.Dial("tcp", host+":"+port, &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Только для учебной сети!
		Timeout:         15 * time.Second,
	})
	if err != nil {
		logWriter.Printf("ОШИБКА ПОДКЛЮЧЕНИЯ: %v\n", err)
		log.Fatalf("%s Ошибка подключения: %v", red(""), err)
	}
	defer client.Close()

	session, stdin, outputChan := createShellSession(client)

	sshSess := &SSHSession{
		client:      client,
		session:     session,
		stdin:       stdin,
		outputChan:  outputChan,
		currentUser: user,
	}

	// Определяем текущего пользователя
	sshSess.sendCommand("whoami")
	sshSess.currentUser = strings.TrimSpace(sshSess.readUntilPrompt(3))
	sshSess.isRoot = (sshSess.currentUser == "root")

	fmt.Printf("\n%s Подключено! Пользователь: %s\n", green("Успех!"), bold(sshSess.currentUser))
	logWriter.Printf("Успешное подключение. Пользователь: %s (root: %v)\n", sshSess.currentUser, sshSess.isRoot)

	// Главный цикл — меню заданий
	for {
		printMainMenu(sshSess.isRoot)
		choice := readInt()

		if choice == 0 {
			fmt.Println(yellow("До новых встреч!"))
			logWriter.Println("=== СЕССИЯ ЗАВЕРШЕНА ===\n")
			return
		}

		if choice >= 1 && choice <= len(tasks) {
			runTask(sshSess, &tasks[choice-1])
		} else if choice == 99 {
			sshSess.customCommand()
		} else {
			fmt.Println(red("Неверный пункт!"))
		}
	}
}

// ═════════════════════════════════════════════════════════════════════════════
//                            ВЫПОЛНЕНИЕ ЗАДАНИЯ
// ═════════════════════════════════════════════════════════════════════════════

// runTask — выполняет одно задание с проверкой и логированием
func runTask(s *SSHSession, task *Task) {
	fmt.Printf("\n%s %s\n", hiBlue("ЗАДАНИЕ"), bold(task.ID), bold(task.Title))
	fmt.Println(cyan(task.Description))
	if task.Command != "" {
		fmt.Printf("Подсказка: %s\n", yellow(task.Command))
	}
	fmt.Println(strings.Repeat("─", 60))

	logWriter.Printf("Задание %d начато: %s\n", task.ID, task.Title)

	// Даём ученику возможность выполнить команду вручную
	fmt.Print("Нажмите Enter, когда выполните задание, я проверю...")
	fmt.Scanln()

	// Проверка результата
	success := task.CheckFunc(s)

	if success {
		fmt.Printf("%s Задание %d выполнено успешно!\n", green("Победа!"), task.ID)
		logWriter.Printf("Задание %d — УСПЕШНО\n", task.ID)
	} else {
		fmt.Printf("%s Задание %d НЕ выполнено :(\n", red("Ошибка!"), task.ID)
		logWriter.Printf("Задание %d — ПРОВАЛЕНО\n", task.ID)

		// Подсказки при типичных ошибках
		if task.ID == 3 && !s.isRoot {
			fmt.Println(yellow("Подсказка: для чтения /etc/shadow нужны права root"))
			if askYesNo("Хотите попробовать стать root прямо сейчас?") {
				s.tryBecomeRoot()
			}
		}
	}

	fmt.Println()
}

// ═════════════════════════════════════════════════════════════════════════════
//                         ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ SSH
// ═════════════════════════════════════════════════════════════════════════════

// createShellSession создаёт интерактивную SSH-сессию с shell
func createShellSession(client *ssh.Client) (*ssh.Session, io.WriteCloser, chan string) {
	session, err := client.NewSession()
	if err != nil {
		log.Fatal("Ошибка сессии:", err)
	}

	session.RequestPty("xterm-256color", 80, 40, ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 115200,
		ssh.TTY_OP_OSPEED: 115200,
	})

	stdin, err := session.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := session.Shell(); err != nil {
		log.Fatal("Не удалось запустить shell:", err)
	}

	output := make(chan string, 200)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			output <- line
			// Также пишем в лог всё, что приходит с сервера
			if logWriter != nil {
				logWriter.Printf("[SERVER] %s\n", line)
			}
		}
	}()

	return session, stdin, output
}

// sendCommand отправляет команду в SSH-сессию
func (s *SSHSession) sendCommand(cmd string) {
	if cmd == "" {
		return
	}
	logWriter.Printf("[SEND] %s\n", cmd)
	s.stdin.Write([]byte(cmd + "\n"))
}

// readUntilPrompt читает вывод до таймаута (в секундах)
func (s *SSHSession) readUntilPrompt(seconds int) string {
	timeout := time.After(time.Duration(seconds) * time.Second)
	var lines []string
	for {
		select {
		case line := <-s.outputChan:
			lines = append(lines, line)
		case <-timeout:
			return strings.Join(lines, "\n")
		}
	}
}

// tryBecomeRoot — пытается стать root через sudo -i
func (s *SSHSession) tryBecomeRoot() {
	fmt.Println(yellow("Пытаемся стать root..."))
	s.sendCommand("sudo -i")

	time.Sleep(800 * time.Millisecond)
	if s.waitFor("password", 4) {
		fmt.Print("Введите пароль для sudo: ")
		pass := readPassword()
		s.sendCommand(pass)

		// Проверяем
		s.sendCommand("whoami")
		out := s.readUntilPrompt(4)
		if strings.Contains(out, "root") {
			s.isRoot = true
			s.currentUser = "root"
			fmt.Println(green("Успех! Теперь вы root!"))
			logWriter.Println("Получены права root")
		} else {
			fmt.Println(red("Не удалось стать root"))
		}
	}
}

// waitFor — ждёт появления строки с подстрокой (регистронезависимо)
func (s *SSHSession) waitFor(phrase string, seconds int) bool {
	timeout := time.After(time.Duration(seconds) * time.Second)
	for {
		select {
		case line := <-s.outputChan:
			if strings.Contains(strings.ToLower(line), strings.ToLower(phrase)) {
				return true
			}
		case <-timeout:
			return false
		}
	}
}

// customCommand — выполнение произвольной команды учеником
func (s *SSHSession) customCommand() {
	cmd := ask("Введите команду", "")
	s.sendCommand(cmd)
	fmt.Println(s.readUntilPrompt(6))
}

// ═════════════════════════════════════════════════════════════════════════════
//                             УТИЛИТЫ UI И ВВОДА
// ═════════════════════════════════════════════════════════════════════════════

func printMainMenu(isRoot bool) {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(bold("                   МЕНЮ ЗАДАНИЙ"))
	fmt.Println(strings.Repeat("=", 60))
	for _, t := range tasks {
		status := " "
		if isRoot && t.ID == 4 {
			status = green("Выполнено")
		}
		fmt.Printf("%s %d. %s %s\n", hiBlue("["), t.ID, t.Title, status)
	}
	fmt.Println(hiBlue("[99]") + " Произвольная команда")
	fmt.Println(hiBlue("[0]") + " Выход")
	if isRoot {
		fmt.Println(green("\nВы работаете от имени root!"))
	}
	fmt.Print("\nВыберите задание → ")
}

func ask(prompt, def string) string {
	fmt.Printf("%s [%s]: ", prompt, def)
	var input string
	fmt.Scanln(&input)
	if input == "" {
		return def
	}
	return input
}

func askYesNo(prompt string) bool {
	fmt.Printf("%s (y/n): ", prompt)
	var s string
	fmt.Scanln(&s)
	return strings.ToLower(s) == "y" || strings.ToLower(s) == "да"
}

func readInt() int {
	var n int
	fmt.Scanf("%d", &n)
	var dummy string
	fmt.Scanln(&dummy)
	return n
}

func readPassword() string {
	// Простое чтение пароля (на Windows пароль будет виден; для скрытия используй golang.org/x/term)
	var p string
	fmt.Scanln(&p)
	return p
}
