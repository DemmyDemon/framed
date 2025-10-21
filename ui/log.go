package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const maxEntries = 200

type LogEntry struct {
	Payload string
}

type LogScreen struct {
	height  int
	width   int
	entries []LogEntry
	sub     chan LogEntry
}

func NewLogScreen() (chan LogEntry, LogScreen) {
	sub := make(chan LogEntry)
	return sub, LogScreen{
		height:  10,
		sub:     sub,
		entries: []LogEntry{},
	}
}

func mkLogEntry(payload string) tea.Cmd {
	return func() tea.Msg {
		return LogEntry{Payload: payload}
	}
}

func waitForEntry(sub chan LogEntry) tea.Cmd {
	return func() tea.Msg {
		return LogEntry(<-sub)
	}
}

func (ls LogScreen) Init() tea.Cmd {
	return tea.Batch(
		mkLogEntry("Starting..."),
		waitForEntry(ls.sub),
	)
}

func (ls LogScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case " ", "enter":
			return ls, mkLogEntry(time.Now().Format("--- 2006-01-02 15:04:05 ---"))
		}
	case tea.WindowSizeMsg:
		ls.height = msg.Height
		ls.width = msg.Width
	case LogEntry:

		if len(ls.entries) >= maxEntries {
			copy(ls.entries, ls.entries[(len(ls.entries)-maxEntries)+1:])
			ls.entries = ls.entries[:maxEntries-1]
		}

		// FIXME: This will probably clobber messages if there are a lot.
		ls.entries = append(ls.entries, msg)
		return ls, waitForEntry(ls.sub)
	}
	return ls, nil
}

func (ls LogScreen) View() string {
	if len(ls.entries) == 0 {
		return ""
	}
	var screen strings.Builder
	start := len(ls.entries) - ls.height
	start = max(0, start)
	i := 0
	for i = start; i < len(ls.entries); i++ {
		if i != start {
			screen.WriteRune('\n')
		}
		screen.WriteString(ls.entries[i].Payload)
	}
	if i < ls.height {
		screen.WriteString(strings.Repeat("\n", ls.height-i))
	}

	return screen.String()
}
