package cli

import "context"

// CloseChannel owns one idempotent shutdown signal derived from a caller-owned
// lifecycle context.
type CloseChannel struct {
	context context.Context
	stop    context.CancelFunc
}

// NewCloseChannel derives a shutdown signal without registering process-wide
// handlers. The composition root owns signal registration and cleanup.
func NewCloseChannel(parent context.Context) CloseChannel {
	ctx, stop := context.WithCancel(parent)
	return CloseChannel{context: ctx, stop: stop}
}

// Done is closed when shutdown is requested.
func (closeChannel CloseChannel) Done() <-chan struct{} { return closeChannel.context.Done() }

// IsClosed checks whether shutdown has been requested.
func (closeChannel CloseChannel) IsClosed() bool {
	select {
	case <-closeChannel.Done():
		return true
	default:
		return false
	}
}

// Close requests shutdown. It is safe to call more than once.
func (closeChannel CloseChannel) Close() { closeChannel.stop() }
