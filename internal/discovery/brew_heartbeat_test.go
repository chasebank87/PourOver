package discovery

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestActivityWriter_TouchesOnWrite(t *testing.T) {
	var buf bytes.Buffer
	w := newActivityWriter(&buf)
	before := w.lastActivity()
	time.Sleep(2 * time.Millisecond)
	if _, err := w.Write([]byte("hi")); err != nil {
		t.Fatal(err)
	}
	if !w.lastActivity().After(before) {
		t.Fatal("expected lastActivity to advance")
	}
	if buf.String() != "hi" {
		t.Fatalf("buf = %q", buf.String())
	}
}

func TestStartBrewHeartbeat_EmitsWhenSilent(t *testing.T) {
	var mu sync.Mutex
	var buf bytes.Buffer
	w := newActivityWriter(&lockedWriter{mu: &mu, buf: &buf})
	stop := startBrewHeartbeat(w, w, []string{"install", "--cask", "anaconda"}, 20*time.Millisecond)
	defer stop()

	// Stay silent long enough for at least one heartbeat.
	time.Sleep(70 * time.Millisecond)
	stop()

	mu.Lock()
	out := buf.String()
	mu.Unlock()
	if !strings.Contains(out, "still working: brew install --cask anaconda") {
		t.Fatalf("missing heartbeat in %q", out)
	}
}

func TestStartBrewHeartbeat_SkipsWhenActive(t *testing.T) {
	var mu sync.Mutex
	var buf bytes.Buffer
	w := newActivityWriter(&lockedWriter{mu: &mu, buf: &buf})
	stop := startBrewHeartbeat(w, w, []string{"install", "fzf"}, 25*time.Millisecond)
	defer stop()

	deadline := time.Now().Add(80 * time.Millisecond)
	for time.Now().Before(deadline) {
		_, _ = w.Write([]byte("x"))
		time.Sleep(5 * time.Millisecond)
	}
	stop()

	mu.Lock()
	out := buf.String()
	mu.Unlock()
	if strings.Contains(out, "still working") {
		t.Fatalf("unexpected heartbeat while active: %q", out)
	}
}

func TestJoinBrewArgs(t *testing.T) {
	got := joinBrewArgs([]string{"install", "--cask", "anaconda", "extra", "more"})
	if got != "install --cask anaconda extra …" {
		t.Fatalf("got %q", got)
	}
}

type lockedWriter struct {
	mu  *sync.Mutex
	buf *bytes.Buffer
}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}
