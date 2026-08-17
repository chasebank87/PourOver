package tty

import (
	"sync"
	"sync/atomic"
)

var (
	echoControl atomic.Bool
	echoMu      sync.Mutex
	echoRestore func()
)

// EnableEchoControl allows EchoOff to change /dev/tty. ExecRunner turns this
// on for interactive brew mutations so tests do not mute the developer terminal.
func EnableEchoControl() {
	echoControl.Store(true)
}

// EchoOff disables local echo on /dev/tty so a sudo password typed at a
// captured prompt is not printed. No-op unless EnableEchoControl was called,
// or if echo is already off. EchoOn restores the previous mode.
func EchoOff() {
	if !echoControl.Load() {
		return
	}
	echoMu.Lock()
	defer echoMu.Unlock()
	if echoRestore != nil {
		return
	}
	restore, err := disableEcho()
	if err != nil {
		return
	}
	echoRestore = restore
}

// EchoOn restores terminal echo after EchoOff. Safe to call when echo was not
// disabled.
func EchoOn() {
	echoMu.Lock()
	defer echoMu.Unlock()
	if echoRestore == nil {
		return
	}
	echoRestore()
	echoRestore = nil
}
