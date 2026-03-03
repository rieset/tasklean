package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/rie/tasklean/internal/models"
)

const (
	indexFilename      = "_index.json"
	unassignedDir      = "all"
	idMarkerPrefix     = "[id:"
	idMarkerSuffix     = "]"
	taskSeparator      = "\n\n\n---\n" // written format: two blank lines before ---
	taskSeparatorParse = "\n---\n"     // parser splits on this, matches both old and new format
	statusLegend       = "📥 backlog | 📋 todo | 🚀 in progress | 👀 in review | ✅ done | ❌ cancelled"
	statusLegendDesc   = "*Обозначения статусов для изменения статуса задачи при push*"
)

// Matches [emoji id:uuid] or [id:uuid]. Group 1: optional emoji, Group 2: id.
var idMarkerRegex = regexp.MustCompile(`^\[([^\s\]]*\s+)?id:([^\]]+)\]\s*`)

type indexEntry struct {
	Remote    string `json:"remote,omitempty"`
	Module    string `json:"module,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type taskIndex map[string]indexEntry

var statusesByLen = []models.TaskStatus{
	models.StatusInProgress, models.StatusInReview, models.StatusCancelled,
	models.StatusBacklog, models.StatusTodo, models.StatusDone,
}

func sanitizeAssigneeForDir(email string) string {
	s := strings.TrimSpace(strings.ToLower(email))
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == '@' {
			b.WriteString("-at-")
		} else if r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-")
}

func sanitizeModuleForFilename(module string) string {
	s := strings.TrimSpace(module)
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '_' {
			b.WriteByte('-')
		} else if r == '-' {
			b.WriteByte('-')
		}
	}
	return strings.ToLower(strings.Trim(b.String(), "-"))
}

func statusFilename(status models.TaskStatus, module string) string {
	base := string(status)
	if module != "" {
		sanitized := sanitizeModuleForFilename(module)
		if sanitized != "" {
			base = base + "-" + sanitized
		}
	}
	return base + ".md"
}

type parsedBlock struct {
	ID     string
	Body   string
	Marker string // raw marker line, e.g. "[📋 id:uuid]" for preservation
}

func formatTaskBlock(id, body string, status models.TaskStatus) string {
	emoji := status.Emoji()
	marker := "[" + emoji + " id:" + id + "]"
	return marker + "\n\n" + strings.TrimSpace(body)
}

func formatParsedBlock(b parsedBlock) string {
	return b.Marker + "\n\n" + strings.TrimSpace(b.Body)
}

func formatStatusFileContent(blocks []string, status models.TaskStatus, module string) string {
	title := status.DisplayName()
	if module != "" {
		title = title + " — " + module
	}
	header := "# " + title + "\n\n" + statusLegend + "\n" + statusLegendDesc + "\n\n"
	if len(blocks) == 0 {
		return header
	}
	return header + strings.Join(blocks, taskSeparator) + "\n"
}

func stripHeader(content string) string {
	s := strings.TrimSpace(content)
	// Strip "# Title" (H1)
	if strings.HasPrefix(s, "# ") {
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = strings.TrimSpace(s[idx+1:])
		} else {
			return ""
		}
	}
	// Strip legend line (emoji line)
	if strings.HasPrefix(s, "📥") || strings.HasPrefix(s, "📋") {
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = strings.TrimSpace(s[idx+1:])
		} else {
			return ""
		}
	}
	// Strip legend description line (*...*)
	if strings.HasPrefix(s, "*") || strings.HasPrefix(s, "_") {
		if idx := strings.Index(s, "\n"); idx >= 0 {
			return strings.TrimSpace(s[idx+1:])
		}
		return ""
	}
	return content
}

func parseTaskBlocks(content string) []parsedBlock {
	content = stripHeader(content)
	var result []parsedBlock
	blocks := strings.Split(content, taskSeparatorParse)
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		match := idMarkerRegex.FindStringSubmatch(block)
		if match == nil {
			continue
		}
		id := match[2]
		marker := strings.TrimSpace(idMarkerRegex.FindString(block))
		body := strings.TrimSpace(idMarkerRegex.ReplaceAllString(block, ""))
		result = append(result, parsedBlock{ID: id, Body: body, Marker: marker})
	}
	return result
}

func statusFromMarker(marker string) (models.TaskStatus, bool) {
	match := idMarkerRegex.FindStringSubmatch(marker)
	if match == nil || match[1] == "" {
		return "", false
	}
	emoji := strings.TrimSpace(match[1])
	st := models.EmojiToStatus(emoji)
	return st, st != ""
}

func removeTaskFromOtherFiles(taskID, directory, keepFilename string) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read directory: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || e.Name() == keepFilename || e.Name() == indexFilename || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if _, _, ok := parseStatusFilename(e.Name()); !ok {
			continue
		}
		path := filepath.Join(directory, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		blocks := parseTaskBlocks(string(data))
		var newBlocks []string
		removed := false
		for _, b := range blocks {
			if b.ID == taskID {
				removed = true
				continue
			}
			newBlocks = append(newBlocks, formatParsedBlock(b))
		}
		if removed {
			status, module := parseStatusFromFilename(e.Name())
			out := formatStatusFileContent(newBlocks, status, module)
			if err := os.WriteFile(path, []byte(out), 0644); err != nil {
				return fmt.Errorf("remove task from %s: %w", e.Name(), err)
			}
		}
	}
	return nil
}

func parseStatusFromFilename(name string) (models.TaskStatus, string) {
	status, module, ok := parseStatusFilename(name)
	if !ok {
		return models.StatusTodo, ""
	}
	return status, module
}

func parseStatusFilename(name string) (models.TaskStatus, string, bool) {
	base := strings.TrimSuffix(name, ".md")
	if base == name {
		return "", "", false
	}
	for _, st := range statusesByLen {
		s := string(st)
		if base == s {
			return st, "", true
		}
		if strings.HasPrefix(base, s+"-") {
			module := base[len(s)+1:]
			if module != "" {
				return st, module, true
			}
		}
	}
	return "", "", false
}

func saveTaskToDir(task *models.Task, dir string, body string, taskBlock string) error {
	targetFilename := statusFilename(task.Status, task.Module)
	if err := removeTaskFromOtherFiles(task.ID, dir, targetFilename); err != nil {
		return err
	}

	statusFilepath := filepath.Join(dir, targetFilename)
	var existingBlocks []parsedBlock
	if data, err := os.ReadFile(statusFilepath); err == nil {
		existingBlocks = parseTaskBlocks(string(data))
	}

	found := false
	for i := range existingBlocks {
		if existingBlocks[i].ID == task.ID {
			// Preserve any resolve block that was added during a skipped push.
			oldResolve := ExtractResolveBlock(existingBlocks[i].Body)
			existingBlocks[i].Body = MergeResolveBlock(body, oldResolve)
			existingBlocks[i].Marker = "[" + task.Status.Emoji() + " id:" + task.ID + "]"
			found = true
			break
		}
	}
	var blocks []string
	if found {
		for _, b := range existingBlocks {
			blocks = append(blocks, formatParsedBlock(b))
		}
	} else {
		blocks = append(blocks, taskBlock)
		for _, b := range existingBlocks {
			blocks = append(blocks, formatParsedBlock(b))
		}
	}

	out := formatStatusFileContent(blocks, task.Status, task.Module)
	if err := os.WriteFile(statusFilepath, []byte(out), 0644); err != nil {
		return fmt.Errorf("write status file: %w", err)
	}
	return nil
}

func removeTaskFromOldAssigneeDirs(taskID, baseDir string, keepAssignees map[string]bool) error {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if keepAssignees[e.Name()] {
			continue
		}
		dir := filepath.Join(baseDir, e.Name())
		if err := removeTaskFromDir(taskID, dir); err != nil {
			return err
		}
	}
	return nil
}

func removeTaskFromDir(taskID, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == indexFilename {
			continue
		}
		if _, _, ok := parseStatusFilename(e.Name()); !ok {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		blocks := parseTaskBlocks(string(data))
		var newBlocks []string
		removed := false
		for _, b := range blocks {
			if b.ID == taskID {
				removed = true
				continue
			}
			newBlocks = append(newBlocks, formatParsedBlock(b))
		}
		if removed {
			status, module := parseStatusFromFilename(e.Name())
			out := formatStatusFileContent(newBlocks, status, module)
			if err := os.WriteFile(path, []byte(out), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

func SaveTask(task *models.Task, directory string) error {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	body := formatTaskBody(task.Title, task.Description)
	taskBlock := formatTaskBlock(task.ID, body, task.Status)

	keepAssignees := make(map[string]bool)
	if len(task.Assignees) == 0 {
		keepAssignees[unassignedDir] = true
		allDir := filepath.Join(directory, unassignedDir)
		if err := os.MkdirAll(allDir, 0755); err != nil {
			return fmt.Errorf("create unassigned dir: %w", err)
		}
		if err := saveTaskToDir(task, allDir, body, taskBlock); err != nil {
			return err
		}
	} else {
		for _, email := range task.Assignees {
			dirName := sanitizeAssigneeForDir(email)
			if dirName == "" {
				continue
			}
			keepAssignees[dirName] = true
			assigneeDir := filepath.Join(directory, dirName)
			if err := os.MkdirAll(assigneeDir, 0755); err != nil {
				return fmt.Errorf("create assignee dir: %w", err)
			}
			if err := saveTaskToDir(task, assigneeDir, body, taskBlock); err != nil {
				return err
			}
		}
	}

	if err := removeTaskFromOldAssigneeDirs(task.ID, directory, keepAssignees); err != nil {
		return err
	}
	_ = removeTaskFromDir(task.ID, directory)

	idx, err := loadIndex(directory)
	if err != nil {
		return err
	}
	if idx == nil {
		idx = make(taskIndex)
	}
	idx[task.ID] = indexEntry{
		Remote:    task.Remote,
		Module:    task.Module,
		CreatedAt: task.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: task.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	return saveIndex(directory, idx)
}

func LoadTask(taskID, directory string) (*models.Task, error) {
	idx, _ := loadIndex(directory)

	tryDir := func(dir string) (*models.Task, error) {
		list, err := tasksFromDir(dir, idx)
		if err != nil {
			return nil, err
		}
		for _, t := range list {
			if t.ID == taskID {
				return t, nil
			}
		}
		return nil, nil
	}

	if t, err := tryDir(directory); err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	} else if t != nil {
		return t, nil
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		subdir := filepath.Join(directory, e.Name())
		if t, err := tryDir(subdir); err != nil {
			return nil, fmt.Errorf("read %s: %w", subdir, err)
		} else if t != nil {
			return t, nil
		}
	}
	return nil, fmt.Errorf("task %q not found", taskID)
}

func tasksFromDir(dir string, idx taskIndex) ([]*models.Task, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var tasks []*models.Task
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == indexFilename {
			continue
		}
		status, module, ok := parseStatusFilename(e.Name())
		if !ok {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		blocks := parseTaskBlocks(string(data))
		for _, b := range blocks {
			if b.ID == "" {
				continue
			}
			title, desc := parseTaskBody(b.Body)
			idxEntry := idx[b.ID]
			createdAt, _ := parseTime(idxEntry.CreatedAt)
			updatedAt, _ := parseTime(idxEntry.UpdatedAt)
			mod := idxEntry.Module
			if mod == "" {
				mod = module
			}
			taskStatus := status
			if st, ok := statusFromMarker(b.Marker); ok {
				taskStatus = st
			}
			tasks = append(tasks, &models.Task{
				ID:          b.ID,
				Title:       title,
				Description: desc,
				Status:      taskStatus,
				Module:      mod,
				Remote:      idxEntry.Remote,
				CreatedAt:   createdAt,
				UpdatedAt:   updatedAt,
			})
		}
	}
	return tasks, nil
}

func ListTasks(directory string) ([]*models.Task, error) {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return []*models.Task{}, nil
	}

	idx, _ := loadIndex(directory)
	seen := make(map[string]bool)
	var tasks []*models.Task

	collect := func(dir string) error {
		list, err := tasksFromDir(dir, idx)
		if err != nil {
			return err
		}
		for _, t := range list {
			if !seen[t.ID] {
				seen[t.ID] = true
				tasks = append(tasks, t)
			}
		}
		return nil
	}

	if err := collect(directory); err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		subdir := filepath.Join(directory, e.Name())
		if err := collect(subdir); err != nil {
			return nil, fmt.Errorf("read %s: %w", subdir, err)
		}
	}
	return tasks, nil
}

func DeleteTask(taskID, directory string) error {
	found := false
	if err := deleteTaskFromDir(taskID, directory); err == nil {
		found = true
	}
	if entries, err := os.ReadDir(directory); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dir := filepath.Join(directory, e.Name())
			if deleteTaskFromDir(taskID, dir) == nil {
				found = true
			}
		}
	}
	if !found {
		return fmt.Errorf("task %q not found", taskID)
	}
	idx, _ := loadIndex(directory)
	if idx != nil {
		delete(idx, taskID)
		_ = saveIndex(directory, idx)
	}
	return nil
}

func deleteTaskFromDir(taskID, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == indexFilename {
			continue
		}
		if _, _, ok := parseStatusFilename(e.Name()); !ok {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		blocks := parseTaskBlocks(string(data))
		var newBlocks []string
		for _, b := range blocks {
			if b.ID != taskID {
				newBlocks = append(newBlocks, formatParsedBlock(b))
			}
		}
		if len(newBlocks) != len(blocks) {
			status, module := parseStatusFromFilename(e.Name())
			out := formatStatusFileContent(newBlocks, status, module)
			return os.WriteFile(path, []byte(out), 0644)
		}
	}
	return fmt.Errorf("not found")
}

func loadIndex(directory string) (taskIndex, error) {
	path := filepath.Join(directory, indexFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var idx taskIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}
	return idx, nil
}

func saveIndex(directory string, idx taskIndex) error {
	if idx == nil {
		return nil
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}
	path := filepath.Join(directory, indexFilename)
	return os.WriteFile(path, data, 0644)
}

// UpdateIndexUpdatedAt updates the updated_at field for a task in the index.
// Used after push to keep the index in sync with the tracker.
func UpdateIndexUpdatedAt(taskID, directory, updatedAt string) error {
	idx, err := loadIndex(directory)
	if err != nil || idx == nil {
		return err
	}
	entry, ok := idx[taskID]
	if !ok {
		return nil
	}
	entry.UpdatedAt = updatedAt
	idx[taskID] = entry
	return saveIndex(directory, idx)
}

func formatTaskBody(title, description string) string {
	if title == "" && description == "" {
		return ""
	}
	if description == "" {
		return "## " + title + "\n"
	}
	return "## " + title + "\n\n" + description
}

func parseTaskBody(body string) (title, description string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", ""
	}
	if strings.HasPrefix(body, "## ") {
		idx := strings.Index(body, "\n")
		if idx >= 0 {
			title = strings.TrimSpace(body[3:idx])
			description = strings.TrimSpace(body[idx+1:])
		} else {
			title = strings.TrimSpace(body[3:])
		}
		return title, description
	}
	return "", body
}

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02T15:04:05Z", s)
}

// ReplaceTaskID replaces oldID with newTask.ID in all files and updates the index.
// Used after CreateTask when a locally-created task gets its real ID from the tracker.
func ReplaceTaskID(oldID string, newTask *models.Task, directory string) error {
	dirs := []string{directory}
	entries, err := os.ReadDir(directory)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, filepath.Join(directory, e.Name()))
			}
		}
	}

	replaced := false
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == indexFilename {
				continue
			}
			if _, _, ok := parseStatusFilename(e.Name()); !ok {
				continue
			}
			path := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			blocks := parseTaskBlocks(string(data))
			var modified []string
			fileChanged := false
			for _, b := range blocks {
				if b.ID == oldID {
					b.ID = newTask.ID
					b.Marker = "[" + newTask.Status.Emoji() + " id:" + newTask.ID + "]"
					fileChanged = true
					replaced = true
				}
				modified = append(modified, formatParsedBlock(b))
			}
			if fileChanged {
				status, module := parseStatusFromFilename(e.Name())
				out := formatStatusFileContent(modified, status, module)
				if err := os.WriteFile(path, []byte(out), 0644); err != nil {
					return fmt.Errorf("write %s: %w", path, err)
				}
			}
		}
	}

	if !replaced {
		return fmt.Errorf("task %q not found for ID replacement", oldID)
	}

	idx, _ := loadIndex(directory)
	if idx != nil {
		delete(idx, oldID)
		idx[newTask.ID] = indexEntry{
			Remote:    newTask.Remote,
			Module:    newTask.Module,
			CreatedAt: newTask.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt: newTask.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
		return saveIndex(directory, idx)
	}
	return nil
}
