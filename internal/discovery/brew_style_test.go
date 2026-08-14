package discovery

import (
	"bytes"
	"strings"
	"testing"
)

func TestStyleBrewLine_RestylesMarkers(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"==> Installing Cask arc", "☕ Installing Cask arc"},
		{"==> Moving App 'Arc.app' to '/Applications/Arc.app'", "☕ Moving App 'Arc.app' to '/Applications/Arc.app'"},
		{"🍺  arc was successfully installed!", "☕  arc was successfully installed!"},
		{"==> Pouring fzf--0.1.0.arm64_sonoma.bottle.tar.gz", "☕ Pouring fzf--0.1.0.arm64_sonoma.bottle.tar.gz"},
		{"Already downloaded: /tmp/foo", ""},
		{"==> Running `brew cleanup fzf`...", ""},
		{"", ""},
		{"  ", ""},
	}
	for _, tc := range cases {
		got := StyleBrewLine(tc.in)
		if got != tc.want {
			t.Errorf("StyleBrewLine(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestBrewStyleWriter_StreamsLines(t *testing.T) {
	var out bytes.Buffer
	w := NewBrewStyleWriter(&out)
	input := "==> Installing Cask arc\n🍺  arc was successfully installed!\nAlready downloaded: x\n"
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	wantLines := []string{
		"☕ Installing Cask arc",
		"☕  arc was successfully installed!",
	}
	for _, want := range wantLines {
		if !strings.Contains(got, want) {
			t.Fatalf("output %q missing %q", got, want)
		}
	}
	if strings.Contains(got, "Already downloaded") {
		t.Fatalf("noise leaked: %q", got)
	}
}

func TestBrewStyleWriter_FlushPartial(t *testing.T) {
	var out bytes.Buffer
	w := NewBrewStyleWriter(&out)
	if _, err := w.Write([]byte("==> Downloading https://example.com/pkg")); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected buffered, got %q", out.String())
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != "☕ Downloading https://example.com/pkg" {
		t.Fatalf("got %q", got)
	}
}
