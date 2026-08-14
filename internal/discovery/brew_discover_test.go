package discovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseBrewList_Empty(t *testing.T) {
	got := parseBrewList(readFixture(t, "list-formula-empty.txt"))
	if got != nil {
		t.Fatalf("parseBrewList(empty) = %#v, want nil", got)
	}
}

func TestParseBrewList_MultipleLines(t *testing.T) {
	got := parseBrewList(readFixture(t, "list-formula.txt"))
	want := []string{"git", "fzf", "jq"}
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	}
}

func TestDiscoverBrew_FromFixtures(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"list --formula": readFixture(t, "list-formula.txt"),
			"list --cask":    readFixture(t, "list-cask.txt"),
		},
	}

	state, err := DiscoverBrew(context.Background(), runner)
	if err != nil {
		t.Fatalf("DiscoverBrew: %v", err)
	}
	if len(state.Formulae) != 3 || state.Formulae[0] != "git" {
		t.Errorf("formulae = %#v", state.Formulae)
	}
	if len(state.Casks) != 2 || state.Casks[1] != "1password" {
		t.Errorf("casks = %#v", state.Casks)
	}
}

func TestDiscoverBrew_EmptyLists(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"list --formula": readFixture(t, "list-formula-empty.txt"),
			"list --cask":    readFixture(t, "list-formula-empty.txt"),
		},
	}

	state, err := DiscoverBrew(context.Background(), runner)
	if err != nil {
		t.Fatalf("DiscoverBrew: %v", err)
	}
	if state.Formulae != nil || state.Casks != nil {
		t.Fatalf("expected nil slices, got formulae=%#v casks=%#v", state.Formulae, state.Casks)
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "test", "fixtures", "brew", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}
