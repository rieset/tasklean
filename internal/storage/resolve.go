package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const resolveBlockHeader = "### tasklean: resolve"

// ExtractResolveBlock returns the resolve subsection (header + body) from task body,
// or an empty string if the block is not present.
func ExtractResolveBlock(body string) string {
	marker := "\n" + resolveBlockHeader
	if idx := strings.Index(body, marker); idx >= 0 {
		return strings.TrimSpace(body[idx:])
	}
	if strings.HasPrefix(strings.TrimSpace(body), resolveBlockHeader) {
		return strings.TrimSpace(body)
	}
	return ""
}

// StripResolveBlock returns the body with the resolve subsection removed.
func StripResolveBlock(body string) string {
	marker := "\n" + resolveBlockHeader
	if idx := strings.Index(body, marker); idx >= 0 {
		return strings.TrimRight(body[:idx], " \t\n")
	}
	if strings.HasPrefix(strings.TrimSpace(body), resolveBlockHeader) {
		return ""
	}
	return body
}

// MergeResolveBlock appends resolveBlock to newBody separated by a blank line.
// If resolveBlock is empty, newBody is returned unchanged.
func MergeResolveBlock(newBody, resolveBlock string) string {
	if resolveBlock == "" {
		return newBody
	}
	return strings.TrimRight(newBody, " \t\n") + "\n\n" + resolveBlock
}

// FormatResolveBlock builds a resolve block with a timestamp, reason, and an optional
// snapshot of the local task content that was being pushed.
// localSnapshot is pre-formatted markdown text (may be empty).
func FormatResolveBlock(reason, localSnapshot string) string {
	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	text := fmt.Sprintf(
		"Пропущено при push: %s — %s.\n"+
			"Разрешите вручную: выполните pull и при необходимости объедините изменения,\n"+
			"либо отредактируйте задачу в UI и повторите push.\n"+
			"Удалите этот блок после разрешения конфликта.",
		ts, reason,
	)
	if localSnapshot != "" {
		text += "\n\n**Ваши изменения, которые пытались запушить:**\n\n" + localSnapshot
	}
	return resolveBlockHeader + "\n\n" + text
}

// WriteResolveBlock locates the task by ID in the directory tree and upserts
// the resolve subsection in its body. localSnapshot is a pre-formatted markdown
// snapshot of the local task content that was being pushed (may be empty).
// Silently succeeds if the task is not found.
func WriteResolveBlock(taskID, directory, reason, localSnapshot string) error {
	rb := FormatResolveBlock(reason, localSnapshot)

	updated, err := writeResolveBlockInDir(taskID, directory, rb)
	if err != nil {
		return err
	}
	if updated {
		return nil
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("read directory: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		updated, err = writeResolveBlockInDir(taskID, filepath.Join(directory, e.Name()), rb)
		if err != nil {
			return err
		}
		if updated {
			return nil
		}
	}
	return nil
}

func writeResolveBlockInDir(taskID, dir, resolveBlock string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read dir %s: %w", dir, err)
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
		found := false
		for i := range blocks {
			if blocks[i].ID != taskID {
				continue
			}
			stripped := StripResolveBlock(blocks[i].Body)
			blocks[i].Body = MergeResolveBlock(stripped, resolveBlock)
			found = true
			break
		}
		if !found {
			continue
		}
		var formatted []string
		for _, b := range blocks {
			formatted = append(formatted, formatParsedBlock(b))
		}
		status, module := parseStatusFromFilename(e.Name())
		out := formatStatusFileContent(formatted, status, module)
		if err := os.WriteFile(path, []byte(out), 0644); err != nil {
			return false, fmt.Errorf("write %s: %w", path, err)
		}
		return true, nil
	}
	return false, nil
}

// ResolvedTask holds display metadata for a task with an unresolved push conflict.
type ResolvedTask struct {
	ID          string
	Title       string
	File        string
	ResolveText string // first non-empty line of the resolve block body (after header)
}

// ListResolvedTasks returns all tasks in the directory tree that have a resolve block.
func ListResolvedTasks(directory string) ([]ResolvedTask, error) {
	seen := make(map[string]bool)
	var result []ResolvedTask

	collect := func(dir string) error {
		items, err := resolvedTasksFromDir(dir, seen)
		if err != nil {
			return err
		}
		result = append(result, items...)
		return nil
	}

	if err := collect(directory); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := collect(filepath.Join(directory, e.Name())); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func resolvedTasksFromDir(dir string, seen map[string]bool) ([]ResolvedTask, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}
	var result []ResolvedTask
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
		for _, b := range parseTaskBlocks(string(data)) {
			if b.ID == "" || seen[b.ID] {
				continue
			}
			rb := ExtractResolveBlock(b.Body)
			if rb == "" {
				continue
			}
			seen[b.ID] = true
			title, _ := parseTaskBody(StripResolveBlock(b.Body))
			result = append(result, ResolvedTask{
				ID:          b.ID,
				Title:       title,
				File:        path,
				ResolveText: resolveFirstLine(rb),
			})
		}
	}
	return result, nil
}

// resolveFirstLine returns the first non-empty content line from the resolve block
// (skipping the header line itself).
func resolveFirstLine(resolveBlock string) string {
	for i, line := range strings.Split(resolveBlock, "\n") {
		if i == 0 {
			continue
		}
		if l := strings.TrimSpace(line); l != "" {
			return l
		}
	}
	return ""
}
