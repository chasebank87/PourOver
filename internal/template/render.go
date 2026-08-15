// Package template provides sandboxed text/template rendering for files.templates.
// Templates receive a fixed Context only — no custom FuncMap and no arbitrary env lookup.
package template

import (
	"bytes"
	"fmt"
	"os"
	osuser "os/user"
	texttemplate "text/template"
)

// Context is the fixed data available to files.templates sources.
// Env is reserved for a future allowlist; DefaultContext leaves it empty.
type Context struct {
	Hostname string
	User     string
	Home     string
	Env      map[string]string // only allowlisted keys (empty in V2)
}

// DefaultContext builds Context from the current host and user.
// Env is an empty map (no os.Environ passthrough).
func DefaultContext() (Context, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return Context{}, fmt.Errorf("hostname: %w", err)
	}
	u, err := osuser.Current()
	if err != nil {
		return Context{}, fmt.Errorf("user: %w", err)
	}
	home := u.HomeDir
	if home == "" {
		home, err = os.UserHomeDir()
		if err != nil {
			return Context{}, fmt.Errorf("home: %w", err)
		}
	}
	return Context{
		Hostname: hostname,
		User:     u.Username,
		Home:     home,
		Env:      map[string]string{},
	}, nil
}

// Render executes src as a text/template with ctx.
// Uses default {{ }} delims, missingkey=error, and no custom FuncMap
// (built-in string helpers only — nothing that can exec a process).
func Render(src string, ctx Context) (string, error) {
	if ctx.Env == nil {
		ctx.Env = map[string]string{}
	}
	tmpl, err := texttemplate.New("pourover").Option("missingkey=error").Parse(src)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}
