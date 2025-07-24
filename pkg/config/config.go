// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Cinnamon struct {
		Clusters         []*ClusterConfig        `yaml:"clusters"`
		SchemaRegistries []*SchemaRegistryConfig `yaml:"schema-registries"`
	} `yaml:"cinnamon"`
}

type ClusterConfig struct {
	Name           string            `yaml:"name"`
	Properties     map[string]string `yaml:"properties"`
	SchemaRegistry string            `yaml:"schema.registry.name"`
	Command        string            `yaml:"command"`
}

type SchemaRegistryConfig struct {
	Name                   string `yaml:"name"`
	SchemaRegistryUrl      string `yaml:"schema.registry.url"`
	SchemaRegistryUsername string `yaml:"schema.registry.sasl.username"`
	SchemaRegistryPassword string `yaml:"schema.registry.sasl.password"`
}

const (
	CinnamonEnvConfigDir = "CINNAMON_CONFIG_DIR"
)

func isEnvSet(env string) bool {
	return os.Getenv(env) != ""
}

func InitConfig() (*Config, error) {
	var configDir string
	switch {
	case isEnvSet(CinnamonEnvConfigDir):
		configDir = os.Getenv(CinnamonEnvConfigDir)
	default:
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal().Err(err).Msg("error getting home directory")
			return nil, err
		}
		configDir = homeDir
	}
	configPath := filepath.Join(configDir, ".cinnamon", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("error reading config file")
		return nil, err
	}

	content := os.ExpandEnv(string(data))

	config := &Config{}
	err = yaml.Unmarshal([]byte(content), config)
	if err != nil {
		log.Fatal().Err(err).Msg("error unmarshalling config")
		return nil, err
	}
	return config, nil
}
