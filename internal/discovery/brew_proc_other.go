//go:build !unix

package discovery

import (
	"io"
	"os/exec"
)

func configureBrewMutationProcess(cmd *exec.Cmd) {}

func attachBrewMutationTTY(cmd *exec.Cmd, stdin io.Reader) func() {
	return func() {}
}
