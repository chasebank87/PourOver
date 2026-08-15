package engine

import (
	"context"
	"testing"

	"github.com/chasebank87/PourOver/internal/plan"
)

func TestApply_NoChanges(t *testing.T) {
	result, err := Apply(context.Background(), &stubBrewRunner{}, plan.Plan{}, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.Taps != 0 || result.Formulae != 0 || result.Casks != 0 ||
		result.Removed != 0 || result.Defaults != 0 || result.Linked != 0 ||
		result.Renames != 0 || result.Skipped != 0 || result.Failures != 0 {
		t.Fatalf("Apply empty plan = %+v, want zero counts", result)
	}
}
