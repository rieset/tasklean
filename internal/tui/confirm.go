package tui

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type confirmModel struct {
	prompt    string
	confirmed *bool
}

func newConfirmModel(prompt string) confirmModel {
	return confirmModel{prompt: prompt}
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			yes := true
			m.confirmed = &yes
			return m, tea.Quit
		case "n", "N", "ctrl+c", "esc":
			no := false
			m.confirmed = &no
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	style := lipgloss.NewStyle().MarginLeft(1)
	return style.Render(m.prompt + "\n\n  y/n to confirm/cancel")
}

func Confirm(prompt string) (bool, error) {
	return ConfirmWithInput(prompt, nil)
}

func ConfirmWithInput(prompt string, input io.Reader) (bool, error) {
	if prompt == "" {
		prompt = "Are you sure?"
	}
	m := newConfirmModel(prompt)
	opts := []tea.ProgramOption{}
	if input != nil {
		opts = append(opts, tea.WithInput(input))
	} else {
		opts = append(opts, tea.WithInputTTY())
	}
	p := tea.NewProgram(m, opts...)
	final, err := p.Run()
	if err != nil {
		return false, err
	}
	model := final.(confirmModel)
	if model.confirmed != nil {
		return *model.confirmed, nil
	}
	return false, nil
}
