// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"os"
	"path/filepath"

	"dario.cat/mergo"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Cinnamon struct {
		Clusters         []*ClusterConfig        `yaml:"clusters"`
		SchemaRegistries []*SchemaRegistryConfig `yaml:"schema-registries"`
	} `yaml:"cinnamon"`
	Colors *ColorConfig
}

type ColorConfig struct {
	Cinnamon struct {
		Cluster struct {
			FgColor string `yaml:"fgColor"`
			BgColor string `yaml:"bgColor"`
		} `yaml:"cluster"`
		Status struct {
			FgColor string `yaml:"fgColor"`
			BgColor string `yaml:"bgColor"`
		} `yaml:"status"`
		Label struct {
			FgColor string `yaml:"fgColor"`
			BgColor string `yaml:"bgColor"`
		} `yaml:"label"`
		Keybinding struct {
			Key   string `yaml:"key"`
			Value string `yaml:"value"`
		} `yaml:"keybinding"`
		Selection struct {
			FgColor string `yaml:"fgColor"`
			BgColor string `yaml:"bgColor"`
		} `yaml:"selection"`
		Title      string `yaml:"title"`
		Border     string `yaml:"border"`
		Background string `yaml:"background"`
		Foreground string `yaml:"foreground"`
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

func loadDefaultColorConfig() (*ColorConfig, error) {
	data, err := os.ReadFile("style.yaml")
	if err != nil {
		log.Error().Err(err).Msg("error reading default style.yaml")
		return nil, err
	}

	defaultConfig := &ColorConfig{}
	if err := yaml.Unmarshal(data, defaultConfig); err != nil {
		log.Error().Err(err).Msg("error unmarshalling default style.yaml")
		return nil, err
	}
	return defaultConfig, nil
}

func loadUserColorConfig(configDir string) (*ColorConfig, error) {
	configPath := filepath.Join(configDir, ".config", "cinnamon", "style.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil // User config is optional
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Error().Err(err).Msg("error reading user style.yaml")
		return nil, err
	}

	userConfig := &ColorConfig{}
	if err := yaml.Unmarshal(data, userConfig); err != nil {
		log.Error().Err(err).Msg("error unmarshalling user style.yaml")
		return nil, err
	}
	return userConfig, nil
}

func LoadColorConfig(configDir string) (*ColorConfig, error) {
	defaultConfig, err := loadDefaultColorConfig()
	if err != nil {
		return nil, err
	}

	userConfig, err := loadUserColorConfig(configDir)
	if err != nil {
		return nil, err
	}

	if userConfig != nil {
		if err := mergo.Merge(defaultConfig, userConfig, mergo.WithOverride); err != nil {
			log.Error().Err(err).Msg("error merging color configs")
			return nil, err
		}
	}

	return defaultConfig, nil
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
	configPath := filepath.Join(configDir, ".config", "cinnamon", "config.yaml")
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

	colorConfig, err := LoadColorConfig(configDir)
	if err != nil {
		log.Fatal().Err(err).Msg("error loading color config")
		return nil, err
	}
	config.Colors = colorConfig

	return config, nil
}
