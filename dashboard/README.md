# Docker Container Dashboard

Простой дашборд для мониторинга Docker контейнеров с использованием Clean Architecture и инлайн стилей на базе Tailwind CSS классов.

## Архитектура

Проект реализован по принципам Clean Architecture:

```
dashboard/
├── domain/           # Бизнес-логика (Entities, Use Cases)
│   ├── entities.go   # Доменные объекты
│   └── usecases.go   # Бизнес-правила
├── infrastructure/   # Внешние интерфейсы
│   └── repository.go # Доступ к данным
├── interfaces/       # Адаптеры интерфейсов
│   └── handlers.go   # HTTP обработчики
├── pkg/twsx/         # Утилиты для стилей
│   └── twsx.go       # Конвертер Tailwind классов
├── config/           # Конфигурация
│   └── config.go     # Настройки приложения
├── data/             # Статические данные
│   └── dashboard.json # JSON данные дашборда
├── main.go           # Точка входа
└── go.mod            # Go модули
```

## Особенности

- ✅ **Clean Architecture**: Разделение ответственности по слоям
- ✅ **Семантические CSS классы**: Чистый HTML5 с объединенными стилями в head
- ✅ **Минифицированные стили**: CSS (3407 chars) и JavaScript (279 chars) без пробелов
- ✅ **Автоматический выбор порта**: Умное определение свободного порта (дефолт 3000)
- ✅ **Graceful shutdown**: Правильное завершение работы по сигналам ОС
- ✅ **Tailwind-like синтаксис**: Использование знакомых классов без внешних файлов
- ✅ **JSON данные**: Шаблоны полностью отделены от данных
- ✅ **REST API**: Полноценное API для работы с данными
- ✅ **Автоматическая генерация CSS**: Стили собираются в head секции
- ✅ **Стандартная библиотека**: Минимум зависимостей

## Запуск

```bash
# Сборка
go build

# Запуск
./dashboard

# Или сразу
go run main.go
```

Сервер запустится на `http://localhost:3000` (или автоматически выберет свободный порт)

## API Endpoints

### Web интерфейс
- `GET /` - Главная страница дашборда

### API
- `GET /api/dashboard` - Получить все данные дашборда
- `PUT /api/containers/{id}` - Обновить статус контейнера

### Пример API запроса

```bash
# Получить данные
curl http://localhost:8080/api/dashboard

# Обновить статус контейнера
curl -X PUT http://localhost:8080/api/containers/web-app-01 \
  -H "Content-Type: application/json" \
  -d '{"status": "stopped"}'
```

## Структура данных

Данные хранятся в `data/dashboard.json`:

```json
{
  "title": "Docker Container Dashboard",
  "subtitle": "Real-time container monitoring",
  "stats": {
    "total_containers": 12,
    "running_containers": 8,
    "stopped_containers": 3,
    "paused_containers": 1
  },
  "containers": [
    {
      "id": "web-app-01",
      "name": "nginx-web",
      "image": "nginx:alpine",
      "status": "running",
      "ports": ["80:80", "443:443"],
      "cpu_usage": 15.2,
      "memory_usage": 45.8
    }
  ],
  "system": {
    "cpu_cores": 4,
    "total_memory": 8192,
    "used_memory": 3456,
    "disk_usage": 2341
  }
}
```

## Система семантических стилей

Проект использует усовершенствованную систему стилей `twsx` с семантическими классами:

### Регистрация стилей

```go
// Создание реестра стилей
registry := twsx.NewStyleRegistry()

// Регистрация семантических классов
registry.CLASS("card", twsx.TWSX("bg-white rounded-lg shadow p-6"))
registry.CLASS("btn-primary", twsx.TWSX("bg-blue-600 text-white px-4 py-2 rounded"))
```

### Генерация CSS

```go
// Получение минифицированных CSS правил (без пробелов и переносов)
css := registry.GenerateCSS()
// Результат: ".body{background-color:#f9fafb;font-family:system-ui, sans-serif;min-height:100vh}..."
```

### Использование в HTML

```html
<div class="card">
    <button class="btn-primary">Click me</button>
</div>
```

### Преимущества подхода

- **Чистый HTML**: Семантические классы вместо инлайн стилей
- **Объединенные стили**: Все CSS в head секции, без дублирования
- **Кеширование**: Стили генерируются один раз при рендере страницы
- **Tailwind синтаксис**: Знакомые классы для быстрой разработки

## Конфигурация

Приложение использует переменные окружения:

- `SERVER_HOST` - Хост сервера (по умолчанию: localhost)
- `SERVER_PORT` - Порт сервера (по умолчанию: 8080)
- `DASHBOARD_DATA_FILE` - Путь к JSON файлу данных (по умолчанию: data/dashboard.json)

```bash
# Изменить порт (дефолтный 3000 с умным авто-выбором)
SERVER_PORT=8080 ./dashboard

# Примеры вывода логов:
# 🚀 Starting Dashboard Server...
# 📡 Server URL: http://localhost:3000
# 📊 Data source: data/dashboard.json
# ⚠️  Port 3000 failed, trying alternative port...
# ✅ Found free port: 3001 ✓ Using alternative port
# 💡 Server ready! Press Ctrl+C to stop gracefully

# Использовать кастомный файл данных
DASHBOARD_DATA_FILE=custom/data.json ./dashboard

# Комбинированные настройки
SERVER_HOST=0.0.0.0 SERVER_PORT=9090 DASHBOARD_DATA_FILE=prod/data.json ./dashboard

# Graceful shutdown работает автоматически по Ctrl+C
# Примеры логов завершения:
# 🛑 Received shutdown signal...
# ⏳ Gracefully shutting down server (5s timeout)
# ✅ Server shutdown completed successfully
# 👋 Goodbye!
```

## Разработка

### Добавление новых стилей

1. Добавьте Tailwind классы в `pkg/twsx/twsx.go`
2. Используйте в шаблонах через `twsx.TWSX()` и `twsx.StylesToInlineCSS()`

### Добавление новых API endpoints

1. Добавьте метод в `domain/usecases.go`
2. Реализуйте обработчик в `interfaces/handlers.go`
3. Зарегистрируйте маршрут в `main.go`

### Добавление новых данных

Обновите структуру в `domain/entities.go` и JSON файл в `data/dashboard.json`.

## Производительность

- Используется стандартная библиотека Go (net/http)
- Инлайн стили исключают дополнительные HTTP запросы
- JSON данные кешируются в памяти
- Clean Architecture обеспечивает легкость тестирования и модификации
