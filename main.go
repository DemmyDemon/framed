package main

import (
	"errors"
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

	filename := "trmnl.txt"
	if len(os.Args) > 1 {
		filename = os.Args[1]
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			file, err := os.Create(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not create %s: %s", filename, err)
				os.Exit(1)
			}
			data = []byte{}
			file.Close() // We don't actually need it now, we just want to make it exist, and verify access.
		} else {
			fmt.Fprintf(os.Stderr, "Something weird about %s: %s", filename, err)
			os.Exit(2)
		}
	}
	initialText := string(data)

	chLog, chText, prog := ui.NewUI(initialText, filename)
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
