package main

import (
	"fmt"
	"os"

	"github.com/KyleStank/kstankcli/internal/cli"
)

func main() {
	if err := cli.NewRootCommand(cli.DefaultOptions()).Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
