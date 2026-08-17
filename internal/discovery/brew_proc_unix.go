//go:build unix

package discovery

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/chasebank87/PourOver/internal/tty"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
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

// attachBrewMutationTTY puts brew in the foreground process group when stdin is
// a terminal. Setpgid alone leaves brew in the background, so sudo's /dev/tty
// read gets SIGTTIN and the cask installer hangs while keystrokes echo in the
// parent. Restores this process group as the foreground after brew exits.
func attachBrewMutationTTY(cmd *exec.Cmd, stdin io.Reader) (restore func()) {
	nop := func() {}
	f, ok := stdin.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return nop
	}
	devTTY, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nop
	}
	fd := int(devTTY.Fd())
	pgrp := unix.Getpgrp()
	signal.Ignore(syscall.SIGTTOU, syscall.SIGTTIN)
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.SysProcAttr.Foreground = true
	cmd.SysProcAttr.Ctty = fd
	tty.EnableEchoControl()
	return func() {
		_ = unix.IoctlSetPointerInt(fd, unix.TIOCSPGRP, pgrp)
		_ = devTTY.Close()
	}
}
