package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var uploadDir = "./uploads"

// formatSize преобразует размер файла в байтах в удобочитаемый формат (B, KB, MB, GB).
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

var funcMap = template.FuncMap{
	"split":      strings.Split,
	"join":       strings.Join,
	"add":        func(a, b int) int { return a + b },
	"slice":      func(arr []string, start, end int) []string { return arr[start:end] },
	"div":        func(a int64, b float64) float64 { return float64(a) / b },
	"formatSize": formatSize,
}

var tmpl *template.Template

func init() {
	tmpl = template.New("index.gohtml").Funcs(funcMap)
	tmpl = template.Must(tmpl.ParseFiles("static/index.gohtml"))
}

type File struct {
	Name          string
	IsDir         bool
	Size          int64
	FormattedSize string
	URL           string
	DeleteURL     string
}

type PageData struct {
	CurrentPath string
	ParentPath  string
	Items       []File
}

func main() {
	log.Println("Инициализация: Создание директории для загрузки")
	os.MkdirAll(uploadDir, os.ModePerm)

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/mkdir", mkdirHandler)
	http.HandleFunc("/delete/", deleteHandler)
	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(uploadDir))))

	log.Println("Сервер запущен на: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	rawPath := strings.TrimPrefix(r.URL.Path, "/")
	cleanPath := path.Clean(rawPath)
	if cleanPath == "." {
		cleanPath = ""
	}

	osPath := filepath.FromSlash(cleanPath)
	fullPath := filepath.Join(uploadDir, osPath)

	rel, err := filepath.Rel(uploadDir, fullPath)
	if err != nil || strings.Contains(rel, "..") {
		http.Error(w, "Доступ запрещён: Недопустимый путь", http.StatusForbidden)
		return
	}

	stat, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
		} else {
			log.Printf("Ошибка при os.Stat(%s): %v", fullPath, err)
			http.Error(w, "Ошибка сервера при чтении файла/папки", http.StatusInternalServerError)
		}
		return
	}

	if !stat.IsDir() {
		fileURL := "/files/" + cleanPath
		http.Redirect(w, r, fileURL, http.StatusTemporaryRedirect)
		return
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		log.Printf("Ошибка при os.ReadDir(%s): %v", fullPath, err)
		http.Error(w, "Не могу прочитать папку", http.StatusInternalServerError)
		return
	}

	var items []File
	for _, entry := range entries {
		info, err := entry.Info()

		size := int64(0)
		formattedSize := "N/A"

		if err == nil {
			size = info.Size()
			formattedSize = formatSize(size)
		} else {
			log.Printf("Ошибка чтения info для %s: %v", entry.Name(), err)
		}

		name := entry.Name()

		item := File{
			Name:          name,
			IsDir:         entry.IsDir(),
			Size:          size,
			FormattedSize: formattedSize,
		}

		if entry.IsDir() {
			item.URL = "/" + path.Join(cleanPath, name)
		} else {
			item.URL = "/files/" + path.Join(cleanPath, name)
		}

		item.DeleteURL = "/delete/" + path.Join(cleanPath, name)

		items = append(items, item)
	}

	parent := path.Dir(cleanPath)
	if parent == "." || parent == "/" {
		parent = ""
	}

	data := PageData{
		CurrentPath: cleanPath,
		ParentPath:  parent,
		Items:       items,
	}

	err = tmpl.ExecuteTemplate(w, "index.gohtml", data)
	if err != nil {
		log.Println("Template execute error:", err)
		http.Error(w, "Ошибка шаблона: "+err.Error(), http.StatusInternalServerError)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(100 << 20) // 100 МБ

	dir := r.FormValue("dir")
	osDir := filepath.FromSlash(dir)
	fullDir := filepath.Join(uploadDir, osDir)

	rel, err := filepath.Rel(uploadDir, fullDir)
	if err != nil || strings.Contains(rel, "..") {
		http.Error(w, "Недопустимый путь", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Файл не выбран или ошибка формы: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	dstPath := filepath.Join(fullDir, header.Filename)
	if _, err := os.Stat(dstPath); err == nil {
		log.Printf("Файл %s уже существует. Перезапись.", dstPath)
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		log.Printf("Ошибка при os.Create(%s): %v", dstPath, err)
		http.Error(w, "Не удалось создать файл на сервере", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		log.Printf("Ошибка копирования в файл %s: %v", dstPath, err)
		http.Error(w, "Ошибка копирования файла", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/"+dir, http.StatusSeeOther)
}

func mkdirHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		return
	}

	dir := r.FormValue("dir")
	name := r.FormValue("name")

	if name == "" || strings.ContainsAny(name, "/\\:") || strings.Contains(name, "..") {
		http.Error(w, "Недопустимое имя папки (запрещены /, \\, :, ..)", http.StatusBadRequest)
		return
	}

	osDir := filepath.FromSlash(dir)
	fullPath := filepath.Join(uploadDir, osDir, name)

	rel, err := filepath.Rel(uploadDir, fullPath)
	if err != nil || strings.Contains(rel, "..") {
		http.Error(w, "Недопустимый путь", http.StatusBadRequest)
		return
	}

	if err := os.Mkdir(fullPath, os.ModePerm); err != nil {
		log.Printf("Ошибка при os.Mkdir(%s): %v", fullPath, err)
		http.Error(w, "Не удалось создать папку (возможно, уже существует)", http.StatusInternalServerError)
		return
	}

	newPath := path.Join(dir, name)
	http.Redirect(w, r, "/"+newPath, http.StatusSeeOther)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		return
	}

	rawPath := strings.TrimPrefix(r.URL.Path, "/delete/")
	cleanPath := path.Clean(rawPath)
	if strings.HasPrefix(cleanPath, "/") {
		cleanPath = strings.TrimPrefix(cleanPath, "/")
	}

	if cleanPath == "" || cleanPath == "." {
		http.Error(w, "Нельзя удалить корневую папку", http.StatusForbidden)
		return
	}

	osPath := filepath.FromSlash(cleanPath)
	fullPath := filepath.Join(uploadDir, osPath)

	rel, err := filepath.Rel(uploadDir, fullPath)
	if err != nil || strings.Contains(rel, "..") {
		http.Error(w, "Доступ запрещён: Недопустимый путь", http.StatusForbidden)
		return
	}

	if err := os.RemoveAll(fullPath); err != nil {
		log.Printf("Ошибка при os.RemoveAll(%s): %v", fullPath, err)
		http.Error(w, "Не удалось удалить", http.StatusInternalServerError)
		return
	}

	parent := path.Dir(cleanPath)
	if parent == "." || parent == "/" {
		parent = ""
	}
	http.Redirect(w, r, "/"+parent, http.StatusSeeOther)
}
