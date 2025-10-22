package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type InputScreen struct {
	height      int
	width       int
	publish     chan string
	textarea    textarea.Model
	drity       bool
	flushedText string
}

var (
	styleSaved = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleDirty = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

func NewInputScreen() (chan string, InputScreen) {
	publish := make(chan string)
	ta := textarea.New()
	ta.Placeholder = "(TODO: Load initial text from disk)"
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.EndOfBufferCharacter = '•'
	ta.Focus()
	return publish, InputScreen{
		height:   10,
		publish:  publish,
		textarea: ta,
	}
}

func (is InputScreen) Init() tea.Cmd {
	return textarea.Blink
}

func (is InputScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		is.height = msg.Height
		is.width = msg.Width
		is.textarea.SetHeight(is.height - 1)
		is.textarea.SetWidth(is.width)
		return is, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			is.flushedText = is.textarea.Value()
			is.publish <- is.flushedText
			is.drity = false
			return is, nil
		}
	}

	mod, cmd := is.textarea.Update(msg)
	if mod.Value() != is.flushedText {
		is.drity = true
	}
	is.textarea = mod
	return is, cmd
}

func (is InputScreen) View() string {
	now := time.Now()
	_, week := now.ISOWeek()
	style := styleSaved
	if is.drity {
		style = styleDirty
	}
	return style.Render(fmt.Sprintf("%s, week %d", now.Format("Monday"), week)) + "\n" + is.textarea.View()
}
