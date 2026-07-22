// Package health tracks whether the live pipeline's ingestion loop is
// actively making progress, so /healthz can report more than just "the
// database is reachable" — it can distinguish a live process that's stuck
// (RPC hanging, deadlocked loop, etc.) from one that's genuinely healthy.
package health

import (
	"sync/atomic"
	"time"
)

var lastTickUnix atomic.Int64

// Touch records that the ingestion loop just completed a poll cycle
// (regardless of whether new ledgers were found), proving the loop is alive
// and its RPC/DB dependencies are responsive.
func Touch() {
	lastTickUnix.Store(time.Now().Unix())
}

// Stale reports whether the pipeline hasn't ticked within maxAge. Before the
// first Touch (e.g. right after process start), it reports false so
// readiness reflects DB-only health until the pipeline has had a chance to
// run at least once.
func Stale(maxAge time.Duration) bool {
	last := lastTickUnix.Load()
	if last == 0 {
		return false
	}
	return time.Since(time.Unix(last, 0)) > maxAge
}
