package main

import (
	"os"

	"github.com/sprite-ai/agrev/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
