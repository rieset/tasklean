# Tasklean — Установка и разработка

Технические аспекты: установка, структура проекта, конфигурация, запуск тестов.

---

## Установка

```bash
# Из корня проекта — установка в $GOPATH/bin (или $HOME/go/bin)
go install ./cmd/tasklean

# Добавить Go bin в PATH (в ~/.zshrc или ~/.bashrc)
export PATH="$PATH:$(go env GOPATH)/bin"

# Проверка
tasklean --help
```

---

## Структура проекта

```
tasklean/
├── cmd/tasklean/     # Точка входа CLI
├── internal/         # Приватный код приложения
│   ├── commands/     # Команды (pull, push, resolve, remote, status)
│   ├── config/       # Конфигурация
│   ├── models/       # Модели данных
│   ├── storage/      # Хранение задач в .md файлах
│   ├── testutil/     # Утилиты для тестов
│   ├── transform/    # Преобразование Markdown ↔ HTML
│   └── tui/          # Интерактивные компоненты (ввод токена, подтверждение)
├── pkg/tracker/      # Адаптеры трекеров (Plane, Stub)
├── configs/          # Файлы конфигурации
└── docs/             # Документация
```

---

## Конфигурация

Конфигурация remotes хранится в `~/.tasklean/`:

- `~/.tasklean/<name>.conf` — один файл на remote (URL, token, directory)
- Конфиг создаётся при `tasklean remote add`

Для Plane дополнительно указываются workspace и project (через TUI или флаги).

---

## Настройка после клонирования

```bash
# Подключить git-хуки репозитория (убирает AI co-author из коммитов)
git config core.hooksPath .githooks
```

---

## Разработка

### Требования

- Go 1.21+
- golangci-lint (опционально)

### Тесты

```bash
go test ./...
```

### Линтинг

```bash
gofmt -w .
goimports -w .
golangci-lint run ./...
```
