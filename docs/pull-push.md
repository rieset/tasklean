# Pull и Push — логика синхронизации

Документация текущей логики pull и подсказки для реализации push.

---

## Текущая логика Pull

### Алгоритм

1. Загрузить конфиг remote из `~/.tasklean/<name>.conf`
2. Вызвать `Tracker.FetchTasks(ctx, remoteCfg, opts)` с опциями:
   - `From` — дата (YYYY-MM-DD), задачи обновлённые с этой даты
   - `Assignee` — email исполнителя для фильтрации
3. Для каждой задачи:
   - Установить `task.Remote = name`
   - Вызвать `storage.SaveTask(task, directory)`
4. Обновить `LastPullAt` в конфиге remote

### Структура хранения задач

Задачи **не сохраняются в корне** директории. Они распределяются по папкам assignee:

```
directory/
├── _index.json              # индекс: task_id → remote, module, created_at, updated_at
├── all/                     # задачи без assignee
│   ├── todo.md
│   ├── in_progress.md
│   ├── done.md
│   └── todo-{module}.md     # по модулю
├── user1-at-example.com/    # sanitized email assignee
│   └── todo.md
└── user2-at-test.com/
    └── in_progress.md
```

**Правила:**
- Задачи без assignee → `all/`
- Задачи с assignee → папка `{sanitized_email}/` (email → `user-at-example.com`)
- Одна задача может быть в нескольких папках (несколько assignees)
- Корень (`directory/`) не содержит файлов задач — только `_index.json`

### Формат файлов

**Имя файла:** `{status}.md` или `{status}-{module}.md`  
Примеры: `todo.md`, `in_progress.md`, `done.md`, `todo-backend-api.md`

**Структура блока задачи:**
```markdown
[📋 id:uuid-задачи]

## Заголовок задачи

Описание задачи...
```

**Эмодзи статуса** (в маркере `[emoji id:...]`) — при push изменение эмодзи меняет статус в трекере:
- 📥 backlog
- 📋 todo
- 🚀 in progress
- 👀 in review
- ✅ done
- ❌ cancelled

**Разделитель блоков:** `---`

### Индекс `_index.json`

```json
{
  "task-uuid": {
    "remote": "origin",
    "module": "Backend API",
    "created_at": "2026-02-17T10:49:19Z",
    "updated_at": "2026-02-17T10:49:31Z"
  }
}
```

- `remote` — имя remote, откуда задача
- `module` — модуль (если есть)
- `created_at`, `updated_at` — для сравнения с трекером

**Важно: `_index.json` управляется только кодом** — вручную редактировать не нужно и нельзя. Следующий вызов `SaveTask` или `pull` перезапишет его.

### Резолвинг модуля задачи

Модуль определяется в следующем порядке приоритета:

1. **Поле `module` в `_index.json`** — если есть, берётся оттуда
2. **Суффикс имени файла** — `todo-backup.md` → модуль `backup`; используется когда в индексе модуль не задан

**Вывод для ручного переноса задач:** чтобы переместить задачу в другой модуль, достаточно перенести её блок в нужный файл (`todo-backup.md`). Обновлять `_index.json` не требуется — модуль будет считан из имени файла.

### Функции storage

| Функция | Описание |
|---------|----------|
| `SaveTask(task, directory)` | Сохраняет задачу в папки assignee, обновляет индекс, удаляет из корня и старых папок |
| `LoadTask(taskID, directory)` | Ищет задачу в корне и во всех подпапках |
| `ListTasks(directory)` | Собирает задачи из корня и подпапок, дедуплицирует по ID |
| `DeleteTask(taskID, directory)` | Удаляет задачу из корня и всех подпапок, обновляет индекс |

---

## Подсказки для реализации Push

### Цель

Восстановить изменения из локальных файлов в task tracker: обновить title, description, status и учесть конфликты.

### Шаги реализации

#### 1. Сбор локальных задач

```go
tasks, err := storage.ListTasks(directory)
```

`ListTasks` возвращает все задачи из всех папок (all/, assignee/), уже дедуплицированные.

