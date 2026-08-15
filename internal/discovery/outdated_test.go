package discovery

import (
	"context"
	"testing"
)

func TestDiscoverOutdated(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"outdated --formula -q": []byte("git\nwget\n"),
			"outdated --cask -q":    []byte("warp\n"),
		},
	}
	got, err := DiscoverOutdated(context.Background(), runner)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Formulae) != 2 || got.Formulae[0] != "git" || got.Formulae[1] != "wget" {
		t.Fatalf("formulae = %#v", got.Formulae)
	}
	if len(got.Casks) != 1 || got.Casks[0] != "warp" {
		t.Fatalf("casks = %#v", got.Casks)
	}
}

func TestDiscoverOutdated_Empty(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string][]byte{
			"outdated --formula -q": []byte(""),
			"outdated --cask -q":    []byte(""),
		},
	}
	got, err := DiscoverOutdated(context.Background(), runner)
	if err != nil {
		t.Fatal(err)
	}
	if got.Formulae != nil || got.Casks != nil {
		t.Fatalf("got %#v", got)
	}
}
