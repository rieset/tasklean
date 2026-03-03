package transform

import (
	"strings"
	"testing"
)

func TestHTMLToMarkdown(t *testing.T) {
	html := `<ul class="not-prose pl-2 space-y-2" data-type="taskList"><li class="relative" data-checked="false" data-type="taskItem"><label><input type="checkbox"><span></span></label><div><p class="editor-paragraph-block">Proxmox</p></div></li><li class="relative" data-checked="false" data-type="taskItem"><label><input type="checkbox"><span></span></label><div><p class="editor-paragraph-block">Бэкап кластеров</p></div></li><li class="relative" data-checked="true" data-type="taskItem"><label><input type="checkbox"><span></span></label><div><p class="editor-paragraph-block">Done item</p></div></li></ul>`
	got := HTMLToMarkdown(html)
	want := "- [ ] Proxmox\n- [ ] Бэкап кластеров\n- [x] Done item"
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestHTMLToMarkdown_NoMatch(t *testing.T) {
	html := `<p>Just a paragraph</p>`
	got := HTMLToMarkdown(html)
	if got != html {
		t.Errorf("unchanged content: got %q", got)
	}
}

func TestTransformDescription(t *testing.T) {
	text := `Some intro

<ul class="not-prose" data-type="taskList"><li data-checked="false" data-type="taskItem"><p class="editor-paragraph-block">Item 1</p></li><li data-checked="true" data-type="taskItem"><p class="editor-paragraph-block">Item 2</p></li></ul>

Trailing text`
	got := TransformDescription(text)
	if !strings.Contains(got, "- [ ] Item 1") {
		t.Errorf("missing unchecked item: %s", got)
	}
	if !strings.Contains(got, "- [x] Item 2") {
		t.Errorf("missing checked item: %s", got)
	}
	if !strings.Contains(got, "Some intro") {
		t.Errorf("missing intro: %s", got)
	}
	if !strings.Contains(got, "Trailing text") {
		t.Errorf("missing trailing: %s", got)
	}
}

func TestTransformDescription_SimpleList(t *testing.T) {
	html := `<ul class="list-disc pl-7 space-y-[--list-spacing-y] tight" data-tight="true"><li class="not-prose space-y-2"><p class="editor-paragraph-block">default b2c</p></li><li class="not-prose space-y-2"><p class="editor-paragraph-block">default b2b</p></li><li class="not-prose space-y-2"><p class="editor-paragraph-block">production b2c</p></li><li class="not-prose space-y-2"><p class="editor-paragraph-block">production b2b</p></li></ul><p class="editor-paragraph-block"></p><p class="editor-paragraph-block">На текущий момент надо разворачивать 4 модификации на разные стенды<br>В соответствии со схемой</p>`
	got := TransformDescription(html)
	if !strings.Contains(got, "- default b2c") {
		t.Errorf("missing list item: %s", got)
	}
	if !strings.Contains(got, "- production b2b") {
		t.Errorf("missing list item: %s", got)
	}
	if !strings.Contains(got, "На текущий момент надо разворачивать") {
		t.Errorf("missing paragraph: %s", got)
	}
	if !strings.Contains(got, "В соответствии со схемой") {
		t.Errorf("missing br-separated line: %s", got)
	}
}

func TestHTMLToMarkdown_UserExample(t *testing.T) {
	html := `<ul class="not-prose pl-2 space-y-2" data-type="taskList"><li class="relative" data-checked="false" data-type="taskItem"><label><input type="checkbox"><span></span></label><div><p class="editor-paragraph-block">Железный "белый" бекап сервер на базе s3 (настройка сервера, настройка интеграций (БД, velero, gitlab, control plane))</p></div></li><li class="relative" data-checked="false" data-type="taskItem"><label><input type="checkbox"><span></span></label><div><p class="editor-paragraph-block">Изучить вопрос получаения БД из продакшен кластера для запуска новой версии по методу blue/green</p></div></li></ul>`
	got := HTMLToMarkdown(html)
	if !strings.Contains(got, "- [ ] Железный") {
		t.Errorf("first item: %s", got)
	}
	if !strings.Contains(got, "velero, gitlab, control plane") {
		t.Errorf("nested parens: %s", got)
	}
	if !strings.Contains(got, "blue/green") {
		t.Errorf("second item: %s", got)
	}
}

func TestMarkdownToHTML_TaskList(t *testing.T) {
	md := "- [ ] Unchecked item\n- [x] Checked item\n- [X] Also checked"
	got := MarkdownToHTML(md)
	if !strings.Contains(got, `<input type="checkbox" disabled="">`) {
		t.Errorf("expected unchecked checkbox, got:\n%s", got)
	}
	if !strings.Contains(got, `<input type="checkbox" checked="" disabled="">`) {
		t.Errorf("expected checked checkbox, got:\n%s", got)
	}
	if !strings.Contains(got, `class="contains-task-list"`) {
		t.Errorf("expected task list class, got:\n%s", got)
	}
}

func TestMarkdownToHTML_MixedContent(t *testing.T) {
	md := "## Heading\n\nSome text\n\n- [ ] Task one\n- [x] Task done\n\nMore text"
	got := MarkdownToHTML(md)
	if !strings.Contains(got, "<h2") {
		t.Errorf("expected heading, got:\n%s", got)
	}
	if !strings.Contains(got, `contains-task-list`) {
		t.Errorf("expected task list, got:\n%s", got)
	}
	if !strings.Contains(got, "More text") {
		t.Errorf("expected trailing text, got:\n%s", got)
	}
}

func TestTransformDescription_EmptyParagraph(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"<p></p>", ""},
		{"Text<p></p>More", "TextMore"},
		{"A<p class=\"editor-paragraph-block\"></p>B", "AB"},
		{"<p></p>\n\n<p></p>", "\n\n"},
	}
	for _, tt := range tests {
		got := TransformDescription(tt.in)
		if got != tt.want {
			t.Errorf("TransformDescription(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
