//go:build !unix

package discovery

import "os/exec"

func configureBrewMutationProcess(cmd *exec.Cmd) {}
