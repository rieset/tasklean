package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tokenInputModel struct {
	input  textinput.Model
	prompt string
}

func newTokenInputModel(prompt string) tokenInputModel {
	ti := textinput.New()
	ti.Placeholder = "API token"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Width = 40
	ti.Focus()

	return tokenInputModel{
		input:  ti,
		prompt: prompt,
	}
}

func (m tokenInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m tokenInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m tokenInputModel) View() string {
	style := lipgloss.NewStyle().MarginLeft(1)
	return style.Render(m.prompt + "\n\n" + m.input.View())
}

func PromptToken(prompt string) (string, error) {
	if prompt == "" {
		prompt = "Enter API token:"
	}
	m := newTokenInputModel(prompt)
	p := tea.NewProgram(m, tea.WithInputTTY())
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	model := final.(tokenInputModel)
	return model.input.Value(), nil
}
