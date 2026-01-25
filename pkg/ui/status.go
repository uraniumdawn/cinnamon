// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// Status represents a status message with optional time-to-live
type Status struct {
	Message string
	TTL     time.Duration // 0 means infinite (no auto-clear)
	Spinner bool          // true to show spinner animation
}

var (
	StatusLineCh    = make(chan Status, 10)
	statusLineTimer *time.Timer

	SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
)

// SendStatus sends a status message with the given TTL and spinner control
func SendStatus(message string, ttl time.Duration, spinner bool) {
	StatusLineCh <- Status{Message: message, TTL: ttl, Spinner: spinner}
}

// SendStatusWithDefaultTTL sends a status message with 10 second TTL and spinner
func SendStatusWithDefaultTTL(message string) {
	StatusLineCh <- Status{Message: message, TTL: 10 * time.Second, Spinner: true}
}

// SendStatusInfinite sends a status message that never auto-clears with spinner
func SendStatusInfinite(message string) {
	StatusLineCh <- Status{Message: message, TTL: 0, Spinner: true}
}

// ClearStatus clears the status line immediately
func ClearStatus() {
	StatusLineCh <- Status{Message: "", TTL: 0, Spinner: false}
}

// RunStatusLineHandler handles status messages with spinner animation
func (app *App) RunStatusLineHandler(ctx context.Context, in chan Status) {
	go func() {
		spinnerTicker := time.NewTicker(100 * time.Millisecond)
		defer spinnerTicker.Stop()

		var currentStatus string
		var spinnerIdx int
		var spinnerActive bool

		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("shutting down status line handler")
				return
			case status := <-in:
				app.QueueUpdateDraw(func() {
					if status.Message != "" {
						currentStatus = status.Message
						spinnerActive = status.Spinner
						app.Layout.StatusHistory.AddEntry(status.Message)

						// Display with or without spinner based on status.Spinner
						if status.Spinner {
							app.Layout.StatusLine.SetText(
								fmt.Sprintf(
									"%s %s",
									SpinnerFrames[spinnerIdx],
									status.Message,
								),
							)
						} else {
							app.Layout.StatusLine.SetText(status.Message)
						}

						// Auto-clear based on TTL (0 means infinite)
						if statusLineTimer != nil {
							statusLineTimer.Stop()
						}
						if status.TTL > 0 {
							statusLineTimer = time.AfterFunc(status.TTL, func() {
								app.QueueUpdateDraw(func() {
									currentStatus = ""
									spinnerActive = false
									app.Layout.StatusLine.SetText("")
								})
							})
						}
					} else {
						// Clear status line immediately
						currentStatus = ""
						spinnerActive = false
						app.Layout.StatusLine.SetText("")
						if statusLineTimer != nil {
							statusLineTimer.Stop()
						}
					}
				})
			case <-spinnerTicker.C:
				if spinnerActive {
					spinnerIdx = (spinnerIdx + 1) % len(SpinnerFrames)
					app.QueueUpdateDraw(func() {
						if currentStatus != "" {
							app.Layout.StatusLine.SetText(
								fmt.Sprintf(
									"%s %s",
									SpinnerFrames[spinnerIdx],
									currentStatus,
								),
							)
						}
					})
				}
			}
		}
	}()
}
