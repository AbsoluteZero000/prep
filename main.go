package main

import (
	"os"

	"github.com/absolutezero000/prep/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
