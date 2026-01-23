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
}

var (
	StatusLineCh    = make(chan Status, 10)
	statusLineTimer *time.Timer
)

// SendStatus sends a status message with the given TTL
func SendStatus(message string, ttl time.Duration) {
	StatusLineCh <- Status{Message: message, TTL: ttl}
}

// SendStatusWithDefaultTTL sends a status message with 10 second TTL
func SendStatusWithDefaultTTL(message string) {
	StatusLineCh <- Status{Message: message, TTL: 10 * time.Second}
}

// SendStatusInfinite sends a status message that never auto-clears
func SendStatusInfinite(message string) {
	StatusLineCh <- Status{Message: message, TTL: 0}
}

// ClearStatus clears the status line immediately
func ClearStatus() {
	StatusLineCh <- Status{Message: "", TTL: 0}
}

// RunStatusLineHandler handles status messages with spinner animation
func (app *App) RunStatusLineHandler(ctx context.Context, in chan Status) {
	go func() {
		spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
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
						spinnerActive = true
						app.Layout.StatusHistory.AddEntry(status.Message)
						app.Layout.StatusLine.SetText(
							fmt.Sprintf("%s %s", spinnerFrames[spinnerIdx], status.Message),
						)

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
					spinnerIdx = (spinnerIdx + 1) % len(spinnerFrames)
					app.QueueUpdateDraw(func() {
						if currentStatus != "" {
							app.Layout.StatusLine.SetText(
								fmt.Sprintf(
									"%s %s",
									spinnerFrames[spinnerIdx],
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
