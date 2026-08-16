//go:build unix

package discovery

import (
	"os/exec"
	"syscall"
)

// configureBrewMutationProcess puts brew in its own process group so idle/absolute
// cancel kills the whole tree (Homebrew spawns curl, installer helpers, etc.).
func configureBrewMutationProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		// Negative PID = process group.
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
