# Tasklean CLI — Technical Specification

## Implementation Status (Summary)

| Component | Status | Notes |
|-----------|--------|-------|
| Project setup | ✅ | go.mod, cobra, bubbletea, lipgloss, bubbles |
| Config | ✅ | Only `~/.tasklean/<name>.conf`, no global config |
| remote add | ✅ | TUI token input, --directory flag |
| remote remove | ✅ | TUI confirmation |
| status | ✅ | Lists remotes with name, URL, directory |
| pull | ✅ | Loads config, fetches via Tracker, saves .md files |
| push | ✅ | UpdateTask, CreateTask, ReplaceTaskID, LastPushAt |
| Basic models | ✅ | Task, SyncStatus, RemoteConfig |
| Task storage | ✅ | internal/storage |
| Tracker API | ✅ | pkg/tracker: Plane (workspace+project), Stub fallback |

---

## Implementation Stages

### Stage 1: Project Setup (Can be tested: ✅)
- Initialize Go module
- Add dependencies: bubbletea, lipgloss, bubbles (textinput), cobra
- Create basic project structure: `cmd/`, `internal/commands/`, `internal/config/`, `internal/tui/`
- Verify: `go build` completes without errors

### Stage 2: Configuration Management (Can be tested: ✅)
- Create `internal/config` package
- Implement remote config (`.conf` files) management only — no global config file
- Create `~/.tasklean/` directory if not exists
- Config path: `~/.tasklean/<name>.conf` — one file per remote
- Verify: config files created/read correctly

### Stage 3: Basic Models (Can be tested: ✅)
- Create `internal/models` package
- Define `Task` struct with fields (ID, Title, Description, Status, Remote, CreatedAt, UpdatedAt)
- Define `RemoteConfig` struct (in `internal/config/remote.go`)
- Define `SyncStatus` struct (RemoteName, Direction, LastPullAt, LastPushAt, TasksTotal, etc.)
- Verify: models serialize/deserialize correctly (Task, SyncStatus JSON tests)

### Stage 4: Remote Add Command (Can be tested: ✅)
- Implement `remote add <name> <url>` command
- Interactive token prompt (Bubble Tea TUI, password-style masking)
- Flag: `-d, --directory` — directory for tasks (default: current)
- Validate URL format (planned)
- Test connection to remote (planned)
- Save config to `~/.tasklean/<name>.conf`
- Verify: config file created with token

### Stage 5: Remote Remove Command (Can be tested: ✅)
- Implement `remote remove <name>` command
- Confirmation dialog (Bubble Tea TUI, y/n)
- Remove config file
- Verify: config file deleted

### Stage 6: Status Command (Can be tested: ✅)
- Implement `status` command
- Read all remote configs from `~/.tasklean/*.conf`
- Display: name, URL, directory, status, last pull/push
- Verify: correct statistics shown

### Stage 7: Task File Operations (Can be tested: ✅)
- Create `internal/storage` package
- Implement task file read/write with frontmatter
- Implement task file parsing
- List tasks from directory
- Verify: files created and parsed correctly

### Stage 8: Pull Command (Can be tested: ✅)
- Implement `pull <name>` command
- Load remote config from `~/.tasklean/<name>.conf`
- Fetch tasks via `pkg/tracker.Tracker` (Plane when workspace+project set, else Stub)
- Save each task as `{id}.md` with frontmatter
- Update `LastPullAt` in config after success
- Verify: tasks downloaded as `.md` files

### Stage 9: Push Command (Partial: ⏳)
- Implement `push <name>` command
- Load remote config
- Scan local task files (planned)
- Parse each file (planned)
- Detect changes (planned)
- Make API requests (planned)
- Handle conflict resolution (planned)
- Verify: remote updated with local changes

### Stage 10: Integration & Polish (Planned: ⏳)
- Add error handling throughout
- Add progress bars for pull/push
- Add help text and usage examples
- Verify: all commands work end-to-end

---

## Overview

Tasklean is a CLI tool for managing tasks from task trackers as local files. Built with Bubble Tea framework for interactive TUI experience.

