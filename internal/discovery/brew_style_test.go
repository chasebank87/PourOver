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
		{"🍺  arc was successfully installed!", "☕ arc was successfully installed!"},
		{"✔︎ Cask antinote (1.1.7)", "☕ Cask antinote (1.1.7)"},
		{"✔︎ Bottle zoxide (0.10.0)", "☕ Bottle zoxide (0.10.0)"},
		{"          🍺  /opt/homebrew/Cellar/zig/0.16.0_1: 19 files, 1MB", "☕ /opt/homebrew/Cellar/zig/0.16.0_1: 19 files, 1MB"},
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
		"☕ arc was successfully installed!",
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

func TestSummarizeBrewStderr_DropsPaddedCheckmarks(t *testing.T) {
	raw := "==> Fetching downloads for: anaconda, antinote\n" +
		"✔︎ Cask antinote (1.1.7)" + strings.Repeat(" ", 80) + "\n" +
		"Error: installer failed\n"
	got := summarizeBrewStderr(raw)
	if strings.Contains(got, "✔") || strings.Contains(got, "antinote (1.1.7)") {
		t.Fatalf("checkmark dump leaked: %q", got)
	}
	if !strings.Contains(got, "Error: installer failed") {
		t.Fatalf("missing error: %q", got)
	}
}

func TestLooksLikeSilentBrewWork(t *testing.T) {
	if !looksLikeSilentBrewWork("==> Running installer script 'Anaconda3.sh'") {
		t.Fatal("installer script should pause idle")
	}
	if !looksLikeSilentBrewWork("Running installer for dotnet-sdk with `sudo` (which may request your password)...") {
		t.Fatal("sudo installer should pause idle")
	}
	if looksLikeSilentBrewWork("==> Installing Cask ghostty") {
		t.Fatal("plain install line is not silent work")
	}
}

func TestBrewStyleWriter_DoesNotForwardPasswordPrompt(t *testing.T) {
	var out bytes.Buffer
	w := NewBrewStyleWriter(&out)
	if _, err := w.Write([]byte("Password:")); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "Password:") {
		t.Fatalf("must not reprint Password: as a typable line: %q", got)
	}
	if !strings.Contains(got, "authentication required") {
		t.Fatalf("want auth hint, got %q", got)
	}
}

func TestBrewStyleWriter_CRPaddingDoesNotIndentNextLine(t *testing.T) {
	var out bytes.Buffer
	w := NewBrewStyleWriter(&out)
	input := "✔︎ Cask foo (1.0)" + strings.Repeat(" ", 60) + "\r" +
		strings.Repeat(" ", 60) + "\r" +
		"==> Running installer for foo with `sudo` (which may request your password)...\n"
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "          ☕ Running") {
		t.Fatalf("installer line indented by CR padding: %q", got)
	}
	if !strings.Contains(got, "☕ Running installer for foo") {
		t.Fatalf("missing installer line: %q", got)
	}
}
