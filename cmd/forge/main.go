package main

import (
	"fmt"
	"os"

	"github.com/glieske/forge/internal/app"
)

var (
	version = "0.1.0-dev"
	commit  = "unknown"
)

func main() {
	if err := app.Execute(version, commit); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