#### 2. Фильтрация по remote

Оставить только задачи, у которых `task.Remote == name` (имя push-remote).  
Индекс хранит remote в `_index.json` — при `ListTasks` remote берётся из индекса по `task.ID`.

**Важно:** `models.Task` не содержит поле `Remote` при загрузке из файлов — нужно смотреть индекс. `storage.ListTasks` возвращает задачи с `Remote` из `idxEntry.Remote`. Проверить в коде.

#### 3. Определение изменённых задач

Для каждой локальной задачи сравнить с данными трекера:

- **Вариант A:** Сделать `FetchTasks` для remote и сравнить по `UpdatedAt` из индекса с `UpdatedAt` из API. Если локальный `updated_at` новее — задача могла быть изменена локально (но это не гарантия — индекс обновляется при pull).
- **Вариант B:** Хеш содержимого — при pull сохранять хеш, при push сравнивать. Требует расширения индекса.
- **Вариант C:** Всегда пушить все задачи данного remote (простой вариант, возможны лишние API-вызовы).

**Рекомендация:** На первом этапе — пушить все задачи remote. Позже добавить `LastPullAt` — если задача в индексе с `updated_at` после `LastPullAt` на remote, значит она пришла с pull и не редактировалась локально. Задачи без `updated_at` в индексе или с `updated_at` до `LastPullAt` — кандидаты на push (осторожно: при первом push все задачи будут такими).

**Практичный подход:** Считать изменённой задачу, если её блок в .md файле отличается от того, что вернул бы FetchTasks. Для этого нужен fetch. Альтернатива: пушить все, трекер сам определит что менять (PATCH только переданные поля).

#### 4. Расширение Tracker интерфейса

```go
type Tracker interface {
    FetchTasks(ctx context.Context, cfg *config.RemoteConfig, opts *FetchOptions) ([]*models.Task, error)
    UpdateTask(ctx context.Context, cfg *config.RemoteConfig, task *models.Task) error  // для push
}
```

Или отдельный интерфейс `PushTracker` с `UpdateTask`.

#### 5. Plane API: обновление задачи

**Self-hosted (issues):**
```
PATCH /api/v1/workspaces/{workspace_slug}/projects/{project_id}/issues/{issue_id}/
```

**Cloud (work-items):**
```
PATCH /api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/
```

**Тело запроса (релевантные поля):**
```json
{
  "name": "Заголовок задачи",
  "description_html": "<p>Описание в HTML</p>",
  "state": "uuid-state-id"
}
```

**Важно:** `state` — это UUID состояния в Plane, не строка `todo`/`in_progress`. Нужно:
1. Получить список состояний проекта: `GET .../projects/{id}/states/`
2. Сопоставить `TaskStatus` → `state_id` по `state_group`:
   - `backlog` → state_group backlog
   - `todo`, `unstarted` → unstarted
   - `in_progress`, `started` → started
   - `in_review` → (если есть)
   - `done`, `completed` → completed
   - `cancelled` → cancelled

#### 6. Маппинг статуса обратно (TaskStatus → Plane state_id)

Обратная функция к `mapStateGroupToStatus` в `pkg/tracker/plane/plane.go`:

| TaskStatus   | state_group (Plane) |
|--------------|---------------------|
| backlog      | backlog             |
| todo         | unstarted           |
| in_progress  | started             |
| in_review    | started (или отдельный state) |
| done         | completed           |
| cancelled    | cancelled           |

Нужно хранить маппинг `state_group` → `state_id` (UUID) из ответа GET states.

#### 7. Описание: Markdown → HTML

Plane ожидает `description_html`. При pull используется `transform.TransformDescription` (HTML → Markdown). Для push нужна обратная конвертация: Markdown → HTML (`transform.MarkdownToHTML` через `gomarkdown/markdown`).

#### 8. Обработка конфликтов

Если задача изменена и на remote, и локально после last pull:

- Сравнить `updated_at` из индекса с `updated_at` из API.
- Если remote новее: спросить пользователя (overwrite / skip / merge) или вывести предупреждение.

#### 9. Обновление LastPushAt

