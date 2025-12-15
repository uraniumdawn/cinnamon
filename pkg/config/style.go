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
		Placeholder string `yaml:"placeholder"`
		Title       string `yaml:"title"`
		Border      string `yaml:"border"`
		Background  string `yaml:"background"`
		Foreground  string `yaml:"foreground"`
	} `yaml:"cinnamon" `
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
	configPath := filepath.Join(configDir, "style.yaml")
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

func LoadColorConfig() (*ColorConfig, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}
	configDir := filepath.Dir(configPath)

	// 1. Load default style.yaml
	config, err := loadDefaultColorConfig()
	if err != nil {
		return nil, err
	}

	// 2. Load and merge user style.yaml
	userStyleConfig, err := loadUserColorConfig(configDir)
	if err != nil {
		return nil, err
	}
	if userStyleConfig != nil {
		if err := mergo.Merge(config, userStyleConfig, mergo.WithOverride); err != nil {
			log.Error().Err(err).Msg("error merging user style.yaml")
			return nil, err
		}
	}

	return config, nil
}
