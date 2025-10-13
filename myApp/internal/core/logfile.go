package core

//logfile.go
import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Logger управляет файлами логов с ротацией
type Logger struct {
	mainFile  *os.File   // Файл для основных логов
	errorFile *os.File   // Файл для ошибок
	mu        sync.Mutex // Синхронизация доступа к файлам
}

var globalLogger *Logger

// InitDailyLog инициализирует ротацию логов, создавая новые файлы на основе текущей даты (OWASP A09: Security Logging and Monitoring Failures)
func InitDailyLog() {
	// Создаёт директорию для логов, если не существует
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Printf("Ошибка создания директории логов: %v", err)
		return
	}

	dateStr := time.Now().Format("02-01-2006")
	mainPath := filepath.Join("logs", dateStr+".log")
	errPath := filepath.Join("logs", "errors-"+dateStr+".log")

	// Открывает файл для основных логов
	mainFile, err := os.OpenFile(mainPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Ошибка открытия основного лог-файла: %v", err)
		return
	}

	// Открывает файл для ошибок
	errorFile, err := os.OpenFile(errPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Ошибка открытия файла ошибок: %v", err)
		_ = mainFile.Close()
		return
	}

	globalLogger = &Logger{
		mainFile:  mainFile,
		errorFile: errorFile,
	}

	// Настраивает логгер для маршрутизации сообщений
	log.SetOutput(newLevelSplitWriter(mainFile, errorFile, os.Stdout))
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("=== Новый лог-файл инициализирован ===")

	// Запускает очистку старых логов
	go cleanupOldLogs("logs", 7)
}

// levelSplitWriter разделяет логи по уровням: ошибки в errors-*.log, остальное в основной лог, дублируя в stdout
type levelSplitWriter struct {
	mainW io.Writer  // Основной лог-файл
	errW  io.Writer  // Файл ошибок
	outW  io.Writer  // Вывод в stdout
	mu    sync.Mutex // Синхронизация записи
}

// newLevelSplitWriter создаёт новый levelSplitWriter с указанными писателями
func newLevelSplitWriter(mainW, errW, outW io.Writer) *levelSplitWriter {
	return &levelSplitWriter{mainW: mainW, errW: errW, outW: outW}
}

// Write записывает лог-сообщение, направляя его в соответствующий файл на основе уровня
func (w *levelSplitWriter) Write(p []byte) (int, error) {
	level := detectLevel(p)

	w.mu.Lock()
	defer w.mu.Unlock()

	// Дублирует вывод в stdout, если он доступен
	if w.outW != nil {
		_, _ = w.outW.Write(p)
	}

	// Направляет сообщение в соответствующий файл
	switch level {
	case "ERROR":
		if w.errW != nil {
			return w.errW.Write(p)
		}
	default:
		if w.mainW != nil {
			return w.mainW.Write(p)
		}
	}
	// Возвращает успешную запись, если файлы недоступны
	return len(p), nil
}

// detectLevel определяет уровень лога из сообщения (JSON или текст)
func detectLevel(p []byte) string {
	s := strings.TrimSpace(string(p))

	// Проверяет JSON-формат лога
	if strings.HasPrefix(s, "{") && strings.Contains(s, `"level"`) {
		var tmp struct {
			Level string `json:"level"`
		}
		if json.Unmarshal([]byte(s), &tmp) == nil && tmp.Level != "" {
			return strings.ToUpper(tmp.Level)
		}
	}

	// Проверяет текстовые индикаторы уровня
	ss := strings.ToUpper(s)
	if strings.Contains(ss, `"LEVEL":"ERROR"`) ||
		strings.Contains(ss, " ERROR ") ||
		strings.HasPrefix(ss, "ERROR ") ||
		strings.Contains(ss, "ERROR:") ||
		strings.HasPrefix(ss, "ERROR:") ||
		strings.Contains(ss, " LEVEL=ERROR") ||
		strings.Contains(ss, " LEVEL=ERR") {
		return "ERROR"
	}

	return "INFO"
}

// LogError записывает сообщение об ошибке в формате JSON (OWASP A09)
func LogError(msg string, fields map[string]interface{}) {
	if globalLogger == nil {
		log.Printf("ERROR: Логгер не инициализирован: %s, поля: %v", msg, fields)
		return
	}

	logEntry := map[string]interface{}{
		"level":  "ERROR",
		"msg":    msg,
		"time":   time.Now().UTC().Format(time.RFC3339),
		"fields": fields,
	}
	logData, err := json.Marshal(logEntry)
	if err != nil {
		log.Printf("ERROR: Ошибка сериализации лога: %v", err)
		return
	}

	log.Println(string(logData))
}

// cleanupOldLogs удаляет лог-файлы старше указанного количества дней (OWASP A09)
func cleanupOldLogs(dir string, maxAgeDays int) {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("ERROR: Ошибка чтения директории логов: %v", err)
		return
	}

	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		info, err := file.Info()
		if err != nil {
			log.Printf("ERROR: Ошибка получения информации о файле %s: %v", file.Name(), err)
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(dir, file.Name())
			if err := os.Remove(path); err != nil {
				log.Printf("ERROR: Ошибка удаления старого лога %s: %v", path, err)
			}
		}
	}
}

// Close закрывает файлы логов, обеспечивая корректное завершение (OWASP A09)
func Close() {
	if globalLogger != nil {
		globalLogger.mu.Lock()
		defer globalLogger.mu.Unlock()
		_ = globalLogger.mainFile.Close()
		_ = globalLogger.errorFile.Close()
	}
}
