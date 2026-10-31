package dns

import (
	"context"
	"testing"
)

func TestListenersWantUnchanged(t *testing.T) {
	noop := func() {}
	want := map[int32]listenerWant{
		1: {bind: "0.0.0.0:53", paused: false},
		2: {bind: "0.0.0.0:54", paused: true},
	}
	listeners := map[int32]context.CancelFunc{1: noop}
	binds := map[int32]string{1: "0.0.0.0:53"}
	if !listenersWantUnchanged(want, listeners, binds) {
		t.Fatal("expected unchanged when active binds match and paused has no listener")
	}
	binds[1] = "127.0.0.1:53"
	if listenersWantUnchanged(want, listeners, binds) {
		t.Fatal("expected changed when bind differs")
	}
}
