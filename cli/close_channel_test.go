package cli

import (
	"context"
	"testing"
)

func TestCloseChannelIsIdempotent(t *testing.T) {
	shutdown := NewCloseChannel(context.Background())
	if shutdown.IsClosed() {
		t.Fatal("new shutdown signal is already closed")
	}
	shutdown.Close()
	shutdown.Close()
	if !shutdown.IsClosed() {
		t.Fatal("shutdown signal remains open")
	}
}

func TestCloseChannelFollowsParentLifecycle(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	shutdown := NewCloseChannel(parent)
	cancel()
	<-shutdown.Done()
	if !shutdown.IsClosed() {
		t.Fatal("parent cancellation did not stop the controller")
	}
}
