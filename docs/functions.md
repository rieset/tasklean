# Tasklean — API Reference

Документация по текущим функциям и типам проекта.

---

## internal/config

### Types

#### Config
```go
type Config struct {
    TasksDirectory string
    DefaultRemote  string
    Editor         string
}
```
Глобальная конфигурация приложения (значения по умолчанию).

#### RemoteConfig
```go
type RemoteConfig struct {
    Name       string `json:"name"`
    URL        string `json:"url"`
    Token      string `json:"token"`
    Directory  string `json:"directory"`
    CreatedAt  string `json:"created_at"`
    LastPullAt string `json:"last_pull_at,omitempty"`
    LastPushAt string `json:"last_push_at,omitempty"`
}
```
Конфигурация удалённого трекера. Хранится в `~/.tasklean/<name>.conf`.

### Constants

| Name           | Value   |
|----------------|---------|
| DefaultTasksDir | `"./tasks"` |
| DefaultRemote   | `"origin"` |
| DefaultEditor   | `"vim"` |

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| DefaultConfig | `func DefaultConfig() *Config` | Возвращает конфиг с дефолтными значениями |
| Load | `func Load() (*Config, error)` | Создаёт `~/.tasklean` при необходимости, возвращает DefaultConfig |
| GetConfigDir | `func GetConfigDir() (string, error)` | Путь к `~/.tasklean` |
| GetTasksDir | `func GetTasksDir() (string, error)` | Директория задач из конфига |
| SaveRemoteConfig | `func SaveRemoteConfig(name, url, token, directory string) error` | Сохраняет remote в `~/.tasklean/<name>.conf` |
| LoadRemoteConfig | `func LoadRemoteConfig(name string) (*RemoteConfig, error)` | Загружает remote из `~/.tasklean/<name>.conf` |
| RemoveRemoteConfig | `func RemoveRemoteConfig(name string) error` | Удаляет `~/.tasklean/<name>.conf` |
| ListRemoteConfigs | `func ListRemoteConfigs() ([]RemoteConfig, error)` | Список всех `*.conf` в `~/.tasklean/` |
| RemoteConfigExists | `func RemoteConfigExists(name string) bool` | Проверяет наличие remote |
| UpdateRemoteLastPullAt | `func UpdateRemoteLastPullAt(name, timestamp string) error` | Обновляет LastPullAt в конфиге remote |

### Validation

Имя remote (`name`) проверяется во всех операциях:
- Не пустое
- Без `/`, `\`, `..` (защита от path traversal)

---

## internal/models

### Types

#### Task
```go
type Task struct {
    ID          string     `json:"id"`
    Title       string     `json:"title"`
    Description string     `json:"description,omitempty"`
    Status      TaskStatus `json:"status"`
    Remote      string     `json:"remote,omitempty"`
    CreatedAt   time.Time  `json:"created_at,omitempty"`
    UpdatedAt   time.Time  `json:"updated_at,omitempty"`
}
```

#### TaskStatus
```go
type TaskStatus string
```
Константы: `StatusBacklog`, `StatusTodo`, `StatusInProgress`, `StatusInReview`, `StatusDone`, `StatusCancelled`.

#### SyncStatus
```go
type SyncStatus struct {
    RemoteName    string        `json:"remote_name"`
    Direction     SyncDirection `json:"direction"`
    LastPullAt    *time.Time    `json:"last_pull_at,omitempty"`
    LastPushAt    *time.Time    `json:"last_push_at,omitempty"`
    TasksTotal    int           `json:"tasks_total"`
    TasksDone     int           `json:"tasks_done"`
    LocalChanges  int           `json:"local_changes"`
    RemoteChanges int           `json:"remote_changes"`
    Error         string        `json:"error,omitempty"`
}
```

#### SyncDirection
```go
type SyncDirection string
```
Константы: `SyncDirectionPull`, `SyncDirectionPush`, `SyncDirectionBoth`.

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| (TaskStatus) String | `func (s TaskStatus) String() string` | Строковое представление статуса |
| (TaskStatus) IsValid | `func (s TaskStatus) IsValid() bool` | Проверка допустимости статуса |
| NewSyncStatus | `func NewSyncStatus(remoteName string) *SyncStatus` | Создаёт SyncStatus с Direction=Both |
| (*SyncStatus) MarkPulled | `func (s *SyncStatus) MarkPulled()` | Устанавливает LastPullAt = now |
| (*SyncStatus) MarkPushed | `func (s *SyncStatus) MarkPushed()` | Устанавливает LastPushAt = now |

---

## internal/tui

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| PromptToken | `func PromptToken(prompt string) (string, error)` | TUI ввод токена (маскированный). Enter — подтвердить, Esc/Ctrl+C — отмена |
| PromptText | `func PromptText(prompt, placeholder string) (string, error)` | TUI ввод текста. Enter — подтвердить, Esc/Ctrl+C — отмена |
| Confirm | `func Confirm(prompt string) (bool, error)` | TUI подтверждение (y/n). y/Enter — true, n/Esc/Ctrl+C — false |
| ConfirmWithInput | `func ConfirmWithInput(prompt string, input io.Reader) (bool, error)` | То же, с кастомным вводом (для тестов). `input == nil` → TTY |

---

## internal/commands

### RootCommand

```go
type RootCommand struct {
    cmd *cobra.Command
    cfg *config.Config
}
```

| Method | Signature | Description |
|--------|-----------|-------------|
| NewRootCommand | `func NewRootCommand(cfg *config.Config, tr tracker.Tracker) *RootCommand` | Создаёт корневую команду. `tr == nil` → StubTracker |
| Execute | `func (rc *RootCommand) Execute() error` | Выполняет команду |
| SetArgs | `func (rc *RootCommand) SetArgs(args []string)` | Устанавливает аргументы (для тестов) |
| SilenceUsage | `func (rc *RootCommand) SilenceUsage()` | Отключает вывод usage при ошибках (для тестов) |

### CLI Commands

| Command | Use | Description |
|---------|-----|-------------|
| remote add | `add <name> <url>` | Добавить remote. Флаги: `-d`, `-t`, `-s` |
| remote remove | `remove <name>` | Удалить remote. Флаг: `-f` (skip confirmation) |
| pull | `pull <name>` | Загрузить задачи. Флаги: `-f`, `-a` |
| push | `push <name>` | Отправить задачи (stub) |
| status | `status` | Показать список remotes |

---

## internal/testutil

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| SetTestHome | `func SetTestHome(t *testing.T, dir string)` | Устанавливает `HOME` для тестов, восстанавливает в Cleanup |

---

## pkg/tracker

### Types

#### Tracker
```go
type Tracker interface {
    FetchTasks(ctx context.Context, cfg *config.RemoteConfig, opts *FetchOptions) ([]*models.Task, error)
}
```
Интерфейс для получения задач из удалённого трекера.

#### FetchOptions
```go
type FetchOptions struct {
    From     string // дата (YYYY-MM-DD), задачи обновлённые с этой даты
    Assignee string // email исполнителя
}
```
Опции фильтрации для FetchTasks.

### Functions / Types

| Name | Description |
|------|--------------|
| StubTracker | Реализация, возвращающая ErrNotImplemented |
| ErrNotImplemented | Ошибка «API не реализован» |

---

## cmd/tasklean

### main
```go
func main()
```
Точка входа: загружает конфиг, создаёт RootCommand, выполняет.
