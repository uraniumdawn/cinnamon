// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"os"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type JqConfig struct {
	Enable  bool     `yaml:"enable,omitempty"`
	Command []string `yaml:"command,omitempty"`
}

type Config struct {
	Cinnamon struct {
		Clusters         []*ClusterConfig        `yaml:"clusters"`
		SchemaRegistries []*SchemaRegistryConfig `yaml:"schema-registries"`
		Jq               JqConfig                `yaml:"jq,omitempty"`
	} `yaml:"cinnamon"`
}

type ClusterConfig struct {
	Name           string            `yaml:"name"`
	Properties     map[string]string `yaml:"properties"`
	Selected       bool              `yaml:"selected,omitempty"`
}

type SchemaRegistryConfig struct {
	Name                   string `yaml:"name"`
	SchemaRegistryUrl      string `yaml:"schema.registry.url"`
	SchemaRegistryUsername string `yaml:"schema.registry.sasl.username,omitempty"`
	SchemaRegistryPassword string `yaml:"schema.registry.sasl.password,omitempty"`
	Selected               bool   `yaml:"selected,omitempty"`
}

func LoadAppConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("error reading config file")
		return nil, err
	}
	content := os.ExpandEnv(string(data))

	config := &Config{}
	if err := yaml.Unmarshal([]byte(content), config); err != nil {
		log.Fatal().Err(err).Msg("error unmarshalling config")
		return nil, err
	}

	return config, nil
}

func (c *Config) Save() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal config")
		return err
	}

	err = os.WriteFile(configPath, data, 0o644)
	if err != nil {
		log.Error().Err(err).Msg("failed to write config")
		return err
	}

	return nil
}
