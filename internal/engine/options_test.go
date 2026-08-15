package engine

import "testing"

func TestApplyOptions_Defaults(t *testing.T) {
	opts := ApplyOptions{}
	if opts.AutoYes {
		t.Fatal("AutoYes should default false")
	}
}
