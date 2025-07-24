// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package shell

import (
	"bufio"
	"os/exec"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

func Execute(args []string, rc chan string, e chan string, sig chan int) {
	cmd := exec.Command(args[0], args[1:]...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		errMsg := "Failed to get stdout: " + err.Error()
		e <- errMsg
		log.Error().Err(err).Msg(errMsg)
		return
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		errMsg := "Failed to get stderr: " + err.Error()
		e <- errMsg
		log.Error().Err(err).Msg(errMsg)
		return
	}

	if err := cmd.Start(); err != nil {
		errMsg := "Failed to start command: " + err.Error()
		e <- errMsg
		log.Error().Err(err).Msg(errMsg)
		return
	}

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			rc <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			e <- "Error reading stdout: " + err.Error()
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			e <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			e <- "Error reading stderr: " + err.Error()
		}
	}()

	go func() {
		for s := range sig {
			if s == 1 {
				err := cmd.Process.Signal(syscall.SIGTERM)
				if err != nil {
					e <- "Failed to send SIGTERM: " + err.Error()
					log.Error().Err(err).Msg("SIGTERM failed")
					return
				}
				e <- "SIGTERM sent. Waiting for graceful shutdown..."

				time.AfterFunc(5*time.Second, func() {
					if err := cmd.Process.Kill(); err == nil {
						e <- "Command forcefully killed after timeout."
						log.Warn().Msg("Command killed after timeout")
					}
				})
			}
		}
	}()

	cmd.Wait()
}