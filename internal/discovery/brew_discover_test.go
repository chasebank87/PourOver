package discovery

import (
	"context"
	"fmt"
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

func TestParseBrewList_MultiColumn(t *testing.T) {
	// Remote/TTY brew list --cask prints a grid; newline-only parsing used to
	// treat each row as one name and miss tokens like omnissa-horizon-client.
	raw := []byte("anaconda\t\tepic-games\t\tkitty\t\tsigmaos\n" +
		"devin-desktop\t\tjump-desktop-connect\t\tpika\t\tvlc\n" +
		"omnissa-horizon-client\t\ttransmission\n")
	got := parseBrewList(raw)
	want := []string{
		"anaconda", "epic-games", "kitty", "sigmaos",
		"devin-desktop", "jump-desktop-connect", "pika", "vlc",
		"omnissa-horizon-client", "transmission",
	}
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
			"tap":                                   readFixture(t, "list-tap.txt"),
			"trust --json=v1":                       []byte(`{"taps":["nikitabobko/tap"],"formulae":[],"casks":[],"commands":[]}`),
			"list --formula -1":                     readFixture(t, "list-formula.txt"),
			"list --cask -1":                        readFixture(t, "list-cask.txt"),
			"list --formula --installed-on-request": []byte("git\nfzf\n"),
		},
	}

	state, err := DiscoverBrew(context.Background(), runner)
	if err != nil {
		t.Fatalf("DiscoverBrew: %v", err)
	}
	if len(state.Taps) != 4 || state.Taps[0] != "homebrew/core" {
		t.Errorf("taps = %#v", state.Taps)
	}
	if len(state.TrustedTaps) != 1 || state.TrustedTaps[0] != "nikitabobko/tap" {
		t.Errorf("trusted taps = %#v", state.TrustedTaps)
	}
	if len(state.Formulae) != 3 || state.Formulae[0] != "git" {
		t.Errorf("formulae = %#v", state.Formulae)
	}
	if len(state.Casks) != 2 || state.Casks[1] != "1password" {
		t.Errorf("casks = %#v", state.Casks)
	}
	if len(state.FormulaeRequested) != 2 || state.FormulaeRequested[0] != "git" {
		t.Errorf("formulae requested = %#v", state.FormulaeRequested)
	}
	if got := DeclarableTaps(state.Taps); len(got) != 2 || got[0] != "homebrew/cask-fonts" {
		t.Errorf("DeclarableTaps = %#v", got)
	}
	if got := state.RemovableTaps(); len(got) != 2 {
		t.Errorf("RemovableTaps = %#v", got)
	}
}

func TestDiscoverBrew_EmptyLists(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"tap":                                   readFixture(t, "list-formula-empty.txt"),
			"list --formula -1":                     readFixture(t, "list-formula-empty.txt"),
			"list --cask -1":                        readFixture(t, "list-formula-empty.txt"),
			"list --formula --installed-on-request": readFixture(t, "list-formula-empty.txt"),
		},
	}

	state, err := DiscoverBrew(context.Background(), runner)
	if err != nil {
		t.Fatalf("DiscoverBrew: %v", err)
	}
	if state.Taps != nil || state.Formulae != nil || state.Casks != nil {
		t.Fatalf("expected nil taps/formulae/casks, got taps=%#v formulae=%#v casks=%#v", state.Taps, state.Formulae, state.Casks)
	}
	if state.FormulaeRequested == nil {
		t.Fatal("FormulaeRequested should be non-nil empty after discovery")
	}
	if len(state.FormulaeRequested) != 0 {
		t.Fatalf("FormulaeRequested = %#v, want empty", state.FormulaeRequested)
	}
}

func TestDiscoverBrew_OnRequestFailsUsesLeaves(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"tap":               []byte("homebrew/core\n"),
			"list --formula -1": []byte("git\ngettext\npcre2\n"),
			"list --cask -1":    []byte("raycast\n"),
			"leaves":            []byte("git\n"),
		},
		errFor: map[string]error{
			"list --formula --installed-on-request": fmt.Errorf("api unavailable"),
		},
	}
	state, err := DiscoverBrew(context.Background(), runner)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.FormulaeRequested) != 1 || state.FormulaeRequested[0] != "git" {
		t.Fatalf("FormulaeRequested = %#v, want [git] from leaves", state.FormulaeRequested)
	}
}

func TestDiscoverBrew_OnRequestAndLeavesFailSkipsFormulaRemoves(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"tap":               []byte(""),
			"list --formula -1": []byte("git\ngettext\n"),
			"list --cask -1":    []byte(""),
		},
		errFor: map[string]error{
			"list --formula --installed-on-request": fmt.Errorf("api unavailable"),
			"leaves":                                fmt.Errorf("api unavailable"),
		},
	}
	state, err := DiscoverBrew(context.Background(), runner)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.FormulaeRequested) != 0 {
		t.Fatalf("FormulaeRequested = %#v, want empty (no dep uninstalls)", state.FormulaeRequested)
	}
}

func TestIsCoreTap(t *testing.T) {
	if !IsCoreTap("homebrew/core") || !IsCoreTap("Homebrew/Cask") {
		t.Fatal("expected core taps")
	}
	if IsCoreTap("homebrew/cask-fonts") || IsCoreTap("nikitabobko/tap") {
		t.Fatal("expected non-core")
	}
}

func TestNeedsExplicitTrust(t *testing.T) {
	if NeedsExplicitTrust("homebrew/cask-fonts") || NeedsExplicitTrust("homebrew/core") {
		t.Fatal("official taps should not need explicit trust")
	}
	if !NeedsExplicitTrust("nikitabobko/tap") {
		t.Fatal("third-party taps need explicit trust")
	}
}

func TestParseTrustTapsJSON(t *testing.T) {
	got := parseTrustTapsJSON([]byte(`{"taps":["a/b","c/d"],"formulae":[]}`))
	if len(got) != 2 || got[0] != "a/b" {
		t.Fatalf("got %#v", got)
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