**CLI framework:** Cobra (command structure)  
**TUI framework:** Bubble Tea + Lipgloss + Bubbles (textinput)

## Commands

### remote add <name> <url>

Adds a new remote tracker connection.

**Behavior:**
1. Prompt user for API token (secure input)
2. Validate connection by making test request to `<url>`
3. Save configuration to `~/.tasklean/<name>.conf`

**Flags:**
- `-d, --directory` — directory to store task files (default: current directory)

**Configuration file format** (`~/.tasklean/<name>.conf`):
```json
{
  "name": "<name>",
  "url": "<url>",
  "token": "<token>",
  "directory": "<path>",
  "created_at": "2024-01-01T00:00:00Z",
  "last_pull_at": "",
  "last_push_at": ""
}
```

**TUI:** Bubble Tea prompt with masked input (•). Enter to submit, Ctrl+C/Esc to cancel.

**Error cases:**
- Invalid URL format
- Connection refused / timeout
- Invalid token
- Remote name already exists

---

### remote remove <name>

Removes a saved remote configuration.

**Behavior:**
1. Find config file `~/.tasklean/<name>.conf`
2. Confirm deletion (Bubble Tea TUI: y/n)
3. Remove config file

**Flags:** `-f, --force` — skip confirmation (for scripts/CI)

**TUI:** Confirmation dialog. y/Enter = confirm, n/Esc/Ctrl+C = cancel.

**Error cases:**
- Config file not found
- Permission denied

---

### pull <name>

Fetches tasks from remote tracker and saves as local files.

**Behavior:**
1. Load config from `~/.tasklean/<name>.conf` — returns error if not found
2. Fetch tasks from tracker API (via `Tracker.FetchTasks`)
3. Save each task as `{id}.md` with frontmatter
4. Update `LastPullAt` in config

**Flags:**
- `-f, --from` — filter tasks updated from date (YYYY-MM-DD)
- `-a, --assignee` — filter by assignee email

**Output file format:** см. [docs/pull-push.md](pull-push.md) — полное описание логики pull, структуры хранения и подсказок для push.

Кратко:
- Задачи сохраняются **только в папках по assignee**: `all/` (без assignee), `{email}/` (с assignee)
- Файлы: `{status}.md`, `{status}-{module}.md` (todo.md, in_progress.md, done.md)
- `_index.json` — индекс: id → remote, module, created_at, updated_at

Структура блока в `todo.md`:
```markdown
[📋 id:b2ab1bfa-95cb-46f9-9bad-22e0b984c80d]

## Task title

Task description...

---

[✅ id:52de03ca-0490-49fa-98af-c023b6f9291b]

## Another task (done)

...
```

**Эмодзи статуса** (перед id): при push изменение эмодзи меняет статус задачи.
- 📥 backlog
- 📋 todo
- 🚀 in progress
- 👀 in review
- ✅ done
- ❌ cancelled

**Error cases:**
- Config not found
- Network error
- API error (unauthorized, rate limited)
- File write permission denied

---

### push <name>

Pushes local task files to remote tracker.

**Behavior:**
1. Load config from `~/.tasklean/<name>.conf` — returns error if not found
2. Scan local tasks directory (planned)
3. Parse each `.md` file (planned)
4. Sync changes to tracker API (planned)
5. Show progress in TUI (planned)

**Current:** Loads config, prints `Pushing tasks to <name> (<url>)`

**Conflict resolution (planned):**
- If remote task was modified after last pull → show warning
- Allow user to choose: overwrite / skip / merge

**Error cases:**
- Config not found
- Network error
- API error
- Invalid file format

---

### status

Shows current synchronization status.

**Behavior:**
1. Read all `*.conf` files from `~/.tasklean/`
2. Display each remote with name, URL, directory, status
3. Check if task directory exists

**Output (current):**
```
Configured remotes (1):

  Name:      origin
  URL:       https://task.h3llo.cloud
  Directory: .
  Status:    Ready
  Last pull: 2024-01-01 12:00:00
  Last push: 2024-01-01 12:05:00
```

