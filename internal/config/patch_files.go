package config

import (
	"fmt"
	"os"
	"strings"
)

// PatchFilesLinksFile replaces files.links in pourover.lua and validates the result.
func PatchFilesLinksFile(path string, links []FileLink) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out, err := PatchFilesLinks(string(src), links)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		return err
	}
	if _, err := LoadManifest(path); err != nil {
		return fmt.Errorf("patched config invalid: %w", err)
	}
	return nil
}

// PatchFilesLinks surgically replaces the files.links array in Lua source text.
// Other files.* keys (managed, templates, unlink) and the rest of the root table
// are preserved. Creates files.links if missing.
func PatchFilesLinks(src string, links []FileLink) (string, error) {
	out, err := ensureNestedTable(src, []string{"files", "links"})
	if err != nil {
		return "", err
	}
	start, end, err := locateTable(out, []string{"files", "links"})
	if err != nil {
		return "", err
	}
	body := formatFilesLinksBody(links)
	return out[:start] + body + out[end:], nil
}

func formatFilesLinksBody(links []FileLink) string {
	if len(links) == 0 {
		return "\n    "
	}
	var b strings.Builder
	for _, link := range links {
		fmt.Fprintf(&b, "\n      { source = %q, target = %q },", link.Source, link.Target)
	}
	b.WriteString("\n    ")
	return b.String()
}
