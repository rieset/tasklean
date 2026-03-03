package transform

import (
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// MarkdownToHTML converts Markdown to HTML for Plane description_html.
// Uses plain HTML output without Smartypants to avoid HTML entities that Plane rejects.
// GFM task lists (- [ ] / - [x] ) are pre-processed into HTML checkboxes
// because gomarkdown has no native task list extension.
func MarkdownToHTML(md string) string {
	if strings.TrimSpace(md) == "" {
		return "<p></p>"
	}
	md = convertTaskLists(md)
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.LaxHTMLBlocks
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(md))
	renderer := html.NewRenderer(html.RendererOptions{Flags: html.UseXHTML})
	result := strings.TrimSpace(string(markdown.Render(doc, renderer)))
	if result == "" {
		return "<p></p>"
	}
	return result
}

// convertTaskLists replaces GFM task list items with HTML checkbox elements.
// Consecutive task list items are grouped into a single <ul class="contains-task-list">.
func convertTaskLists(md string) string {
	lines := strings.Split(md, "\n")
	var sb strings.Builder
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimLeft(lines[i], " \t")
		if isTaskItem(trimmed) {
			sb.WriteString("\n<ul class=\"contains-task-list\">\n")
			for i < len(lines) {
				trimmed = strings.TrimLeft(lines[i], " \t")
				if !isTaskItem(trimmed) {
					break
				}
				checked := len(trimmed) > 3 && (trimmed[3] == 'x' || trimmed[3] == 'X')
				text := strings.TrimSpace(trimmed[6:])
				sb.WriteString(taskItemHTML(text, checked))
				sb.WriteByte('\n')
				i++
			}
			sb.WriteString("</ul>\n\n")
		} else {
			sb.WriteString(lines[i])
			sb.WriteByte('\n')
			i++
		}
	}
	return sb.String()
}

func isTaskItem(trimmed string) bool {
	return len(trimmed) >= 6 &&
		(strings.HasPrefix(trimmed, "- [ ] ") ||
			strings.HasPrefix(trimmed, "- [x] ") ||
			strings.HasPrefix(trimmed, "- [X] "))
}

func taskItemHTML(text string, checked bool) string {
	cb := `<input type="checkbox" disabled="">`
	if checked {
		cb = `<input type="checkbox" checked="" disabled="">`
	}
	return `<li class="task-list-item">` + cb + ` ` + text + `</li>`
}
