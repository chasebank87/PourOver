package discovery

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultBrewHeartbeatInterval is how often to emit a "still working" line when
// a streamed brew mutation has produced no output.
const DefaultBrewHeartbeatInterval = 60 * time.Second

// activityWriter records the last time real brew output was written.
// Heartbeat lines must not call touchOutput (idle timeout uses lastOutput only).
type activityWriter struct {
	out             io.Writer
	lastOutputUnix  atomic.Int64 // unix nano; 0 means never
	heartbeatPaused atomic.Bool
	idlePaused      atomic.Bool
}

func newActivityWriter(out io.Writer) *activityWriter {
	w := &activityWriter{out: out}
	w.touchOutput()
	return w
}

func (w *activityWriter) Write(p []byte) (int, error) {
	w.noteBrewBytes(p)
	if w.out == nil {
		return len(p), nil
	}
	return w.out.Write(p)
}

func (w *activityWriter) noteBrewBytes(p []byte) {
	text := string(p)
	if looksLikeSilentBrewWork(text) {
		// Sudo/installer/pour often go silent for a long time; do not idle-kill.
		w.idlePaused.Store(true)
	}
	if looksLikeAuthPrompt(text) {
		w.heartbeatPaused.Store(true)
	} else if len(bytes.TrimSpace(p)) > 0 {
		w.heartbeatPaused.Store(false)
		if !looksLikeSilentBrewWork(text) {
			w.idlePaused.Store(false)
		}
	}
	w.touchOutput()
}

func (w *activityWriter) Flush() error {
	// Flush alone is not brew output; do not reset idle.
	if f, ok := w.out.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

func (w *activityWriter) touchOutput() {
	w.lastOutputUnix.Store(time.Now().UnixNano())
}

func (w *activityWriter) lastOutput() time.Time {
	n := w.lastOutputUnix.Load()
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(0, n)
}

// lastActivity is an alias for lastOutput (heartbeat silence detection).
func (w *activityWriter) lastActivity() time.Time {
	return w.lastOutput()
}

// startBrewHeartbeat emits periodic "still working" lines while brew is silent.
// Heartbeat writes do not reset lastOutput (idle timeout stays honest).
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
				if time.Since(activity.lastOutput()) < interval {
					continue
				}
				msg := fmt.Sprintf("☕ still working: %s (no new output)\n", label)
				_, _ = io.WriteString(out, msg)
				// Do not touchOutput — idle killer must see real silence.
			}
		}
	}()
	return func() {
		once.Do(func() { close(done) })
	}
}

// startBrewIdleCancel cancels the brew mutation context when there is no
// brew output for idle duration. Idle <= 0 disables. Stop is safe to call
// multiple times.
func startBrewIdleCancel(cancel context.CancelFunc, activity *activityWriter, idle time.Duration, fired *atomic.Bool) (stop func()) {
	if idle <= 0 || cancel == nil || activity == nil {
		return func() {}
	}
	done := make(chan struct{})
	var once sync.Once
	go func() {
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				if activity.idlePaused.Load() || activity.heartbeatPaused.Load() {
					continue
				}
				if time.Since(activity.lastOutput()) < idle {
					continue
				}
				if fired != nil {
					fired.Store(true)
				}
				cancel()
				return
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
