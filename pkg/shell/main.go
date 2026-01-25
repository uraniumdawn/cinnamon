// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Package shell provides functionality for executing shell commands.
package shell

import (
	"bufio"
	"errors"
	"os/exec"
	"sync"
	"syscall"

	"github.com/rs/zerolog/log"
)

// Execute runs a shell command and streams output to channels.
// processDone receives exit codes following Unix convention:
// 0=success, 1-127=process error codes, 128+N=killed by signal N
// (e.g., 143=SIGTERM, 137=SIGKILL)
func Execute(args []string, rc, e chan string, sig chan syscall.Signal, processDone chan int) {
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

	var wg sync.WaitGroup
	wg.Add(2) // Two goroutines: stdout and stderr readers

	// Track which signal was received
	var receivedSignal syscall.Signal
	var signalOnce sync.Once

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			rc <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			log.Debug().Err(err).Msg("stdout scanner finished")
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			// Send stderr lines to the same record channel
			rc <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			log.Debug().Err(err).Msg("stderr scanner finished")
		}
	}()

	// Handle termination signals
	go func() {
		for s := range sig {
			signalOnce.Do(func() {
				if cmd.Process == nil {
					return
				}

				receivedSignal = s
				err := cmd.Process.Signal(s)
				if err != nil {
					errMsg := s.String() + " failed: " + err.Error()
					e <- errMsg
					log.Error().Err(err).Str("signal", s.String()).Msg("signal failed")
				} else {
					log.Info().Str("signal", s.String()).Msg("signal sent to process")
				}
				return
			})
		}
	}()

	waitErr := cmd.Wait()
	wg.Wait()
	var exitCode int

	// Check if user sent a signal
	if receivedSignal != 0 {
		exitCode = 128 + int(receivedSignal)
		log.Debug().
			Str("signal", receivedSignal.String()).
			Int("exitCode", exitCode).
			Msg("process terminated by user signal")
	} else if waitErr == nil {
		// Process completed successfully
		exitCode = 0
	} else {
		// Process failed - extract exit code
		exitCode = extractExitCode(waitErr)
	}

	// Signal that the process has terminated and all output has been consumed
	if processDone != nil {
		processDone <- exitCode
	}
}

// extractExitCode extracts the exit code from a command wait error
func extractExitCode(err error) int {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return 1 // Generic error
	}

	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		return 1 // Couldn't get status
	}

	if status.Signaled() {
		signal := status.Signal()
		exitCode := 128 + int(signal)
		log.Debug().
			Str("signal", signal.String()).
			Int("exitCode", exitCode).
			Msg("process terminated by signal")
		return exitCode
	}

	// Process exited with error code
	exitCode := status.ExitStatus()
	log.Debug().Int("exitCode", exitCode).Msg("process exited with error")
	return exitCode
}
