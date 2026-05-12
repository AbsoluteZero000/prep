package ui

import (
	"fmt"
	"os"
	"time"
)

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner shows an animated spinner for blocking operations.
type Spinner struct {
	label  string
	stopCh chan struct{}
	doneCh chan struct{}
	failed bool
	errMsg string
}

// NewSpinner creates a new spinner with the given label.
func NewSpinner(label string) *Spinner {
	return &Spinner{
		label:  label,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	if !IsTTY() {
		fmt.Printf("%s...\n", s.label)
		return
	}

	go func() {
		defer close(s.doneCh)
		i := 0
		for {
			select {
			case <-s.stopCh:
				return
			default:
				frame := spinFrames[i%len(spinFrames)]
				fmt.Printf("\r%s %s", frame, s.label)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

// Stop stops the spinner and shows a success checkmark.
func (s *Spinner) Stop() {
	close(s.stopCh)
	<-s.doneCh

	if s.failed {
		fmt.Printf("\r✗ %s: %s\n", s.label, s.errMsg)
	} else {
		if IsTTY() {
			fmt.Printf("\r✓ %s\n", s.label)
		} else {
			fmt.Printf("✓ %s\n", s.label)
		}
	}
}

// Fail stops the spinner and shows a failure message.
func (s *Spinner) Fail(err error) {
	s.failed = true
	if err != nil {
		s.errMsg = err.Error()
	}
	s.Stop()
}

// PrintError prints an error message to stderr.
func PrintError(err error) {
	msg := Colorize("Error: ", ColorRed+ColorBold) + err.Error()
	fmt.Fprintln(os.Stderr, msg)
}

// PrintWarning prints a warning message.
func PrintWarning(msg string) {
	warn := Colorize("Warning: ", ColorYellow+ColorBold) + msg
	fmt.Fprintln(os.Stderr, warn)
}
