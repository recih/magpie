package agent

import (
	"sync"
	"testing"
	"time"
)

// A previous drift can still have a probe out when the next test starts.
// Replacing the map then makes that probe's eventual write panic.
func TestFakeReachWaitsForEarlierProbe(t *testing.T) {
	reachForget()
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unhold := func() { once.Do(func() { close(release) }) }
	defer unhold()
	probed(t.Name(), func() bool { close(started); <-release; return true })
	<-started
	reach.Lock()
	pending := reach.m
	reach.Unlock()
	ready := make(chan struct{})
	go func() { fakeReach(t, map[string]bool{}); close(ready) }()
	select {
	case <-ready:
		// Restore the pending entry before releasing its callback, so a
		// regression reports the ordering failure without leaking a probe.
		reach.Lock()
		reach.m = pending
		reach.Unlock()
		unhold()
		reachForget()
		t.Fatal("fixture reset its cache before the earlier probe finished")
	case <-time.After(50 * time.Millisecond):
	}
	unhold()
	select {
	case <-ready:
	case <-time.After(6 * time.Second):
		t.Fatal("fixture did not finish after the earlier probe")
	}
	reach.Lock()
	n := len(reach.m)
	reach.Unlock()
	if n != 0 {
		t.Fatalf("new fixture kept %d answers from the previous probe", n)
	}
}
