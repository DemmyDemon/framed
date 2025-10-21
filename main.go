package main

import (
	"fmt"
	"os"

	"github.com/DemmyDemon/framed/server"
	"github.com/DemmyDemon/framed/ui"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	PORT   = 7100
	WIDTH  = 800
	HEIGHT = 480
)

func main() {
	fmt.Printf("Shall listen on port %d\n", PORT)

	chLog, _, prog := ui.NewUI()

	go func() {
		err := server.Begin(PORT, 1, chLog)
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
