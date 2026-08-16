package discovery

import (
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

type mapMasDiskProbe struct {
	// name -> adamID; adamID 0 means exists but mdls unknown
	apps map[string]int64
	// names that do not exist even if listed with a sentinel
	missing map[string]bool
}

func (p mapMasDiskProbe) FindApp(name string) (bool, int64) {
	if p.missing[name] {
		return false, 0
	}
	id, ok := p.apps[name]
	if !ok {
		return false, 0
	}
	return true, id
}

func TestMergeDesiredMasOnDisk_EmptyListPlusOnDisk(t *testing.T) {
	t.Parallel()
	state := MasState{}
	desired := []config.MasApp{
		{Name: "Xcode", ID: 497799835},
		{Name: "Boop", ID: 1518425043},
	}
	probe := mapMasDiskProbe{apps: map[string]int64{
		"Xcode": 0, // cold index
		"Boop":  1518425043,
	}}
	got := MergeDesiredMasOnDisk(state, desired, probe)
	if len(got.Apps) != 2 {
		t.Fatalf("Apps = %#v, want 2", got.Apps)
	}
	byID := map[int64]string{}
	for _, a := range got.Apps {
		byID[a.ID] = a.Name
	}
	if byID[497799835] != "Xcode" || byID[1518425043] != "Boop" {
		t.Fatalf("Apps = %#v", got.Apps)
	}
}

func TestMergeDesiredMasOnDisk_AlreadyListedUnchanged(t *testing.T) {
	t.Parallel()
	state := MasState{Apps: []MasInstalled{{ID: 497799835, Name: "Xcode"}}}
	desired := []config.MasApp{{Name: "Xcode", ID: 497799835}}
	probe := mapMasDiskProbe{apps: map[string]int64{"Xcode": 497799835}}
	got := MergeDesiredMasOnDisk(state, desired, probe)
	if len(got.Apps) != 1 || got.Apps[0].ID != 497799835 {
		t.Fatalf("Apps = %#v, want single Xcode", got.Apps)
	}
}

func TestMergeDesiredMasOnDisk_MissingStillAbsent(t *testing.T) {
	t.Parallel()
	state := MasState{}
	desired := []config.MasApp{{Name: "Xcode", ID: 497799835}}
	probe := mapMasDiskProbe{missing: map[string]bool{"Xcode": true}}
	got := MergeDesiredMasOnDisk(state, desired, probe)
	if len(got.Apps) != 0 {
		t.Fatalf("Apps = %#v, want empty", got.Apps)
	}
}

func TestMergeDesiredMasOnDisk_AdamIDMismatchSkipped(t *testing.T) {
	t.Parallel()
	state := MasState{}
	desired := []config.MasApp{{Name: "Copilot", ID: 1447330651}}
	probe := mapMasDiskProbe{apps: map[string]int64{"Copilot": 999}}
	got := MergeDesiredMasOnDisk(state, desired, probe)
	if len(got.Apps) != 0 {
		t.Fatalf("Apps = %#v, want empty on adam ID mismatch", got.Apps)
	}
}

func TestMergeDesiredMasOnDisk_NilProbeNoop(t *testing.T) {
	t.Parallel()
	state := MasState{Apps: []MasInstalled{{ID: 1, Name: "A"}}}
	got := MergeDesiredMasOnDisk(state, []config.MasApp{{Name: "B", ID: 2}}, nil)
	if len(got.Apps) != 1 {
		t.Fatalf("Apps = %#v", got.Apps)
	}
}

func TestMergeDesiredMasOnDisk_DoesNotInventUndeclared(t *testing.T) {
	t.Parallel()
	// Probe would find "Extra" but it is not desired — must not appear.
	state := MasState{}
	desired := []config.MasApp{{Name: "Xcode", ID: 497799835}}
	probe := mapMasDiskProbe{apps: map[string]int64{
		"Xcode": 0,
		"Extra": 123,
	}}
	got := MergeDesiredMasOnDisk(state, desired, probe)
	if len(got.Apps) != 1 || got.Apps[0].Name != "Xcode" {
		t.Fatalf("Apps = %#v, want only Xcode", got.Apps)
	}
}
