package cli

import (
	"os"
	"os/signal"
)

type CloseChannel chan bool

// Handle the interrupt signal
func (closeChannel CloseChannel) handleInterruptSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for range c {
			closeChannel <- true
			break
		}
	}()
}

// Create a new close channel
func NewCloseChannel() CloseChannel {
	var closeChannel CloseChannel = make(chan bool)
	closeChannel.handleInterruptSignal()

	return closeChannel
}

// Check if the close channel is closed
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
	go func() {
		closeChannel <- true
	}()
}
