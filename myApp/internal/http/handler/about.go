package handler

//about.go
import (
	"net/http"

	"myApp/internal/view"
)

// About возвращает обработчик для страницы "О нас" (OWASP A03: Injection)
func About(tpl *view.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Рендерит шаблон "about" с заголовком
		tpl.Render(w, r, "about", "О нас", nil)
	}
}
