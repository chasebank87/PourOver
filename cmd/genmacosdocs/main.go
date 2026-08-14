package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chasebank87/PourOver/internal/config"
)

func main() {
	out := "docs/macos-defaults.md"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	md, err := config.RenderMacOSDefaultsMarkdown()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, []byte(md), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d keys)\n", out, config.CatalogKeyCount())
}
