package cli

import (
	"os"
	"os/signal"
)

// Define the close channel type.
type CloseChannel chan struct{}

// Function to handle the interrupt signal.
func (closeChannel CloseChannel) handleInterruptSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for range c {
			closeChannel.Close()
		}
	}()
}

// Constructor like function to create a new close channel.
func NewCloseChannel() CloseChannel {
	var closeChannel CloseChannel = make(chan struct{})
	closeChannel.handleInterruptSignal()

	return closeChannel
}

// Function to check if the close channel is closed, unblocking.
func (closeChannel CloseChannel) IsClosed() bool {
	select {
	case <-closeChannel:
		return true
	default:
	}

	return false
}

// Function to close the close channel.
func (closeChannel CloseChannel) Close() {
	close(closeChannel)
}
