// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		Search struct {
			FgColor string `yaml:"fgColor"`
			BgColor string `yaml:"bgColor"`
		} `yaml:"search"`
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
	data, err := os.ReadFile("default_style.yaml")
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

// LoadColorConfig loads color configuration.
// Loads default_style.yaml as the base. If stylePath is set (cinnamon.style in config.yaml)
// that file is loaded and merged on top, overriding any fields it defines.
// Relative stylePath is resolved against the config directory.
func LoadColorConfig(stylePath string) (*ColorConfig, error) {
	base, err := loadDefaultColorConfig()
	if err != nil {
		return nil, err
	}

	if stylePath == "" {
		return base, nil
	}

	if strings.HasPrefix(stylePath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot resolve ~: %w", err)
		}
		stylePath = filepath.Join(home, stylePath[2:])
	}

	override, err := loadColorConfigFromFile(stylePath)
	if err != nil {
		return nil, err
	}

	if err := mergo.Merge(base, override, mergo.WithOverride); err != nil {
		log.Error().Err(err).Str("path", stylePath).Msg("error merging style file")
		return nil, err
	}

	return base, nil
}

func loadColorConfigFromFile(path string) (*ColorConfig, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("style file not found: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Error().Err(err).Str("path", path).Msg("error reading style file")
		return nil, err
	}
	cfg := &ColorConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		log.Error().Err(err).Str("path", path).Msg("error unmarshalling style file")
		return nil, err
	}
	return cfg, nil
}
