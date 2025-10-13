package app

//app.go
import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"myApp/internal/core"
	"myApp/internal/http/handler"
	mw "myApp/internal/http/middleware"
	"myApp/internal/view"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/csrf"
)

// New инициализирует HTTP-роутер с middleware и маршрутами приложения
func New(cfg core.Config, csrfKey []byte) (http.Handler, error) {
	// Инициализирует шаблонизатор для рендеринга HTML
	tpl, err := view.New()
	if err != nil {
		return nil, err
	}

	r := chi.NewRouter()

	// Добавляет nonce в контекст каждого запроса для Content Security Policy
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nonce, err := generateNonce()
			if err != nil {
				core.LogError("Ошибка генерации nonce", map[string]interface{}{"error": err.Error()})
				core.Fail(w, r, core.Internal("Ошибка генерации nonce", err))
				return
			}
			ctx := context.WithValue(r.Context(), core.CtxNonce, nonce)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	// Применяет middleware для обработки запросов
	r.Use(mw.TrustedProxy([]string{"127.0.0.1", "::1"})) // Ограничивает доверенные прокси
	r.Use(middleware.RequestID)                          // Добавляет уникальный ID запроса
	r.Use(middleware.RealIP)                             // Определяет реальный IP клиента
	r.Use(middleware.Logger)                             // Логирует запросы
	r.Use(middleware.Recoverer)                          // Восстанавливает после паники
	r.Use(middleware.Timeout(cfg.RequestTimeout))        // Ограничивает время выполнения запроса

	// Добавляет заголовки безопасности (CSP, X-Frame-Options и др.)
	r.Use(mw.SecureHeaders())

	// Включает HSTS для продакшен-среды, если HTTPS включён
	if cfg.Secure {
		r.Use(mw.HSTS(cfg.Env == "prod"))
	}

	// Настраивает CSRF-защиту для форм
	r.Use(csrf.Protect(
		csrfKey,
		csrf.Secure(cfg.Secure),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.HttpOnly(true),
		csrf.Path("/"),
	))
	// Логирует предупреждение, если CSRF используется без HTTPS в не-продакшен среде
	if !cfg.Secure && cfg.Env != "prod" {
		core.LogError("CSRF работает без HTTPS в не-продакшен среде", nil)
	}

	// Обслуживает статические файлы из директории web/assets
	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets"))))

	// Регистрирует маршруты приложения
	r.Get("/", handler.Home(tpl))
	r.Get("/about", handler.About(tpl))
	r.Get("/form", handler.FormIndex(tpl))
	r.Post("/form", handler.FormSubmit(tpl))
	r.Get("/healthz", handler.Health)
	r.NotFound(handler.NotFound(tpl))

	return r, nil
}

// generateNonce создаёт случайный nonce для Content Security Policy (OWASP A05)
func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
