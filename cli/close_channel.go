package cli

import (
	"os"
	"os/signal"
)

type CloseChannel chan struct{}

// Handle the interrupt signal
func (closeChannel CloseChannel) handleInterruptSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for range c {
			close(closeChannel)
		}
	}()
}

// Create a new close channel
func NewCloseChannel() CloseChannel {
	var closeChannel CloseChannel = make(chan struct{})
	closeChannel.handleInterruptSignal()

	return closeChannel
}

// Non-blocking check on if the close channel is closed
func (closeChannel CloseChannel) IsClosed() bool {
	select {
	case <-closeChannel:
		return true
	default:
	}

	return false
}

// Close the close channel
func (closeChannel CloseChannel) Close() {
	close(closeChannel)
}
