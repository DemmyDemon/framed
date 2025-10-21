package main

import (
	"fmt"

	"github.com/DemmyDemon/framed/server"
)

const (
	PORT   = 7100
	WIDTH  = 800
	HEIGHT = 480
)

func main() {
	fmt.Printf("Shall listen on port %d\n", PORT)
	err := server.Begin(PORT, 1)
	if err != nil {
		fmt.Printf("\nERROR:  %v\n", err)
	}
}
