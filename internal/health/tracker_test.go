package health

import (
	"testing"
	"time"
)

func TestStaleBeforeFirstTouch(t *testing.T) {
	lastTickUnix.Store(0)

	if Stale(time.Minute) {
		t.Error("expected Stale to be false before the first Touch")
	}
}

func TestStaleAfterRecentTouch(t *testing.T) {
	Touch()

	if Stale(time.Minute) {
		t.Error("expected Stale to be false immediately after Touch")
	}
}

func TestStaleAfterOldTouch(t *testing.T) {
	lastTickUnix.Store(time.Now().Add(-time.Hour).Unix())

	if !Stale(time.Minute) {
		t.Error("expected Stale to be true when the last tick is older than maxAge")
	}
}
