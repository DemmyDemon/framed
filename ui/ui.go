package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type UI struct {
	screens   []tea.Model
	screenIdx int
	chLog     chan LogEntry
	chText    chan string
}

func NewUI(initialText string, filename string) (chan LogEntry, chan string, UI) {

	chLog, logScreen := NewLogScreen()
	chText, inputScreen := NewInputScreen(initialText, filename)

	ui := UI{
		screens: []tea.Model{inputScreen, logScreen},
		chLog:   chLog,
		chText:  chText,
	}

	return chLog, chText, ui

}

func (ui UI) Init() tea.Cmd {
	return nil
}

func (ui UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return ui, tea.Quit
		case "tab":
			ui.screenIdx++
			if ui.screenIdx >= len(ui.screens) {
				ui.screenIdx = 0
			}
			return ui, nil
		default:
			model, command := ui.screens[ui.screenIdx].Update(msg)
			ui.screens[ui.screenIdx] = model
			return ui, command
		}
	default:
		cmds := []tea.Cmd{}
		for i, model := range ui.screens {
			newModel, cmd := model.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			ui.screens[i] = newModel
		}
		return ui, tea.Batch(cmds...)
	}
	// return ui, nil
}

func (ui UI) View() string {
	return ui.screens[ui.screenIdx].View()
}
