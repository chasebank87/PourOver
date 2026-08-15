package discovery

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultBrewHeartbeatInterval is how often to emit a "still working" line when
// a streamed brew mutation has produced no output.
const DefaultBrewHeartbeatInterval = 15 * time.Second

// activityWriter records the last time Write was called.
type activityWriter struct {
	out             io.Writer
	lastUnix        atomic.Int64 // unix nano; 0 means never
	heartbeatPaused atomic.Bool
}

func newActivityWriter(out io.Writer) *activityWriter {
	w := &activityWriter{out: out}
	w.touch()
	return w
}

func (w *activityWriter) Write(p []byte) (int, error) {
	if looksLikeAuthPrompt(string(p)) {
		w.heartbeatPaused.Store(true)
	} else if len(bytes.TrimSpace(p)) > 0 {
		// Resume after password once brew prints more output.
		w.heartbeatPaused.Store(false)
	}
	w.touch()
	if w.out == nil {
		return len(p), nil
	}
	return w.out.Write(p)
}

func (w *activityWriter) Flush() error {
	w.touch()
	if f, ok := w.out.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

func (w *activityWriter) touch() {
	w.lastUnix.Store(time.Now().UnixNano())
}

func (w *activityWriter) lastActivity() time.Time {
	n := w.lastUnix.Load()
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(0, n)
}

// startBrewHeartbeat emits periodic "still working" lines while brew is silent.
// The returned stop func is safe to call multiple times.
func startBrewHeartbeat(out io.Writer, activity *activityWriter, args []string, interval time.Duration) (stop func()) {
	if out == nil || activity == nil {
		return func() {}
	}
	if interval <= 0 {
		interval = DefaultBrewHeartbeatInterval
	}
	label := "brew " + joinBrewArgs(args)
	done := make(chan struct{})
	var once sync.Once
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				if activity.heartbeatPaused.Load() {
					continue
				}
				if time.Since(activity.lastActivity()) < interval {
					continue
				}
				msg := fmt.Sprintf("☕ still working: %s (no new output)\n", label)
				_, _ = io.WriteString(out, msg)
				activity.touch()
			}
		}
	}()
	return func() {
		once.Do(func() { close(done) })
	}
}

func joinBrewArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	// Keep heartbeat short: "install --cask anaconda"
	n := len(args)
	if n > 4 {
		n = 4
	}
	out := args[0]
	for i := 1; i < n; i++ {
		out += " " + args[i]
	}
	if len(args) > 4 {
		out += " …"
	}
	return out
}
