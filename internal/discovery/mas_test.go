package discovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func readMasFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestParseMasList_FromFixture(t *testing.T) {
	got := parseMasList(readMasFixture(t, "mas-list.txt"))
	want := []MasInstalled{
		{ID: 497799835, Name: "Xcode"},
		{ID: 1569813296, Name: "1Password for Safari"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestParseMasList_Empty(t *testing.T) {
	if got := parseMasList([]byte("")); got != nil {
		t.Fatalf("parseMasList(empty) = %#v, want nil", got)
	}
	if got := parseMasList([]byte("\n\n  \n")); got != nil {
		t.Fatalf("parseMasList(blank) = %#v, want nil", got)
	}
}

func TestParseMasList_SkipsNonIDLines(t *testing.T) {
	raw := []byte("not-an-id Something\n497799835 Xcode\n\nbogus\n")
	got := parseMasList(raw)
	if len(got) != 1 || got[0].ID != 497799835 || got[0].Name != "Xcode" {
		t.Fatalf("got %#v", got)
	}
}

func TestParseMasOutdated_FromFixture(t *testing.T) {
	got := parseMasOutdated(readMasFixture(t, "mas-outdated.txt"))
	want := []int64{497799835, 640199958}
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestParseMasOutdated_Empty(t *testing.T) {
	if got := parseMasOutdated([]byte("")); got != nil {
		t.Fatalf("parseMasOutdated(empty) = %#v, want nil", got)
	}
}

func TestParseMasOutdated_Resilient(t *testing.T) {
	raw := []byte("\nwarning: something\n497799835 Xcode (15.4 -> 16.0)\nnot-a-number App\n640199958 Developer\n")
	got := parseMasOutdated(raw)
	want := []int64{497799835, 640199958}
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestDiscoverMas_FromFixtures(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"list": readMasFixture(t, "mas-list.txt"),
		},
	}
	state, err := DiscoverMas(context.Background(), runner)
	if err != nil {
		t.Fatalf("DiscoverMas: %v", err)
	}
	if len(state.Apps) != 2 {
		t.Fatalf("Apps = %#v, want 2", state.Apps)
	}
	if state.Apps[0].ID != 497799835 || state.Apps[0].Name != "Xcode" {
		t.Errorf("Apps[0] = %#v", state.Apps[0])
	}
	if state.Apps[1].ID != 1569813296 || state.Apps[1].Name != "1Password for Safari" {
		t.Errorf("Apps[1] = %#v", state.Apps[1])
	}
	if state.Outdated != nil {
		t.Fatalf("Outdated = %#v, want nil (not discovered)", state.Outdated)
	}
}

func TestDiscoverMas_Empty(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"list": []byte(""),
		},
	}
	state, err := DiscoverMas(context.Background(), runner)
	if err != nil {
		t.Fatalf("DiscoverMas: %v", err)
	}
	if state.Apps != nil {
		t.Fatalf("Apps = %#v, want nil", state.Apps)
	}
	if state.Outdated != nil {
		t.Fatalf("Outdated = %#v, want nil", state.Outdated)
	}
}

func TestDiscoverMasOutdated_FromFixtures(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"outdated": readMasFixture(t, "mas-outdated.txt"),
		},
	}
	got, err := DiscoverMasOutdated(context.Background(), runner)
	if err != nil {
		t.Fatalf("DiscoverMasOutdated: %v", err)
	}
	want := []int64{497799835, 640199958}
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestDiscoverMasOutdated_Empty(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"outdated": []byte(""),
		},
	}
	got, err := DiscoverMasOutdated(context.Background(), runner)
	if err != nil {
		t.Fatalf("DiscoverMasOutdated: %v", err)
	}
	if got != nil {
		t.Fatalf("got %#v, want nil", got)
	}
}

func TestExecMasRunner_ImplementsMasRunner(t *testing.T) {
	var _ MasRunner = (*ExecMasRunner)(nil)
}
