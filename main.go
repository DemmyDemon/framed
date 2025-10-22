package main

import (
	"fmt"
	"os"

	"github.com/DemmyDemon/framed/server"
	"github.com/DemmyDemon/framed/ui"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	PORT = 7100
)

func main() {

	chLog, chText, prog := ui.NewUI()
	go func() {
		chLog <- ui.NewLogEntry(fmt.Sprintf("Shall listen on port %d\n", PORT))
		err := server.Begin(PORT, 1, chLog, chText)
		if err != nil {
			fmt.Printf("\nERROR:  %v\n", err)
			os.Exit(9)
		}
	}()

	fmt.Println("UI incoming...")

	p := tea.NewProgram(prog)
	final, err := p.Run()
	if err != nil {
		fmt.Printf("ERROR running UI: %s\n\n", err)
	}
	fmt.Println(final.View())

}
