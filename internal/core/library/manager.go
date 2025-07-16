// SPDX-License-Identifier: Apache-2.0

package library

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Manager provides robust library path resolution and validation
type Manager struct {
	// Configuration
	globalLibraryPath string
	cmdLineLibraryPath string
	verboseLogging    bool
	
	// Cached validation results
	validatedPaths map[string]bool
}

// LibraryInfo contains information about a resolved library
type LibraryInfo struct {
	Path        string
	Source      string // "cmdline", "config", "default", "env"
	ActionsDir  string
	TemplatesDir string
	ConfigsDir  string
	MappingsDir string
	Exists      bool
	Valid       bool
	Errors      []string
}

// NewManager creates a new library manager
func NewManager(globalLibraryPath, cmdLineLibraryPath string, verbose bool) *Manager {
	return &Manager{
		globalLibraryPath:  globalLibraryPath,
		cmdLineLibraryPath: cmdLineLibraryPath,
		verboseLogging:     verbose,
		validatedPaths:     make(map[string]bool),
	}
}

// ResolveLibraryPath determines the library path using clear precedence rules
func (m *Manager) ResolveLibraryPath() (*LibraryInfo, error) {
	// Precedence order (highest to lowest):
	// 1. Command line flag (--library-path)
	// 2. DARN_HOME environment variable (for testing)
	// 3. Global config file setting
	// 4. Default global library path

	candidates := []struct {
		path   string
		source string
		desc   string
	}{
		{m.cmdLineLibraryPath, "cmdline", "command line --library-path flag"},
		{os.Getenv("DARN_HOME"), "env", "DARN_HOME environment variable"},
		{m.globalLibraryPath, "config", "global configuration file"},
		{expandPath("~/.darn/library"), "default", "default global library"},
	}

	var lastInfo *LibraryInfo
	var errors []string

	for _, candidate := range candidates {
		if candidate.path == "" {
			continue
		}

		expandedPath := expandPath(candidate.path)
		if m.verboseLogging {
			fmt.Printf("Checking library path from %s: %s\n", candidate.desc, expandedPath)
		}

		info := m.validateLibraryPath(expandedPath, candidate.source)
		lastInfo = info

		if info.Valid {
			if m.verboseLogging {
				fmt.Printf("✓ Using library from %s: %s\n", candidate.desc, expandedPath)
			}
			return info, nil
		}

		errorMsg := fmt.Sprintf("%s (%s): %s", candidate.desc, expandedPath, strings.Join(info.Errors, ", "))
		errors = append(errors, errorMsg)

		if m.verboseLogging {
			fmt.Printf("✗ Invalid library at %s: %s\n", expandedPath, strings.Join(info.Errors, ", "))
		}
	}

	// If we get here, no valid library was found
	if lastInfo != nil {
		lastInfo.Errors = errors
		return lastInfo, fmt.Errorf("no valid library found. Tried:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil, fmt.Errorf("no library paths configured")
}

// validateLibraryPath validates that a library path exists and has required structure
func (m *Manager) validateLibraryPath(path, source string) *LibraryInfo {
	info := &LibraryInfo{
		Path:        path,
		Source:      source,
		ActionsDir:  filepath.Join(path, "actions"),
		TemplatesDir: filepath.Join(path, "templates"),
		ConfigsDir:  filepath.Join(path, "configs"),
		MappingsDir: filepath.Join(path, "mappings"),
		Exists:      false,
		Valid:       false,
		Errors:      []string{},
	}

	// Check if main path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		info.Errors = append(info.Errors, "library directory does not exist")
		return info
	}
	info.Exists = true

	// Check required subdirectories
	requiredDirs := map[string]string{
		"actions":   info.ActionsDir,
		"templates": info.TemplatesDir,
		"configs":   info.ConfigsDir,
		"mappings":  info.MappingsDir,
	}

	missingDirs := []string{}
	for name, dirPath := range requiredDirs {
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			missingDirs = append(missingDirs, name)
		}
	}

	if len(missingDirs) > 0 {
		info.Errors = append(info.Errors, fmt.Sprintf("missing required subdirectories: %s", strings.Join(missingDirs, ", ")))
		return info
	}

	// Check if we can read from the directories
	for name, dirPath := range requiredDirs {
		if !isReadable(dirPath) {
			info.Errors = append(info.Errors, fmt.Sprintf("cannot read %s directory: %s", name, dirPath))
			return info
		}
	}

	info.Valid = true
	return info
}

