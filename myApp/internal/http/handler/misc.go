package handler

//misc.go
import (
	"net/http"

	"myApp/internal/core"
	"myApp/internal/view"
)

// Health возвращает обработчик для проверки состояния сервера (OWASP A09)
func Health(w http.ResponseWriter, r *http.Request) {
	// Отправляет JSON-ответ с статусом "ok"
	core.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// NotFound возвращает обработчик для страницы 404 (OWASP A03)
func NotFound(tpl *view.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Устанавливает статус 404 и рендерит шаблон "notfound"
		w.WriteHeader(http.StatusNotFound)
		tpl.Render(w, r, "notfound", "Страница не найдена", nil)
	}
}
