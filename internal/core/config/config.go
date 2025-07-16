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
	"github.com/kusari-oss/darn/internal/core/library"
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
	LibraryPath        string `yaml:"library_path"`
	CmdLineLibraryPath string `yaml:"-"` // Exclude from YAML marshalling
	UseGlobal          bool   `yaml:"use_global"`
	UseLocal           bool   `yaml:"use_local"`
	GlobalFirst        bool   `yaml:"global_first"`
	
	// Runtime library manager
	LibraryManager *library.Manager `yaml:"-"`
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
		LibraryPath:  ExpandPathWithTilde(DefaultGlobalLibrary),
		// CmdLineLibraryPath is initialized as an empty string by default
		UseGlobal:   true,
		UseLocal:    false,
		GlobalFirst: true,
	}
}

// NewState creates a new state object with the current project directory
func NewState(projectDir, version string) *State {
	now := time.Now().Format(time.RFC3339)
	return &State{
		ProjectDir:    projectDir,
		LibraryInUse:  ExpandPathWithTilde(DefaultGlobalLibrary),
		LastUpdated:   now,
		InitializedAt: now,
		Version:       version,
	}
}

// ExpandPathWithTilde expands ~ to user home directory
// It respects the DARN_HOME environment variable for testing purposes.
func ExpandPathWithTilde(path string) string {
	if path == "~" {
		home := getHomeDir()
		if home == "" {
			return path // Return original if can't expand
		}
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home := getHomeDir()
		if home == "" {
			return path // Return original if can't expand
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// getHomeDir returns the home directory, respecting DARN_HOME for testing
func getHomeDir() string {
	// Check for test override first
	if darnHome := os.Getenv("DARN_HOME"); darnHome != "" {
		return darnHome
	}
	
	home, err := os.UserHomeDir()
	if err != nil {
		return "" // Return empty if can't determine
	}
	return home
}

// GlobalConfigFilePath returns the absolute path to the global darn config file.
// It respects the DARN_HOME environment variable for testing purposes.
func GlobalConfigFilePath() (string, error) {
	var home string
	
	// Check for test override first
	if darnHome := os.Getenv("DARN_HOME"); darnHome != "" {
		home = darnHome
	} else {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not get user home directory: %w", err)
		}
	}
	
	return filepath.Join(home, DefaultConfigDir, DefaultConfigFileName), nil
}

// LoadConfig loads the application configuration.
// It starts with default settings, then attempts to merge settings from a global
// configuration file. A command-line provided library path will override any
// library path found in the configuration files.
// The globalConfigPathOverride parameter allows specifying a custom path for the global
// config file, primarily for testing or special use cases. If empty, the default
// global config path (e.g., ~/.darn/config.yaml) is used.
func LoadConfig(cmdLineLibraryPath string, globalConfigPathOverride string) (*Config, error) {
	// Start with default configuration
	config := NewDefaultConfig()

	// Determine global config path
	var globalConfigPath string
	var err error
	if globalConfigPathOverride != "" {
		globalConfigPath = ExpandPathWithTilde(globalConfigPathOverride)
	} else {
		globalConfigPath, err = GlobalConfigFilePath()
		if err != nil {
			// Non-fatal, as global config might not be required or might be created later.
			// Proceed with an empty path, loadConfigFile will handle it.
			// Or, decide if this should be a fatal error. For now, let's assume it can proceed.
			fmt.Printf("Warning: could not determine global config path: %v\n", err)
			globalConfigPath = "" // Explicitly set to empty to avoid using uninitialized path
		}
	}

	// Try to load global config
	globalConfig, err := LoadConfigFile(globalConfigPath)
	if err == nil {
		// Merge global config with defaults
		mergeConfigs(config, globalConfig)
		// Ensure LibraryPath from global config is expanded if it was set
		if globalConfig.LibraryPath != "" {
			config.LibraryPath = ExpandPathWithTilde(globalConfig.LibraryPath)
		}
	} else if !os.IsNotExist(err) && globalConfigPath != "" {
		// Only print a warning if the error is not "file not found"
		// and a specific globalConfigPath was attempted (not empty).
		fmt.Printf("Warning: could not load global config file '%s': %v\n", globalConfigPath, err)
	}

	// Override with command-line library path if provided
	if cmdLineLibraryPath != "" {
		config.LibraryPath = ExpandPathWithTilde(cmdLineLibraryPath)
		config.CmdLineLibraryPath = ExpandPathWithTilde(cmdLineLibraryPath) // Also store the original cmd line path
	}

	// Post-condition: config.LibraryPath will be:
	// 1. The cmdLineLibraryPath (expanded), if provided.
	// 2. The globalConfig.LibraryPath (expanded), if cmdLine was not provided and global was found and had a path.
	// 3. The NewDefaultConfig().LibraryPath (expanded), if neither of the above set it.
	// All paths that are tilde-expanded by ExpandPathWithTilde will be absolute.
	// Paths that are not tilde-expanded (e.g. already absolute, or relative without tilde) remain as is.

	// Initialize the library manager
	config.LibraryManager = library.NewManager(config.LibraryPath, config.CmdLineLibraryPath, false)

	return config, nil
}

// LoadConfigFile loads a configuration from a specific file path
func LoadConfigFile(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("config file path cannot be empty")
	}
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
		target.LibraryPath = ExpandPathWithTilde(source.LibraryPath)
	}
	// CmdLineLibraryPath is not merged here as it's handled in LoadConfig directly.
	// Boolean fields - only override if they're explicitly set in the source
	// This isn't perfect since there's no way to tell from the parsed struct if they were omitted,
	// but it's a reasonable approach for this use case
	target.UseGlobal = source.UseGlobal
	target.UseLocal = source.UseLocal
	target.GlobalFirst = source.GlobalFirst
}