// ValidateShellCommand validates that a shell command exists and is executable
func (m *Manager) ValidateShellCommand(command string) error {
	if command == "" {
		return fmt.Errorf("empty command")
	}

	// Handle platform-specific executables
	if runtime.GOOS == "windows" {
		// On Windows, try with common extensions if no extension provided
		if !strings.Contains(command, ".") {
			for _, ext := range []string{".exe", ".bat", ".cmd"} {
				if _, err := exec.LookPath(command + ext); err == nil {
					return nil
				}
			}
		}
	}

	// Standard PATH lookup
	_, err := exec.LookPath(command)
	if err != nil {
		return fmt.Errorf("command '%s' not found in PATH: %w", command, err)
	}

	return nil
}

// CreateLibraryStructure creates a new library structure at the given path
func (m *Manager) CreateLibraryStructure(path string) error {
	expandedPath := expandPath(path)
	
	if m.verboseLogging {
		fmt.Printf("Creating library structure at: %s\n", expandedPath)
	}

	// Create main directory
	if err := os.MkdirAll(expandedPath, 0755); err != nil {
		return fmt.Errorf("failed to create library directory: %w", err)
	}

	// Create required subdirectories
	subdirs := []string{"actions", "templates", "configs", "mappings"}
	for _, subdir := range subdirs {
		subdirPath := filepath.Join(expandedPath, subdir)
		if err := os.MkdirAll(subdirPath, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", subdir, err)
		}
		if m.verboseLogging {
			fmt.Printf("Created directory: %s\n", subdirPath)
		}
	}

	return nil
}

// ListAvailableActions returns a list of available actions in the library
func (m *Manager) ListAvailableActions(libraryPath string) ([]string, error) {
	actionsDir := filepath.Join(libraryPath, "actions")
	
	entries, err := os.ReadDir(actionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read actions directory: %w", err)
	}

	var actions []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			// Remove .yaml extension
			actionName := strings.TrimSuffix(entry.Name(), ".yaml")
			actions = append(actions, actionName)
		}
	}

	return actions, nil
}

// expandPath expands ~ to home directory and handles DARN_HOME for testing
func expandPath(path string) string {
	if path == "" {
		return ""
	}

	// Handle DARN_HOME environment variable for testing
	if darnHome := os.Getenv("DARN_HOME"); darnHome != "" {
		if path == "~" || path == "~/" {
			return darnHome
		}
		if strings.HasPrefix(path, "~/") {
			return filepath.Join(darnHome, path[2:])
		}
		if strings.HasPrefix(path, "~/.darn/library") {
			return filepath.Join(darnHome, ".darn", "library")
		}
	}

	// Standard home directory expansion
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			// Return original path if we can't expand
			return path
		}
		return filepath.Join(home, path[2:])
	}

	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}

	return path
}

// isReadable checks if a directory is readable
func isReadable(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Try to read the directory
	_, err = file.Readdir(1)
	// EOF is expected for empty directories
	return err == nil || err.Error() == "EOF"
}

// GetDiagnostics returns diagnostic information about the library system
func (m *Manager) GetDiagnostics() map[string]interface{} {
	diagnostics := make(map[string]interface{})
	
	diagnostics["cmdline_library_path"] = m.cmdLineLibraryPath
	diagnostics["global_library_path"] = m.globalLibraryPath
	diagnostics["darn_home"] = os.Getenv("DARN_HOME")
	
	if home, err := os.UserHomeDir(); err == nil {
		diagnostics["user_home"] = home
	} else {
		diagnostics["user_home_error"] = err.Error()
	}
	
	// Test library resolution
	if info, err := m.ResolveLibraryPath(); err == nil {
		diagnostics["resolved_library"] = map[string]interface{}{
			"path":   info.Path,
			"source": info.Source,
			"valid":  info.Valid,
			"exists": info.Exists,
			"errors": info.Errors,
		}
	} else {
		diagnostics["resolution_error"] = err.Error()
	}
	
	return diagnostics
}