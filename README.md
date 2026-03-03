# Tasklean

CLI для работы с задачами из трекеров как с локальными файлами. Превращает Plane (и другие трекеры) в file-based workflow.

## Что это

Tasklean синхронизирует задачи между трекером (Plane, Jira и др.) и локальными Markdown-файлами. Каждая задача — блок в `.md` файле. Редактируете в любимом редакторе, пушите изменения обратно в трекер.

## Преимущества

- **Работа как с кодом** — задачи в Git, diff, merge, привычный workflow
- **Любой редактор** — VS Code, vim, Obsidian, что угодно
- **Офлайн** — можно править локально, синхронизировать позже
- **Прозрачность** — всё в файлах, без чёрного ящика
- **Конфликты под контролем** — при расхождении с UI задача помечается, локальные изменения сохраняются в блоке resolve

## Установка

Ссылки ведут на последнюю версию (менять не нужно):

| Платформа | Ссылка |
|-----------|--------|
| Linux amd64 | [tasklean_linux_amd64.tar.gz](https://github.com/rie/tasklean/releases/latest/download/tasklean_linux_amd64.tar.gz) |
| Linux arm64 | [tasklean_linux_arm64.tar.gz](https://github.com/rie/tasklean/releases/latest/download/tasklean_linux_arm64.tar.gz) |
| macOS amd64 | [tasklean_darwin_amd64.tar.gz](https://github.com/rie/tasklean/releases/latest/download/tasklean_darwin_amd64.tar.gz) |
| macOS arm64 (Apple Silicon) | [tasklean_darwin_arm64.tar.gz](https://github.com/rie/tasklean/releases/latest/download/tasklean_darwin_arm64.tar.gz) |
| Windows amd64 | [tasklean_windows_amd64.zip](https://github.com/rie/tasklean/releases/latest/download/tasklean_windows_amd64.zip) |

```bash
# Распаковать и положить в PATH (пример для Linux)
tar xzf tasklean_linux_amd64.tar.gz
sudo mv tasklean /usr/local/bin/
```

Или через Go: `go install github.com/rie/tasklean/cmd/tasklean@latest`

## Использование

```bash
# Добавить remote (TUI запросит токен)
tasklean remote add origin https://plane.example.com

# Статус remotes
tasklean status

# Подтянуть задачи с трекера
tasklean pull origin

# Отправить изменения в трекер
tasklean push origin

# Список задач с неразрешёнными конфликтами (пропущенных при push)
tasklean resolve origin

# Удалить remote
tasklean remote remove origin
```

## Документация

| Документ | Описание |
|----------|----------|
| [docs/pull-push.md](docs/pull-push.md) | Логика pull/push, формат файлов, структура директорий |
| [docs/functions.md](docs/functions.md) | Справочник по функциям и типам API |
| [docs/specification.md](docs/specification.md) | Техническая спецификация |
| [docs/resolve-spec.md](docs/resolve-spec.md) | Подсекция resolve для конфликтов |
| [docs/development.md](docs/development.md) | Установка, структура проекта, разработка |

## Лицензия

Бесплатно для личного (некоммерческого) использования.
Коммерческое использование запрещено без отдельного соглашения.
Подробности — в [LICENSE](LICENSE).
