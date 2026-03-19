package main

import (
	"os"

	"github.com/yeasy/mdpress/cmd"
)

// main is the entry point for the mdpress application.
func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
