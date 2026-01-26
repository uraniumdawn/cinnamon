// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Package config provides configuration management for the cinnamon application.
package config

import (
	"os"
	"time"

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
		CliTemplates     []string                `yaml:"cli_templates,omitempty"`
		API              ApiConfig               `yaml:"api,omitempty"`
	} `yaml:"cinnamon"`
}

type ApiConfig struct {
	Timeout int `yaml:"timeout"`
}

type ClusterConfig struct {
	Name       string            `yaml:"name"`
	Properties map[string]string `yaml:"properties"`
	Selected   bool              `yaml:"selected,omitempty"`
}

func (c *ClusterConfig) GetBootstrapServers() string {
	if bootstrap, ok := c.Properties["bootstrap.servers"]; ok {
		return bootstrap
	}
	return ""
}

// GetAPICallTimeout returns the API call timeout duration.
// Returns 10 seconds as default if not configured or invalid.
func (c *Config) GetAPICallTimeout() time.Duration {
	if c.Cinnamon.API.Timeout <= 0 {
		return 10 * time.Second
	}
	return time.Duration(c.Cinnamon.API.Timeout) * time.Second
}

// SchemaRegistryConfig holds Schema Registry connection properties.
type SchemaRegistryConfig struct {
	Name                   string `yaml:"name"`
	SchemaRegistryURL      string `yaml:"schema.registry.url"`
	SchemaRegistryUsername string `yaml:"schema.registry.sasl.username,omitempty"`
	SchemaRegistryPassword string `yaml:"schema.registry.sasl.password,omitempty"`
	Selected               bool   `yaml:"selected,omitempty"`
}

// LoadAppConfig loads the application configuration from the config file.
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

// Save writes the current configuration back to the config file.
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
