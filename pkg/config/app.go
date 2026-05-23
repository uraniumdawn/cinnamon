// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Package config provides configuration management for the cinnamon application.
package config

import (
	"os"
	"time"

	"dario.cat/mergo"
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
		Connect          []*ConnectConfig        `yaml:"connect,omitempty"`
		API              ApiConfig               `yaml:"api,omitempty"`
		Style            string                  `yaml:"style,omitempty"`
	} `yaml:"cinnamon"`
}

type ApiConfig struct {
	Timeout        int `yaml:"timeout"`
	MaxConcurrency int `yaml:"max_concurrency,omitempty"`
}

type ClusterConfig struct {
	Name       string            `yaml:"name"`
	Properties map[string]string `yaml:"properties"`
	Selected   bool              `yaml:"selected,omitempty"`
	Mode       string            `yaml:"mode,omitempty"`
}

// IsReadOnly returns true when the cluster mode is set to "read-only".
func (c *ClusterConfig) IsReadOnly() bool {
	return c.Mode == "read-only"
}

func (c *ClusterConfig) GetBootstrapServers() string {
	if bootstrap, ok := c.Properties["bootstrap.servers"]; ok {
		return bootstrap
	}
	return ""
}

// GetAPICallTimeout returns the API call timeout duration.
func (c *Config) GetAPICallTimeout() time.Duration {
	return time.Duration(c.Cinnamon.API.Timeout) * time.Second
}

// GetMaxConcurrency returns the maximum number of concurrent Kafka API calls.
func (c *Config) GetMaxConcurrency() int {
	return c.Cinnamon.API.MaxConcurrency
}

// SchemaRegistryConfig holds Schema Registry connection properties.
type SchemaRegistryConfig struct {
	Name                   string `yaml:"name"`
	SchemaRegistryURL      string `yaml:"schema.registry.url"`
	SchemaRegistryUsername string `yaml:"schema.registry.sasl.username,omitempty"`
	SchemaRegistryPassword string `yaml:"schema.registry.sasl.password,omitempty"`
	Selected               bool   `yaml:"selected,omitempty"`
}

// ConnectConfig holds Kafka Connect connection properties.
type ConnectConfig struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Selected bool   `yaml:"selected,omitempty"`
	Mode     string `yaml:"mode,omitempty"`
}

// IsReadOnly returns true when the Connect cluster mode is set to "read-only".
func (c *ConnectConfig) IsReadOnly() bool {
	return c.Mode == "read-only"
}

// LoadAppConfig loads the application configuration by merging default_config.yaml
// (from the working directory) with the user config file. User values take precedence.
func LoadAppConfig() (*Config, error) {
	defaults, err := loadDefaultAppConfig()
	if err != nil {
		return nil, err
	}

	defAPI := defaults.Cinnamon.API

	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("error reading config file")
		return nil, err
	}

	override := &Config{}
	if err := yaml.Unmarshal([]byte(os.ExpandEnv(string(data))), override); err != nil {
		log.Fatal().Err(err).Msg("error unmarshalling config")
		return nil, err
	}

	if err := mergo.Merge(defaults, override, mergo.WithOverride); err != nil {
		log.Fatal().Err(err).Msg("error merging config")
		return nil, err
	}

	validateAPIConfig(&defaults.Cinnamon.API, defAPI)

	return defaults, nil
}

// validateAPIConfig resets any invalid (<=0) API fields to their defaults and logs a warning.
func validateAPIConfig(cfg *ApiConfig, def ApiConfig) {
	if cfg.Timeout <= 0 {
		log.Warn().Int("value", cfg.Timeout).Int("default", def.Timeout).
			Msg("invalid api.timeout, using default")
		cfg.Timeout = def.Timeout
	}
	if cfg.MaxConcurrency <= 0 {
		log.Warn().Int("value", cfg.MaxConcurrency).Int("default", def.MaxConcurrency).
			Msg("invalid api.max_concurrency, using default")
		cfg.MaxConcurrency = def.MaxConcurrency
	}
}

func loadDefaultAppConfig() (*Config, error) {
	data, err := os.ReadFile("default_config.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("error reading default_config.yaml")
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		log.Fatal().Err(err).Msg("error unmarshalling default_config.yaml")
		return nil, err
	}
	return cfg, nil
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
