package main

import (
	"errors"
	"log"

	"github.com/dyammarcano/presencial/internal/program"
)

func main() {
	app, err := program.NewMainApp("presencial")
	if err != nil {
		log.Fatal(errors.New("erro ao criar app"))
	}

	app.RunApp()
}
