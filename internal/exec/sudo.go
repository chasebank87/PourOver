package exec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/chasebank87/PourOver/internal/tty"
)

// DefaultSudoTimeout is the max wait for an interactive sudo -v / sudo prompt.
const DefaultSudoTimeout = 5 * time.Minute

// ensureSudo is sudoValidate in production; tests may stub it.
var ensureSudo = sudoValidate

// EnsureSudo caches admin credentials once via `sudo -v` so later elevated
// brew/mas/defaults/PAM calls in the same apply do not re-prompt.
// beforeAuth parks fancy UI before the Password: prompt on /dev/tty.
func EnsureSudo(ctx context.Context, beforeAuth func()) error {
	return ensureSudo(ctx, beforeAuth)
}

func sudoValidate(ctx context.Context, beforeAuth func()) error {
	if beforeAuth != nil {
		beforeAuth()
	} else {
		tty.SyncPromptLine()
	}
	tty.SyncPromptLine()

	ctx, cancel := context.WithTimeout(ctx, DefaultSudoTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sudo", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo -v (admin access): %w", err)
	}
	return nil
}
