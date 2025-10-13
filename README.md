**Проект интернет магазина ( Go 1.25.1 )**


# 🧱 myApp — минималистичный веб-сервер на Go за NGINX

**myApp** — это учебный, но продакшен-готовый boilerplate веб-сервера на Go 1.25.1.
- Работает за NGINX (реверс-прокси), обеспечивая безопасность, производительность и масштабируемость.
- Поддерживает HTML-страницы, формы с валидацией, JSON-ответы и готов к расширению (API, аутентификация, БД).
- Архитектура — слоистая, близкая к Clean/Hexagonal, с фокусом на OWASP Top 10.


---

## 📁 Структура проекта

```
myApp/
├─ cmd/
│  └─ app/
│     └─ main.go    # Точка входа приложения: загружает конфиг, инициализирует логи и CSRF, 
│                    создаёт и запускает HTTP-сервер, ожидает сигнал завершения и выполняет graceful shutdown
├─ internal/
│  ├─ app/
│  │  ├─ app.go                     # Сборка: chi.Router, middleware, статика, маршруты, 404
│  ├─ core/
│  │  ├─ config.go                  # ENV-конфиг, проверки для prod
│  │  ├─ ctx.go                     # тип ключей контекста (чтобы избежать коллизий)
│  │  ├─ errors.go                  # AppError, фабрики (BadRequest, Internal)
│  │  ├─ response.go                # JSON(), Fail() (RFC7807)
│  │  └─ logfile.go                 # JSON-логи с ротацией (7 дней)
│  ├─ http/
│  │  ├─ handler/
│  │  │  ├─ home.go                 # / (HTML)
│  │  │  ├─ about.go                # /about (HTML)
│  │  │  ├─ form.go                 # /form (GET/POST, валидация, PRG)
│  │  │  └─ misc.go                 # /healthz (JSON), NotFound (404 HTML)
│  │  └─ middleware/
│  │     ├─ proxy.go                # TrustedProxy для NGINX (X-Forwarded-For, Proto)
│  │     └─ security.go             # CSP, XFO, nosniff, Referrer, Permissions, COOP, HSTS
│  └─ view/
│     └─ view.go                    # Централизованный рендер шаблонов
├─ web/
│  ├─ assets/                      # CSS/JS/изображения/шрифты
│  └─ templates/
│     ├─ layouts/base.gohtml       # Основной layout
│     ├─ partials/nav.gohtml       # Навигация
│     ├─ partials/footer.gohtml    # Футер
│     │ 
│     └─ pages/{home,about,form,404}.gohtml # Страницы
│ 
├─ logs/                           # DD-MM-YYYY.log, errors-DD-MM-YYYY.log
├─ nginx.conf                      # NGINX: TLS, rate limiting, кэш, сжатие
├─ go.mod
└─ go.sum
