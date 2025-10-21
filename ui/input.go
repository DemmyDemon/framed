package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type InputScreen struct {
	height  int
	width   int
	publish chan string
}

func NewInputScreen() (chan string, InputScreen) {
	publish := make(chan string)
	return publish, InputScreen{
		height:  10,
		publish: publish,
	}
}

func (is InputScreen) Init() tea.Cmd {
	return nil
}

func (is InputScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return is, tea.Quit
		}
	case tea.WindowSizeMsg:
		is.height = msg.Height
		is.width = msg.Width
	}
	return is, nil
}

func (is InputScreen) View() string {
	return "Input screen is very much a work in progress!"
}
