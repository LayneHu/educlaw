package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pingjie/educlaw/pkg/config"
)

// ResolveConfigPath returns the config path to load and the path to save.
func ResolveConfigPath(path string) (string, string) {
	if path != "" {
		return path, path
	}

	if _, err := os.Stat("config.json"); err == nil {
		return "config.json", "config.json"
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		exeConfig := filepath.Join(exeDir, "config.json")
		if _, err := os.Stat(exeConfig); err == nil {
			return exeConfig, exeConfig
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		homePath := filepath.Join(home, ".educlaw", "config.json")
		if _, err := os.Stat(homePath); err == nil {
			return homePath, homePath
		}
	}
	if _, err := os.Stat("config.example.json"); err == nil {
		return "config.example.json", "config.json"
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		exeExample := filepath.Join(exeDir, "config.example.json")
		if _, err := os.Stat(exeExample); err == nil {
			return exeExample, filepath.Join(exeDir, "config.json")
		}
	}
	return "config.json", "config.json"
}

// LoadConfigWithPath looks for config.json or config.example.json in common locations.
// If no config file exists yet, it returns the default config plus the path where it should be saved.
func LoadConfigWithPath(path string) (*config.Config, string, error) {
	loadPath, savePath := ResolveConfigPath(path)
	if _, err := os.Stat(loadPath); err == nil {
		cfg, err := config.Load(loadPath)
		if err != nil {
			return nil, "", err
		}
		return cfg, savePath, nil
	}
	return config.Default(), savePath, nil
}

// LoadConfig looks for config.json or config.example.json in common locations.
func LoadConfig(path string) (*config.Config, error) {
	cfg, _, err := LoadConfigWithPath(path)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return cfg, nil
}
