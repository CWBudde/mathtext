package mathtext

import "sync"

var (
	unknownCmdMu      sync.RWMutex
	unknownCmdHandler func(name string)
)

// SetUnknownCommandHandler installs fn, which is invoked with the name of each
// unrecognized TeX command encountered while parsing (e.g. `\foo` reports
// "foo"). It returns a function that restores the previously installed handler;
// a nil fn disables reporting.
//
// Matplotlib raises on unknown mathtext commands. This port keeps rendering the
// command as literal text (so output is never wholly lost) but surfaces the
// command through this hook so callers can warn — or escalate to an error —
// instead of failing silently.
func SetUnknownCommandHandler(fn func(string)) (restore func()) {
	unknownCmdMu.Lock()
	prev := unknownCmdHandler
	unknownCmdHandler = fn
	unknownCmdMu.Unlock()
	return func() {
		unknownCmdMu.Lock()
		unknownCmdHandler = prev
		unknownCmdMu.Unlock()
	}
}

// reportUnknownCommand notifies the active handler, if any.
func reportUnknownCommand(name string) {
	unknownCmdMu.RLock()
	h := unknownCmdHandler
	unknownCmdMu.RUnlock()
	if h != nil {
		h(name)
	}
}
