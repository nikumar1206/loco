package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type YesNoModel struct {
	Question string
	Choice   string // "yes", "no", or ""
}

func NewYesNoModel(question string) YesNoModel {
	return YesNoModel{Question: question}
}

func (m YesNoModel) Init() tea.Cmd {
	return nil
}

func (m YesNoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.Choice = "yes"
			return m, tea.Quit
		case "n", "N":
			m.Choice = "no"
			return m, tea.Quit
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m YesNoModel) View() string {
	if m.Choice == "" {
		return fmt.Sprintf("%s (y/n): ", m.Question)
	}
	return ""
}

func AskYesNo(question string) (bool, error) {
	p := tea.NewProgram(NewYesNoModel(question))
	model, err := p.Run()
	if err != nil {
		return false, err
	}

	m := model.(YesNoModel)
	return m.Choice == "yes", nil
}
