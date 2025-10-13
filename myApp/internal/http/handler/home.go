package handler

//home.go
import (
	"net/http"

	"myApp/internal/view"
)

// Home возвращает обработчик для главной страницы (OWASP A03: Injection)
func Home(tpl *view.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Рендерит шаблон "home" с заголовком
		tpl.Render(w, r, "home", "Главная", nil)
	}
}
