package handler

//form.go
import (
	"log"
	"net/http"
	"strings"

	"myApp/internal/core"
	"myApp/internal/view"

	"github.com/go-playground/validator/v10"
	"github.com/microcosm-cc/bluemonday"
)

// FormData определяет структуру данных формы
type FormData struct {
	Name    string `validate:"required,min=2,max=100"` // Имя пользователя (обязательное, 2-100 символов)
	Email   string `validate:"required,email"`         // Email (обязательный, валидный формат)
	Message string `validate:"required,max=2000"`      // Сообщение (обязательное, до 2000 символов)
}

// FormView определяет данные для рендеринга формы
type FormView struct {
	Form   FormData          // Данные формы
	Errors map[string]string // Ошибки валидации
	OK     bool              // Флаг успешной отправки формы
}

var (
	validate  = validator.New()        // Валидатор для проверки полей формы
	sanitizer = bluemonday.UGCPolicy() // Политика санитизации ввода
)

// FormIndex возвращает обработчик для отображения формы (OWASP A03: Injection)
func FormIndex(tpl *view.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверяет параметр успешной отправки формы
		ok := r.URL.Query().Get("ok") == "1"
		data := FormView{
			Form:   FormData{},
			Errors: map[string]string{},
			OK:     ok,
		}
		// Рендерит шаблон формы
		tpl.Render(w, r, "form", "Форма", data)
	}
}

// FormSubmit возвращает обработчик для обработки отправки формы (OWASP A03, A05)
func FormSubmit(tpl *view.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверяет метод запроса
		if r.Method != http.MethodPost {
			http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
			return
		}

		// Ограничивает размер тела запроса до 1MB
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Некорректный запрос", http.StatusBadRequest)
			return
		}

		// Санитизирует данные формы
		f := FormData{
			Name:    sanitizer.Sanitize(strings.TrimSpace(r.Form.Get("name"))),
			Email:   sanitizer.Sanitize(strings.TrimSpace(r.Form.Get("email"))),
			Message: sanitizer.Sanitize(strings.TrimSpace(r.Form.Get("message"))),
		}

		// Проверяет валидацию формы
		errs := map[string]string{}
		if err := validate.Struct(f); err != nil {
			if verrs, ok := err.(validator.ValidationErrors); ok {
				for _, e := range verrs {
					switch e.Field() {
					case "Name":
						switch e.Tag() {
						case "required":
							errs["name"] = "Укажите имя"
						case "min":
							errs["name"] = "Имя должно быть не короче 2 символов"
						case "max":
							errs["name"] = "Слишком длинное имя (макс. 100)"
						default:
							errs["name"] = "Некорректное имя"
						}
					case "Email":
						switch e.Tag() {
						case "required":
							errs["email"] = "Укажите email"
						case "email":
							errs["email"] = "Введите корректный email"
						default:
							errs["email"] = "Некорректный email"
						}
					case "Message":
						switch e.Tag() {
						case "required":
							errs["message"] = "Напишите сообщение"
						case "max":
							errs["message"] = "Слишком длинное сообщение (макс. 2000)"
						default:
							errs["message"] = "Некорректное сообщение"
						}
					}
				}
				// Логирует ошибки валидации как INFO
				log.Printf("INFO: Ошибка валидации формы: %v", errs)
			} else {
				errs["form"] = "Ошибка валидации"
				core.LogError("Неожиданная ошибка валидации", map[string]interface{}{"error": err.Error()})
			}
		}

		// Если есть ошибки, отображает форму с ошибками
		if len(errs) > 0 {
			data := FormView{
				Form:   f,
				Errors: errs,
				OK:     false,
			}
			tpl.Render(w, r, "form", "Форма", data)
			return
		}

		// Перенаправляет на форму с флагом успеха (PRG-паттерн)
		http.Redirect(w, r, "/form?ok=1", http.StatusSeeOther)
	}
}