// SaveConfig saves the configuration to the specified directory (typically for project-local configs)
func SaveConfig(config *Config, dir string) error {
	// Create config directory if it doesn't exist
	configDir := filepath.Join(dir, DefaultConfigDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("error creating config directory '%s': %w", configDir, err)
	}

	// Marshal config to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	// Write config to file
	configPath := filepath.Join(configDir, DefaultConfigFileName)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("error writing config file '%s': %w", configPath, err)
	}

	return nil
}

// SaveGlobalConfig saves the provided configuration to the user's global darn config path.
func SaveGlobalConfig(config *Config) error {
	globalPath, err := GlobalConfigFilePath()
	if err != nil {
		return fmt.Errorf("could not determine global config path for saving: %w", err)
	}

	// Ensure the directory exists (e.g., ~/.darn/)
	globalDir := filepath.Dir(globalPath)
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		return fmt.Errorf("error creating global config directory '%s': %w", globalDir, err)
	}

	// Marshal config to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshaling global config: %w", err)
	}

	// Write config to file
	if err := os.WriteFile(globalPath, data, 0644); err != nil {
		return fmt.Errorf("error writing global config file '%s': %w", globalPath, err)
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

// GetLibraryInfo resolves and validates the library path using the library manager
func (c *Config) GetLibraryInfo() (*library.LibraryInfo, error) {
	if c.LibraryManager == nil {
		c.LibraryManager = library.NewManager(c.LibraryPath, c.CmdLineLibraryPath, false)
	}
	return c.LibraryManager.ResolveLibraryPath()
}

// SetVerboseLibraryLogging enables verbose logging for library operations
func (c *Config) SetVerboseLibraryLogging(verbose bool) {
	if c.LibraryManager == nil {
		c.LibraryManager = library.NewManager(c.LibraryPath, c.CmdLineLibraryPath, verbose)
	} else {
		// Recreate with new verbosity setting
		c.LibraryManager = library.NewManager(c.LibraryPath, c.CmdLineLibraryPath, verbose)
	}
}

// ValidateLibrarySetup validates that the configured library is usable
func (c *Config) ValidateLibrarySetup() error {
	info, err := c.GetLibraryInfo()
	if err != nil {
		return fmt.Errorf("library validation failed: %w", err)
	}
	
	if !info.Valid {
		return fmt.Errorf("library at %s is not valid: %s", info.Path, strings.Join(info.Errors, ", "))
	}
	
	return nil
}

// GetLibraryDiagnostics returns diagnostic information about the library system
func (c *Config) GetLibraryDiagnostics() map[string]interface{} {
	if c.LibraryManager == nil {
		c.LibraryManager = library.NewManager(c.LibraryPath, c.CmdLineLibraryPath, false)
	}
	return c.LibraryManager.GetDiagnostics()
}

// ValidateCommand validates that a command exists and is executable
func (c *Config) ValidateCommand(command string) error {
	if c.LibraryManager == nil {
		c.LibraryManager = library.NewManager(c.LibraryPath, c.CmdLineLibraryPath, false)
	}
	return c.LibraryManager.ValidateShellCommand(command)
}
