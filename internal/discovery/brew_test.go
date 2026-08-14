package discovery

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// fakeRunner maps "arg1 arg2 ..." to stdout bytes (no real brew calls).
type fakeRunner struct {
	responses map[string][]byte
	errFor    map[string]error
	err       error
}

func (f *fakeRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	key := strings.Join(args, " ")
	if f.errFor != nil {
		if err, ok := f.errFor[key]; ok {
			return nil, err
		}
	}
	out, ok := f.responses[key]
	if !ok {
		return nil, fmt.Errorf("unexpected brew args: %s", key)
	}
	return out, nil
}

func TestFakeRunner_ListFormula(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"list --formula": []byte("git\nfzf\n"),
		},
	}

	out, err := runner.Run(context.Background(), "list", "--formula")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 || lines[0] != "git" || lines[1] != "fzf" {
		t.Fatalf("stdout = %q, want git and fzf", string(out))
	}
}

func TestFakeRunner_UnexpectedArgs(t *testing.T) {
	runner := &fakeRunner{responses: map[string][]byte{}}
	_, err := runner.Run(context.Background(), "list", "--cask")
	if err == nil {
		t.Fatal("expected error for unexpected args")
	}
}

func TestExecRunner_ImplementsRunner(t *testing.T) {
	var _ Runner = (*ExecRunner)(nil)
}
