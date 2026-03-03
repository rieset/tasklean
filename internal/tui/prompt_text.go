package tui

import (
	"io"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type textPromptModel struct {
	input  textinput.Model
	prompt string
}

func newTextPromptModel(prompt, placeholder string) textPromptModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Width = 50
	ti.Focus()

	return textPromptModel{
		input:  ti,
		prompt: prompt,
	}
}

func (m textPromptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m textPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return m, tea.Quit
		case "ctrl+c", "esc":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m textPromptModel) View() string {
	style := lipgloss.NewStyle().MarginLeft(1)
	return style.Render(m.prompt + "\n\n" + m.input.View())
}

// PromptText shows a TUI prompt for plain text input.
// Returns the entered value. Empty string if user presses Enter without typing.
func PromptText(prompt, placeholder string) (string, error) {
	return PromptTextWithInput(prompt, placeholder, nil)
}

// PromptTextWithInput is like PromptText but accepts custom input for tests.
// When input is nil, uses TTY.
func PromptTextWithInput(prompt, placeholder string, input io.Reader) (string, error) {
	if prompt == "" {
		prompt = "Enter value:"
	}
	m := newTextPromptModel(prompt, placeholder)
	opts := []tea.ProgramOption{}
	if input != nil {
		opts = append(opts, tea.WithInput(input))
	} else {
		opts = append(opts, tea.WithInputTTY())
	}
	p := tea.NewProgram(m, opts...)
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	model := final.(textPromptModel)
	return model.input.Value(), nil
}
