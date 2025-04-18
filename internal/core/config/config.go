// SPDX-License-Identifier: Apache-2.0

// TODO: This file could use some love.
// Need to decide if we support both local and global config.
// Need to decide how we want to manage per project config if at all.
// A lot of the stuff in here isn't really used yet.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kusari-oss/darn/internal/core/action"
	"gopkg.in/yaml.v3"
)

// Constants for default paths
const (
	DefaultConfigDir      = ".darn"
	DefaultGlobalLibrary  = "~/.darn/library"
	DefaultConfigFileName = "config.yaml"
	DefaultStateFileName  = "state.yaml"
)

// Config holds the global application configuration
type Config struct {
	// Standard configuration
	TemplatesDir string `yaml:"templates_dir"`
	ActionsDir   string `yaml:"actions_dir"`
	ConfigsDir   string `yaml:"configs_dir"`
	MappingsDir  string `yaml:"mappings_dir"` // New field

	// Library-related configuration
	LibraryPath string `yaml:"library_path"`
	UseGlobal   bool   `yaml:"use_global"`
	UseLocal    bool   `yaml:"use_local"`
	GlobalFirst bool   `yaml:"global_first"`
}

// State holds the runtime state of darn
type State struct {
	ProjectDir    string `yaml:"project_dir"`    // Current project directory
	LibraryInUse  string `yaml:"library_in_use"` // Path to the library currently in use
	LastUpdated   string `yaml:"last_updated"`   // When the library was last updated
	InitializedAt string `yaml:"initialized_at"` // When the project was initialized
	Version       string `yaml:"version"`        // Version of darn used
}

// NewDefaultConfig creates a default configuration
func NewDefaultConfig() *Config {
	return &Config{
		TemplatesDir: "templates",
		ActionsDir:   "actions",
		ConfigsDir:   "configs",
		MappingsDir:  "mappings",
		LibraryPath:  ExpandPath(DefaultGlobalLibrary),
		UseGlobal:    true,
		UseLocal:     false,
		GlobalFirst:  true,
	}
}

// NewState creates a new state object with the current project directory
func NewState(projectDir, version string) *State {
	now := time.Now().Format(time.RFC3339)
	return &State{
		ProjectDir:    projectDir,
		LibraryInUse:  ExpandPath(DefaultGlobalLibrary),
		LastUpdated:   now,
		InitializedAt: now,
		Version:       version,
	}
}

// ExpandPath expands ~ to user home directory
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path // Return original if can't expand
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// LoadConfig loads the application configuration, merging global and project configs
func LoadConfig(projectDir string) (*Config, error) {
	// Start with default configuration
	config := NewDefaultConfig()

	// Try to load global config
	globalConfig, err := loadConfigFile(ExpandPath("~/.darn/config.yaml"))
	if err == nil {
		// Merge global config with defaults
		mergeConfigs(config, globalConfig)
	}

	// Try to load project config
	projectConfigPath := filepath.Join(projectDir, DefaultConfigDir, DefaultConfigFileName)
	projectConfig, err := loadConfigFile(projectConfigPath)
	if err == nil {
		// Project config overrides global config
		mergeConfigs(config, projectConfig)
	}

	return config, nil
}

// loadConfigFile loads a configuration from a specific file path
func loadConfigFile(path string) (*Config, error) {
	// Read the config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse the YAML
	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return config, nil
}

// mergeConfigs merges source config into target config
// Only non-zero values from source override target
func mergeConfigs(target, source *Config) {
	if source.TemplatesDir != "" {
		target.TemplatesDir = source.TemplatesDir
	}
	if source.ActionsDir != "" {
		target.ActionsDir = source.ActionsDir
	}
	if source.ConfigsDir != "" {
		target.ConfigsDir = source.ConfigsDir
	}
	if source.LibraryPath != "" {
		target.LibraryPath = source.LibraryPath
	}
	// Boolean fields - only override if they're explicitly set in the source
	// This isn't perfect since there's no way to tell from the parsed struct if they were omitted,
	// but it's a reasonable approach for this use case
	target.UseGlobal = source.UseGlobal
	target.UseLocal = source.UseLocal
	target.GlobalFirst = source.GlobalFirst
}

// SaveConfig saves the configuration to the specified directory
func SaveConfig(config *Config, dir string) error {
	// Create config directory if it doesn't exist
	configDir := filepath.Join(dir, DefaultConfigDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	// Marshal config to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	// Write config to file
	configPath := filepath.Join(configDir, DefaultConfigFileName)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

// SaveState saves the state to the specified directory
func SaveState(state *State, dir string) error {
	// Create config directory if it doesn't exist
	configDir := filepath.Join(dir, DefaultConfigDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	// Marshal state to YAML
	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("error marshaling state: %w", err)
	}

	// Write state to file
	statePath := filepath.Join(configDir, DefaultStateFileName)
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("error writing state file: %w", err)
	}

	return nil
}

// LoadState loads the state from the specified directory
func LoadState(dir string) (*State, error) {
	statePath := filepath.Join(dir, DefaultConfigDir, DefaultStateFileName)

	// Read the state file
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, err
	}

	// Parse the YAML
	state := &State{}
	if err := yaml.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("error parsing state file: %w", err)
	}

	return state, nil
}

// LoadActions loads all action definitions from the appropriate directories
// based on the configuration settings
func LoadActions(config *Config, projectDir string) (map[string]action.Config, error) {
	actions := make(map[string]action.Config)

	// Load global actions if enabled
	if config.UseGlobal {
		globalActionsDir := filepath.Join(config.LibraryPath, config.ActionsDir)
		globalActions, err := loadActionsFromDir(globalActionsDir)
		if err == nil {
			// Add global actions to the map
			for name, actionConfig := range globalActions {
				actions[name] = actionConfig
			}
		}
	}

	// Load local actions if enabled
	if config.UseLocal {
		localActionsDir := filepath.Join(projectDir, config.ActionsDir)
		localActions, err := loadActionsFromDir(localActionsDir)
		if err == nil {
			// Add or override with local actions
			for name, actionConfig := range localActions {
				// If global takes precedence and we already have this action, skip
				if config.GlobalFirst && config.UseGlobal {
					if _, exists := actions[name]; exists {
						continue
					}
				}
				actions[name] = actionConfig
			}
		}
	}

	if len(actions) == 0 {
		return nil, fmt.Errorf("no actions found in configured directories")
	}

	return actions, nil
}

// loadActionsFromDir loads actions from a specific directory
func loadActionsFromDir(actionsDir string) (map[string]action.Config, error) {
	actions := make(map[string]action.Config)

	// Check if directory exists
	if _, err := os.Stat(actionsDir); os.IsNotExist(err) {
		return actions, nil // Return empty map if directory doesn't exist
	}

	// Walk through the actions directory
	err := filepath.Walk(actionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-yaml files
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".yaml") {
			return nil
		}

		// Read the action file
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading action file %s: %w", path, err)
		}

		// Parse the YAML
		var actionConfig action.Config
		if err := yaml.Unmarshal(data, &actionConfig); err != nil {
			return fmt.Errorf("error parsing action file %s: %w", path, err)
		}

		// If no name is specified, use the filename (without extension)
		if actionConfig.Name == "" {
			actionConfig.Name = strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
		}

		actions[actionConfig.Name] = actionConfig
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error loading actions: %w", err)
	}

	return actions, nil
}
