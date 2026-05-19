package config

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// M2.1 spike: embed Lua, execute a return table, read a nested string in Go.
func TestLuaSpike_ReadPolicyFromTable(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	const script = `return { policy = { uninstall_mode = "safe" } }`
	if err := L.DoString(script); err != nil {
		t.Fatalf("DoString: %v", err)
	}

	root := L.Get(-1)
	defer L.Pop(1)

	if root.Type() != lua.LTTable {
		t.Fatalf("expected table, got %s", root.Type())
	}

	policy := L.GetField(root.(*lua.LTable), "policy")
	if policy.Type() != lua.LTTable {
		t.Fatalf("policy: expected table, got %s", policy.Type())
	}

	mode := L.GetField(policy.(*lua.LTable), "uninstall_mode")
	if mode.Type() != lua.LTString {
		t.Fatalf("uninstall_mode: expected string, got %s", mode.Type())
	}
	if got := mode.String(); got != "safe" {
		t.Fatalf("uninstall_mode = %q, want safe", got)
	}
}
