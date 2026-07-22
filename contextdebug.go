// Package contextdebug lets a service record per-dependency-call debug data
// (name, request, response, error, latency) onto a context.Context, for
// debug purposes only.
package contextdebug

import (
	"context"
	"sync"
)

type ctxKey struct{}

var debugCtxKey = ctxKey{}

// Entry captures the details of a single dependency call.
type Entry struct {
	Name      string      `json:"name"`
	Request   interface{} `json:"req"`
	Response  interface{} `json:"resp"`
	Error     error       `json:"err"`
	LatencyMs int         `json:"lat_ms"`
}

type store struct {
	mu      sync.Mutex
	entries []Entry
}

// New attaches a fresh debug store to ctx and returns the derived context.
// It is the caller's responsibility to decide when debug mode should be
// enabled (e.g. based on a request flag) before calling New.
func New(ctx context.Context) context.Context {
	return context.WithValue(ctx, debugCtxKey, &store{})
}

// Enabled reports whether ctx has an active debug store, i.e. whether New
// was previously called on it (or an ancestor context).
func Enabled(ctx context.Context) bool {
	_, ok := ctx.Value(debugCtxKey).(*store)
	return ok
}

// Collect appends entry to ctx's debug store. If ctx has no debug store
// (debug mode was never enabled via New), Collect is a no-op.
func Collect(ctx context.Context, entry Entry) {
	s, ok := ctx.Value(debugCtxKey).(*store)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entry)
}

// Snapshot returns a copy of all entries collected so far on ctx. It
// returns nil if ctx has no debug store.
func Snapshot(ctx context.Context) []Entry {
	s, ok := ctx.Value(debugCtxKey).(*store)
	if !ok {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}