**Output when no remotes:**
```
No remotes configured
Use 'tasklean remote add <name> <url>' to add a remote
```

**Planned output (Stage 10):**
```
Remote: origin (connected)
Last pull: 2024-01-01 12:00:00
Last push: 2024-01-01 12:05:00

Tasks: 15 total, 5 done, 10 in progress

Local changes: 2
Remote changes: 0
```

---

## Configuration

**Directory:** `~/.tasklean/`

**Files:** Only `{name}.conf` — one file per remote. No global config file.

**Naming:** `tasklean pull origin` → loads `~/.tasklean/origin.conf`

---

## TUI Components (Bubble Tea)

### Implemented
- **Token input** (`internal/tui/token_input.go`) — secure password-style input, EchoPassword mode
- **Confirmation** (`internal/tui/confirm.go`) — y/n dialog for destructive actions

### Planned
- **Progress view** — progress bar for pull/push operations
- **Conflict resolution** — overwrite / skip / merge UI

---

## Acceptance Criteria

| Criterion | Status |
|----------|--------|
| All 5 commands implemented | ✓ remote add/remove, status, pull, push (pull/push stubs) |
| Token input via TUI | ✓ Bubble Tea, masked |
| Confirmation for remove | ✓ Bubble Tea y/n |
| Error handling with clear messages | ✓ |
| Config persisted in `~/.tasklean/<name>.conf` | ✓ |
| Works offline for status | ✓ |
| Token encrypted at rest | ⏳ Planned |
| Pull fetches tasks from API | ⏳ Planned |
| Push syncs local files to API | ⏳ Planned |

---

## Dependencies

- **Cobra** (`github.com/spf13/cobra`) — CLI framework
- **Bubble Tea** (`github.com/charmbracelet/bubbletea`) — TUI framework
- **Lipgloss** (`github.com/charmbracelet/lipgloss`) — styling
- **Bubbles** (`github.com/charmbracelet/bubbles`) — textinput component

**Planned:** gostore for encrypted token storage

---

## File Structure

```
tasklean/
├── cmd/
│   └── tasklean/
│       └── main.go              # Entry point
├── internal/
│   ├── commands/                # CLI command handlers (Cobra)
│   │   ├── root.go
│   │   ├── remote.go            # Parent: remote add/remove
│   │   ├── remote_add.go
│   │   ├── remote_remove.go
│   │   ├── pull.go
│   │   ├── push.go
│   │   └── status.go
│   ├── config/                  # Configuration
│   │   ├── config.go            # DefaultConfig, Load, GetConfigDir
│   │   └── remote.go            # RemoteConfig, Load/Save/Remove/List
│   ├── tui/                     # Bubble Tea components
│   │   ├── token_input.go
│   │   └── confirm.go
│   ├── models/                  # Planned
│   ├── storage/                 # Planned
│   └── sync/                    # Planned
└── pkg/
    └── tracker/                 # Planned: Tracker adapters
```

---

## Usage Examples

```bash
# Add remote (TUI prompts for token)
tasklean remote add origin https://task.h3llo.cloud
tasklean remote add jira https://jira.example.com -d ./jira-tasks

# List remotes
tasklean status

# Pull tasks (planned: full implementation)
tasklean pull origin

# Push tasks (planned: full implementation)
tasklean push origin

# Remove remote (TUI prompts for confirmation)
tasklean remote remove origin
```

---

## API Reference

- [docs/functions.md](functions.md) — справочник по функциям, типам и константам
- [docs/pull-push.md](pull-push.md) — логика pull, структура хранения, подсказки для push

---

## Error Messages

| Situation | Message |
|-----------|---------|
| Remote not found (pull/push) | `remote "<name>" not found: failed to read config: ...` |
| Remote already exists (add) | `remote "<name>" already exists` |
| Remote does not exist (remove) | `remote "<name>" does not exist` |
| Token input cancelled | `token input cancelled: ...` |
| Empty token | `token cannot be empty` |
| Config dir creation failed | `failed to create config directory: ...` |
