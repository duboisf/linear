package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Command defines a named custom command for interactive mode.
type Command struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
	Exec    bool   `yaml:"exec"`
}

// Config holds all user configuration loaded from config.yaml.
type Config struct {
	Interactive InteractiveConfig `yaml:"interactive"`
}

// InteractiveConfig holds settings for interactive (fzf) mode.
type InteractiveConfig struct {
	Commands []Command `yaml:"commands"`
}

// Load reads config from $XDG_CONFIG_HOME/linear/config.yaml.
// If the file does not exist, default values are returned (not an error).
// configDir overrides the config directory resolution; nil uses os.UserConfigDir.
func Load(configDir func() (string, error)) (*Config, error) {
	if configDir == nil {
		configDir = os.UserConfigDir
	}
	dir, err := configDir()
	if err != nil {
		return defaults(), nil
	}

	data, err := os.ReadFile(filepath.Join(dir, "linear", "config.yaml"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaults(), nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return &cfg, nil
}

// FilePath returns the path to the config file based on the user's config directory.
// Returns empty string if the config directory cannot be determined.
func FilePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "linear", "config.yaml")
}

func defaults() *Config {
	return &Config{}
}
