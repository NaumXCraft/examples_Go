package view

//view.go
import (
	"html/template"
	"net/http"

	"myApp/internal/core"

	"github.com/gorilla/csrf"
)

// Templates хранит предварительно загруженные HTML-шаблоны
type Templates struct {
	templates map[string]*template.Template // Карта шаблонов по именам
}

// PageData определяет данные для рендеринга шаблонов (OWASP A03: Injection, A07: Identification and Authentication Failures)
type PageData struct {
	Title     string        // Заголовок страницы
	CSRFField template.HTML // Поле CSRF-токена
	Nonce     string        // Nonce для Content Security Policy
	Data      interface{}   // Дополнительные данные для шаблона
}

// New инициализирует шаблоны из файлов (OWASP A05: Security Misconfiguration)
func New() (*Templates, error) {
	layouts := []string{
		"web/templates/layouts/base.gohtml",    // Основной шаблон
		"web/templates/partials/nav.gohtml",    // Частичный шаблон навигации
		"web/templates/partials/footer.gohtml", // Частичный шаблон футера
	}
	pages := map[string][]string{
		"home":     {"web/templates/pages/home.gohtml"},
		"about":    {"web/templates/pages/about.gohtml"},
		"form":     {"web/templates/pages/form.gohtml"},
		"notfound": {"web/templates/pages/404.gohtml"},
	}

	t := &Templates{templates: make(map[string]*template.Template)}
	for name, pageFiles := range pages {
		// Комбинирует общие и страничные шаблоны
		files := append(layouts, pageFiles...)
		tpl, err := template.ParseFiles(files...)
		if err != nil {
			return nil, err
		}
		t.templates[name] = tpl
	}
	return t, nil
}

// Render рендерит шаблон с данными, включая CSRF-токен и nonce (OWASP A03, A05)
func (t *Templates) Render(w http.ResponseWriter, r *http.Request, templateName string, title string, data interface{}) {
	tpl, ok := t.templates[templateName]
	if !ok {
		core.Fail(w, r, core.Internal("Шаблон не найден: "+templateName, nil))
		return
	}

	// Получает nonce из контекста запроса
	nonce, _ := r.Context().Value(core.CtxNonce).(string)
	if nonce == "" {
		core.Fail(w, r, core.Internal("Nonce не найден в контексте", nil))
		return
	}

	// Устанавливает заголовок Content-Type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Рендерит шаблон с данными
	if err := tpl.ExecuteTemplate(w, "base", PageData{
		Title:     title,
		CSRFField: csrf.TemplateField(r),
		Nonce:     nonce,
		Data:      data,
	}); err != nil {
		core.LogError("Ошибка рендеринга шаблона", map[string]interface{}{
			"template": templateName,
			"error":    err.Error(),
		})
		core.Fail(w, r, core.Internal("Ошибка шаблона", err))
	}
}
