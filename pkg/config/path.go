// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

const (
	CinnamonEnvConfigDir = "CINNAMON_CONFIG_DIR"
)

func isEnvSet(env string) bool {
	return os.Getenv(env) != ""
}

func GetConfigPath() (string, error) {
	var configDir string
	switch {
	case isEnvSet(CinnamonEnvConfigDir):
		configDir = os.Getenv(CinnamonEnvConfigDir)
	default:
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal().Err(err).Msg("error getting home directory")
			return "", err
		}
		configDir = homeDir
	}
	return filepath.Join(configDir, ".config", "cinnamon", "config.yaml"), nil
}
