package transform

import (
	"html"
	"regexp"
	"strings"
)

// Plane HTML patterns
var (
	checklistULRegex  = regexp.MustCompile(`(?s)<ul[^>]*data-type="taskList"[^>]*>(.*?)</ul>`)
	checklistLiRegex  = regexp.MustCompile(`(?s)<li[^>]*data-checked="(true|false)"[^>]*>.*?<p[^>]*class="editor-paragraph-block"[^>]*>(.*?)</p>`)
	simpleListULRegex = regexp.MustCompile(`(?s)<ul[^>]*list-disc[^>]*>(.*?)</ul>`)
	simpleListLiRegex = regexp.MustCompile(`(?s)<li[^>]*>.*?<p[^>]*class="editor-paragraph-block"[^>]*>(.*?)</p>`)
	paragraphRegex    = regexp.MustCompile(`<p[^>]*class="editor-paragraph-block"[^>]*>([\s\S]*?)</p>`)
	htmlTagRegex      = regexp.MustCompile(`<[^>]+>`)
	emptyParagraph    = regexp.MustCompile(`<p[^>]*>\s*</p>`)

	// Our own generated format: <ul class="contains-task-list"> with <input type="checkbox">
	ownTaskListULRegex = regexp.MustCompile(`(?s)<ul[^>]*contains-task-list[^>]*>(.*?)</ul>`)
	ownTaskListLiRegex = regexp.MustCompile(`(?s)<li[^>]*task-list-item[^>]*><input[^>]*type="checkbox"([^>]*)>(.*?)(?:</li>|$)`)

	// Broken format from old push: <ul><li>[ ] text</li></ul> (no checkbox element)
	brokenTaskListULRegex = regexp.MustCompile(`(?s)<ul[^>]*>(.*?)</ul>`)
	brokenTaskListLiRegex = regexp.MustCompile(`(?s)<li[^>]*>\[[ xX]\] (.*?)</li>`)
)

// HTMLToMarkdown converts Plane checklist HTML to Markdown.
// Input: <ul data-type="taskList"><li data-checked="false">...<p class="editor-paragraph-block">text</p>...
// Output: - [ ] item\n- [x] item
func HTMLToMarkdown(htmlContent string) string {
	ulMatch := checklistULRegex.FindStringSubmatch(htmlContent)
	if ulMatch == nil {
		return htmlContent
	}
	ulContent := ulMatch[1]
	liMatches := checklistLiRegex.FindAllStringSubmatch(ulContent, -1)
	if len(liMatches) == 0 {
		return htmlContent
	}
	var lines []string
	for _, m := range liMatches {
		checked := m[1] == "true"
		rawText := m[2]
		text := stripHTML(rawText)
		text = html.UnescapeString(text)
		text = strings.TrimSpace(text)
		if checked {
			lines = append(lines, "- [x] "+text)
		} else {
			lines = append(lines, "- [ ] "+text)
		}
	}
	return strings.Join(lines, "\n")
}

// SimpleListToMarkdown converts Plane bullet list HTML to Markdown.
// Input: <ul class="list-disc">...<li><p class="editor-paragraph-block">text</p></li>...
// Output: - item\n- item
func SimpleListToMarkdown(htmlContent string) string {
	ulMatch := simpleListULRegex.FindStringSubmatch(htmlContent)
	if ulMatch == nil {
		return htmlContent
	}
	ulContent := ulMatch[1]
	liMatches := simpleListLiRegex.FindAllStringSubmatch(ulContent, -1)
	if len(liMatches) == 0 {
		return htmlContent
	}
	var lines []string
	for _, m := range liMatches {
		rawText := m[1]
		text := stripHTML(rawText)
		text = html.UnescapeString(text)
		text = strings.TrimSpace(text)
		lines = append(lines, "- "+text)
	}
	return strings.Join(lines, "\n")
}

// TransformDescription replaces Plane HTML in text with Markdown.
// - Checklist <ul data-type="taskList"> → - [ ] / - [x] items
// - Our format <ul class="contains-task-list"> with <input type="checkbox"> → - [ ] / - [x]
// - Broken old format <ul><li>[ ] text</li> → - [ ] items
// - Simple list <ul class="list-disc"> → - item
// - <p class="editor-paragraph-block"> → plain text (<br>→\n)
// - Empty <p></p> → removed
func TransformDescription(text string) string {
	s := emptyParagraph.ReplaceAllString(text, "")
	s = checklistULRegex.ReplaceAllStringFunc(s, func(match string) string {
		return HTMLToMarkdown(match)
	})
	s = ownTaskListULRegex.ReplaceAllStringFunc(s, func(match string) string {
		return ownCheckboxListToMarkdown(match)
	})
	s = brokenTaskListULRegex.ReplaceAllStringFunc(s, func(match string) string {
		return brokenTaskListToMarkdown(match)
	})
	s = simpleListULRegex.ReplaceAllStringFunc(s, func(match string) string {
		return SimpleListToMarkdown(match)
	})
	s = paragraphRegex.ReplaceAllStringFunc(s, func(match string) string {
		sub := paragraphRegex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		content := stripHTML(sub[1])
		content = html.UnescapeString(content)
		return strings.TrimSpace(content)
	})
	return s
}

// ownCheckboxListToMarkdown converts our generated checklist HTML back to GFM.
// Input: <ul class="contains-task-list"><li class="task-list-item"><input type="checkbox" disabled=""> text</li></ul>
func ownCheckboxListToMarkdown(htmlContent string) string {
	ulMatch := ownTaskListULRegex.FindStringSubmatch(htmlContent)
	if ulMatch == nil {
		return htmlContent
	}
	liMatches := ownTaskListLiRegex.FindAllStringSubmatch(htmlContent, -1)
	if len(liMatches) == 0 {
		return htmlContent
	}
	var lines []string
	for _, m := range liMatches {
		attrs := m[1]
		rawText := m[2]
		checked := strings.Contains(attrs, "checked")
		text := strings.TrimSpace(stripHTML(rawText))
		text = html.UnescapeString(text)
		if checked {
			lines = append(lines, "- [x] "+text)
		} else {
			lines = append(lines, "- [ ] "+text)
		}
	}
	return strings.Join(lines, "\n")
}

// brokenTaskListToMarkdown converts old broken format <li>[ ] text</li> back to GFM.
// This handles task lists that were pushed before the checkbox fix was applied.
func brokenTaskListToMarkdown(htmlContent string) string {
	liMatches := brokenTaskListLiRegex.FindAllStringSubmatch(htmlContent, -1)
	if len(liMatches) == 0 {
		return htmlContent
	}
	var lines []string
	for _, m := range liMatches {
		fullMatch := m[0] // e.g. "<li>[ ] text</li>" or "<li>[x] text</li>"
		text := strings.TrimSpace(stripHTML(m[1]))
		text = html.UnescapeString(text)
		// The char inside brackets is at the position after the first "[" in the match
		checked := false
		if idx := strings.Index(fullMatch, "["); idx >= 0 && idx+1 < len(fullMatch) {
			c := fullMatch[idx+1]
			checked = c == 'x' || c == 'X'
		}
		if checked {
			lines = append(lines, "- [x] "+text)
		} else {
			lines = append(lines, "- [ ] "+text)
		}
	}
	return strings.Join(lines, "\n")
}

func stripHTML(s string) string {
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	return htmlTagRegex.ReplaceAllString(s, "")
}