После успешного push обновить `LastPushAt` в конфиге remote (аналогично `UpdateRemoteLastPullAt`).

#### 10. Новые задачи (создание)

Если в локальных файлах есть блок с `id:`, которого нет в трекере — это либо ошибка, либо задача создана вручную. Для создания нужен `POST .../issues/` с полями `name`, `description_html`, `state`, `project`. Отдельная фича.

---

## Self-hosted Plane API — особенности

Реальное поведение self-hosted Plane v1 (наблюдения из `task.h3llo.cloud`):

### Issues API

`GET /api/v1/workspaces/{slug}/projects/{id}/issues/`

- **Модулей нет в ответе** — поля `module`, `module_ids`, `module_details` отсутствуют в JSON issues. Связь issue↔module хранится в отдельном API.
- **`state`** — возвращается как UUID строка, поле `state__group` **не возвращается**. Для получения `state_group` нужно отдельно получить states: `GET .../states/`.
- **`assignees`** — возвращается как пустой массив `[]` даже при `expand=assignees`. Assignees приходят через поле `assignee_ids` (массив UUID членов проекта). Для резолва email → members: `GET .../members/`.
- **Пагинация** — offset/limit работает, но ответ содержит cursor-based поля: `next_cursor`, `prev_cursor`, `total_count`.

Пример минимального ответа issue:
```json
{
  "id": "b2ab1bfa-...",
  "name": "Заголовок задачи",
  "description_html": "<p></p>",
  "state": "9d31d494-...",
  "assignees": [],
  "priority": "none",
  "project": "3481d8a2-...",
  "workspace": "d1e4cc2a-...",
  "created_at": "2026-02-24T...",
  "updated_at": "2026-03-03T..."
}
```

### Modules API

`GET /api/v1/workspaces/{slug}/projects/{id}/modules/`

- Возвращает список модулей с `id` и `name`.
- Работает с offset/limit.

### Module-Issues API

`GET /api/v1/workspaces/{slug}/projects/{id}/modules/{module_id}/module-issues/`

- **Self-hosted возвращает полные объекты issues** (с `id`, `name`, etc.), а не `{issue: uuid}` как у cloud.
- Поле для извлечения issue ID: **`id`** (а не `issue`).
- Пагинация cursor-based: `next_cursor`, `prev_cursor`, `total_count`, `results`.

Пример ответа:
```json
{
  "total_count": 5,
  "next_cursor": "1000:1:0",
  "results": [
    {"id": "afd23d44-...", "name": "Бэкап инфраструктуры", ...},
    ...
  ]
}
```

### States API

`GET /api/v1/workspaces/{slug}/projects/{id}/states/`

Возвращает список состояний с полями `id`, `name`, `group`:

```json
[
  {"id": "9d31d494-...", "name": "Backlog", "group": "backlog"},
  {"id": "...", "name": "Todo", "group": "unstarted"},
  {"id": "...", "name": "In Progress", "group": "started"},
  {"id": "...", "name": "Done", "group": "completed"},
  {"id": "...", "name": "Cancelled", "group": "cancelled"}
]
```

### Маппинг state_group → TaskStatus

| state_group  | TaskStatus    |
|--------------|---------------|
| backlog      | backlog       |
| unstarted    | todo          |
| started      | in_progress   |
| completed    | done          |
| cancelled    | cancelled     |

---

## Чеклист для Push

- [x] Расширить `Tracker` интерфейс: `UpdateTask`, `CreateTask`
- [x] Реализовать `UpdateTask` в Plane: PATCH issues/work-items
- [x] Добавить получение states проекта для маппинга status → state_id
- [x] Реализовать `statusToStateID` — TaskStatus → UUID по state_group
- [x] Конвертация description: Markdown → HTML (`transform.MarkdownToHTML`)
- [x] Команда push: ListTasks → фильтр по remote → UpdateTask/CreateTask
- [x] Обновление `LastPushAt` после успеха
- [x] Создание новых задач (POST) — задачи с ID не из remote
- [x] `storage.ReplaceTaskID` — замена временного ID на реальный после create
