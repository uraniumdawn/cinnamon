// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Package shell provides functionality for executing shell commands.
package shell

import (
	"bufio"
	"errors"
	"os/exec"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

// Execute runs a shell command and streams output to channels.
func Execute(args []string, rc, e chan string, sig chan int) {
	cmd := exec.Command(args[0], args[1:]...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		errMsg := "failed to get stdout: " + err.Error()
		e <- errMsg
		log.Error().Err(err).Msg(errMsg)
		return
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		errMsg := "failed to get stderr: " + err.Error()
		e <- errMsg
		log.Error().Err(err).Msg(errMsg)
		return
	}

	if err := cmd.Start(); err != nil {
		errMsg := "failed to start command: " + err.Error()
		e <- errMsg
		log.Error().Err(err).Msg(errMsg)
		return
	}

	// Channel to track when output reading is complete
	done := make(chan bool, 2)

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			rc <- scanner.Text()
		}
		// Only report error if it's not due to pipe being closed
		if err := scanner.Err(); err != nil {
			log.Debug().Err(err).Msg("stdout scanner finished")
		}
		done <- true
	}()

	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			e <- scanner.Text()
		}
		// Only report error if it's not due to pipe being closed
		if err := scanner.Err(); err != nil {
			log.Debug().Err(err).Msg("stderr scanner finished")
		}
		done <- true
	}()

	// Handle termination signals
	go func() {
		for s := range sig {
			if s == 1 {
				// Send SIGTERM for graceful shutdown
				if cmd.Process != nil {
					err := cmd.Process.Signal(syscall.SIGTERM)
					if err != nil {
						log.Error().Err(err).Msg("SIGTERM failed")
					} else {
						log.Info().Msg("SIGTERM sent to process")
					}

					// Set up force kill after timeout
					time.AfterFunc(5*time.Second, func() {
						if cmd.Process != nil {
							if err := cmd.Process.Kill(); err == nil {
								log.Warn().Msg("process killed after timeout")
							}
						}
					})
				}
				return
			}
			if s == 2 {
				// Send SIGKILL for immediate termination
				if cmd.Process != nil {
					if err := cmd.Process.Kill(); err != nil {
						log.Error().Err(err).Msg("SIGKILL failed")
					} else {
						log.Info().Msg("SIGKILL sent to process")
					}
				}
				return
			}
		}
	}()

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		// Only log if it's not a signal-related exit
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() {
					log.Debug().
						Str("signal", status.Signal().String()).
						Msg("process terminated by signal")
				} else {
					log.Debug().Int("code", status.ExitStatus()).Msg("process exited")
				}
			}
		}
	}

	// Wait for both output readers to finish
	<-done
	<-done
}
