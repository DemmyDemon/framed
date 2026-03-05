package main

import (
	"errors"
	"log"
	"os"

	"github.com/DemmyDemon/framed/server"
)

const (
	PORT = 7100
)

func main() {

	filename := "trmnl.txt"
	if len(os.Args) > 1 {
		filename = os.Args[1]
	}
	_, err := os.Stat(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			file, err := os.Create(filename)
			if err != nil {
				log.Fatal("Could not create file", filename, err)
			}
			file.Close() // We don't actually need it now, we just want to make it exist, and verify access.
		} else {
			log.Fatal("Something weird about this file!", filename, err)
		}
	}

	log.Println("Listening on port", PORT)
	err = server.Begin(PORT, filename)
	if err != nil {
		log.Fatal("Serve error", err)
	}
}
