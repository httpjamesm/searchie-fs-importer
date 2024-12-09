package main

import (
	"log"

	"github.com/httpjamesm/searchie-fs-importer/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
