package commands

import (
	"testing"
)

func TestNewInitCmd_HasForceFlag(t *testing.T) {
	cmd := NewInitCmd()
	if cmd.Flags().Lookup("force") == nil {
		t.Fatal("missing --force flag")
	}
}
