package main

import (
	"os"

	"github.com/aezell/agrev/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
