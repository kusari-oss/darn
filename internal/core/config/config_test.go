package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// Helper function to create a temporary config file
func createTempConfigFile(t *testing.T, dir string, filename string, content *Config) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	data, err := yaml.Marshal(content)
	assert.NoError(t, err)
	err = os.WriteFile(path, data, 0644)
	assert.NoError(t, err)
	return path
}

// Helper function to create a temporary .darn directory with a config file
func createTempDarnConfig(t *testing.T, baseDir string, configContent *Config) string {
	t.Helper()
	darnDir := filepath.Join(baseDir, DefaultConfigDir)
	err := os.MkdirAll(darnDir, 0755)
	assert.NoError(t, err)
	return createTempConfigFile(t, darnDir, DefaultConfigFileName, configContent)
}

func TestLoadConfig_Prioritization(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err)

	// Test cases
	tests := []struct {
		name                     string
		setupGlobalConfig        func(t *testing.T, globalDir string) // For default global ~/.darn/config.yaml
		setupCustomGlobalConfig  func(t *testing.T, customGlobalDir string) string // For globalConfigPathOverride
		// setupProjectConfig       func(t *testing.T, projectDir string) // Removed
		cmdLineLibraryPath       string
		globalConfigPathOverride string
		// projectDir               string // Removed
		expectedLibraryPath      string // Use placeholders like {{HOME}}
		expectedCmdLinePath      string // Use placeholders
		expectError              bool
	}{
		{
			name:                "1. Default global library when nothing else is set",
			expectedLibraryPath: ExpandPathWithTilde(DefaultGlobalLibrary), // Should be expanded
		},
		{
			name: "2. Global config (~/.darn/config.yaml) overrides default",
			setupGlobalConfig: func(t *testing.T, globalDir string) { // globalDir is tempHomeDir
				createTempDarnConfig(t, globalDir, &Config{LibraryPath: "~/global_lib_from_default_loc"})
			},
			expectedLibraryPath: filepath.Join(homeDir, "global_lib_from_default_loc"), // homeDir is actual user home for expectation
		},
		// Test Case 3 (Original: Project config overrides global config) - REMOVED as project config is no longer loaded by LoadConfig.
		// Expected behavior: Global config is used if present.
		{
			name: "3. Command-line path overrides global config (previously test 4, adapted)",
			setupGlobalConfig: func(t *testing.T, globalDir string) { // globalDir is tempHomeDir
				createTempDarnConfig(t, globalDir, &Config{LibraryPath: "~/global_lib_ignored_by_cmdline"})
			},
			cmdLineLibraryPath:  "~/cmd_line_over_global",
			expectedLibraryPath: filepath.Join(homeDir, "cmd_line_over_global"),
			expectedCmdLinePath: filepath.Join(homeDir, "cmd_line_over_global"),
		},
		{
			name: "4. Custom global config (override path) overrides default (previously test 5)",
			setupCustomGlobalConfig: func(t *testing.T, customGlobalDir string) string {
				return createTempConfigFile(t, customGlobalDir, "custom_global.yaml", &Config{LibraryPath: "~/custom_global_lib"})
			},
			expectedLibraryPath: filepath.Join(homeDir, "custom_global_lib"),
		},
		// Test Case 6 (Original: Project config overrides custom global config) - REMOVED.
		// Expected behavior: Custom global config is used if present.
		{
			name: "5. Command-line overrides custom global config (previously test 7, adapted)",
			setupCustomGlobalConfig: func(t *testing.T, customGlobalDir string) string {
				return createTempConfigFile(t, customGlobalDir, "custom_global_ignored_by_cmdline.yaml", &Config{LibraryPath: "~/custom_global_lib_ignored_by_cmdline"})
			},
			cmdLineLibraryPath:  "/abs/cmd_line_lib_override_custom",
			expectedLibraryPath: "/abs/cmd_line_lib_override_custom",
			expectedCmdLinePath: "/abs/cmd_line_lib_override_custom",
		},
		{
			name:                "6. Expansion of ~ in cmdLineLibraryPath (previously test 8)",
			cmdLineLibraryPath:  "~/path_from_cmd",
			expectedLibraryPath: filepath.Join(homeDir, "path_from_cmd"),
			expectedCmdLinePath: filepath.Join(homeDir, "path_from_cmd"),
		},
		// Test Case 9 (Original: Expansion of ~ in project config LibraryPath) - REMOVED.
		{
			name: "7. Expansion of ~ in global config LibraryPath (default location) (previously test 10)",
			setupGlobalConfig: func(t *testing.T, globalDir string) { // globalDir is tempHomeDir
				createTempDarnConfig(t, globalDir, &Config{LibraryPath: "~/path_from_global_config"})
			},
			expectedLibraryPath: filepath.Join(homeDir, "path_from_global_config"),
		},
		{
			name: "8. Expansion of ~ in global config LibraryPath (custom location) (previously test 11)",
			setupCustomGlobalConfig: func(t *testing.T, customGlobalDir string) string {
				return createTempConfigFile(t, customGlobalDir, "custom_global_for_tilde.yaml", &Config{LibraryPath: "~/path_from_custom_global"})
			},
			expectedLibraryPath: filepath.Join(homeDir, "path_from_custom_global"),
		},
		// Test Cases 12 & 13 (Original: Project config relative/absolute paths) - REMOVED.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directories
			tempBaseDir, err := os.MkdirTemp("", "darn_config_test_")
			assert.NoError(t, err)
			defer os.RemoveAll(tempBaseDir)

			// projectDir := tt.projectDir // Removed
			// if projectDir == "" { // Removed
			// 	projectDir = filepath.Join(tempBaseDir, "project") // Removed
			// 	err = os.MkdirAll(projectDir, 0755) // Removed
			// 	assert.NoError(t, err) // Removed
			// } // Removed

			// --- Global Config Setup (Default Location: ~/.darn) ---
			// We need to simulate the ~/.darn directory structure.
			// To do this without actually writing to the user's home,
			// we can temporarily set USER_HOME_DIR or HOME for the test's environment,
			// but a cleaner way for Go 1.17+ is t.Setenv.
			// However, ExpandPath uses os.UserHomeDir which isn't affected by t.Setenv directly.
			// So, we'll mock the default global config path by creating it in a *predictable*
			// temporary location and ensure ExpandPath resolves it there.
			// For tests involving `~`, we rely on `os.UserHomeDir()` returning the actual home.
			// For tests specifically about `~/.darn/config.yaml`, we'll create a fake global dir.

			fakeGlobalDarnDir := filepath.Join(tempBaseDir, "fake_home", ".darn")
			if tt.setupGlobalConfig != nil {
				err := os.MkdirAll(fakeGlobalDarnDir, 0755)
				assert.NoError(t, err)
				// To make LoadConfig look here, we'd need to modify where it looks for global config.
				// The current LoadConfig uses ExpandPath("~/.darn/config.yaml").
				// So, for these tests, setupGlobalConfig writes to the *actual* user's home if it's testing that.
				// This is not ideal. Let's adjust.
				// The tests are designed to check `globalConfigPathOverride` for custom global,
				// and the default `~/.darn/config.yaml` for the standard global.
				// For the standard global, we will assume it's either not present or use its actual path.
				// Let's refine how global configs are handled for testing.
			}
			
			// Adjust global config path for test isolation if testing default global path
			// originalUserHomeDir := os.Getenv("HOME") // Or "USERPROFILE" on Windows - Not used
			testGlobalDir := filepath.Join(tempBaseDir, "test_home")
			err = os.MkdirAll(testGlobalDir, 0755)
			assert.NoError(t, err)
			
			// For tests that rely on the default global path `~/.darn/config.yaml`
			// we will temporarily set HOME to our testGlobalDir.
			// This is a bit of a hack for os.UserHomeDir().
			// A better solution would be to inject the homedir getter into ExpandPath or config.
			// For now, we accept this limitation or focus on globalConfigPathOverride.

			// Let's simplify: if setupGlobalConfig is defined, it means we are testing the default global path.
			// We will create a temporary directory that *acts* as the home for this test run.
			// This is tricky because os.UserHomeDir() is hard to mock without linker flags.
			// --- Global Config Setup ---
			var actualGlobalConfigPathToUse string // This will be passed to LoadConfig

			// Setup for default global config path testing (e.g. ~/.darn/config.yaml)
			if tt.setupGlobalConfig != nil {
				// Create a temporary directory to act as HOME
				tempHomeDir, err := os.MkdirTemp(tempBaseDir, "testhome_")
				assert.NoError(t, err)
				defer os.RemoveAll(tempHomeDir) // Clean up temp home

				originalHome := os.Getenv("HOME")
				os.Setenv("HOME", tempHomeDir)
				defer os.Setenv("HOME", originalHome)

				// Now, GlobalConfigFilePath() inside LoadConfig should use this tempHomeDir.
				// The setupGlobalConfig function creates the .darn/config.yaml in this tempHomeDir.
				// For LoadConfig to pick it up without an override, globalConfigPathOverride should be empty.
				tt.setupGlobalConfig(t, tempHomeDir) // tt.setupGlobalConfig creates <tempHomeDir>/.darn/config.yaml
				actualGlobalConfigPathToUse = "" // Rely on default mechanism to find it in new HOME
			}

			// Setup for custom global config path (globalConfigPathOverride)
			if tt.setupCustomGlobalConfig != nil {
				customGlobalConfigDir := filepath.Join(tempBaseDir, "custom_global_configs")
				err := os.MkdirAll(customGlobalConfigDir, 0755)
				assert.NoError(t, err)
				// This path will be directly passed to LoadConfig
				actualGlobalConfigPathToUse = tt.setupCustomGlobalConfig(t, customGlobalConfigDir)
			}
			
			// If tt.globalConfigPathOverride is explicitly set in the test case, it takes precedence.
			// This is useful for testing scenarios where the override is directly provided.
			if tt.globalConfigPathOverride != "" {
				actualGlobalConfigPathToUse = tt.globalConfigPathOverride
			}


			// if tt.setupProjectConfig != nil { // Removed
			// 	tt.setupProjectConfig(t, projectDir) // Removed
			// } // Removed

			cfg, err := LoadConfig(tt.cmdLineLibraryPath, actualGlobalConfigPathToUse) // projectDir argument removed

			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, cfg)

			var expectedLibPath string
			// Paths starting with '~' or '/' are absolute or home-relative and should be expanded.
			// Other paths are relative and should be compared as is (they are stored as read from config).
			if strings.HasPrefix(tt.expectedLibraryPath, "~") || strings.HasPrefix(tt.expectedLibraryPath, "/") {
				// Determine the correct homeDir for path expansion in expected values.
				// If HOME was overridden for setupGlobalConfig, use that overridden HOME.
				var currentHomeDirForTest string
				if tt.setupGlobalConfig != nil {
					currentHomeDirForTest = os.Getenv("HOME") // This will be tempHomeDir
				} else {
					currentHomeDirForTest = homeDir // Actual user home
				}

				expectedLibPath = strings.ReplaceAll(tt.expectedLibraryPath, "{{HOME}}", currentHomeDirForTest)
				// {{PROJECT_DIR}} replacement for expectedLibraryPath
				// This logic assumes that if {{PROJECT_DIR}} is in expectedLibraryPath,
				// it means the path in the config file was relative and should now be an absolute path
				// relative to the projectDir for comparison.
				// However, LoadConfig stores paths as they are if not tilde-prefixed.
				// So, if config has "rel_path", cfg.LibraryPath will be "rel_path".
				// The test's expectedLibraryPath should reflect this.
				// If the test expects an absolute path like "projectDir/rel_path", then the placeholder
				// should be used carefully.
				// For now, let's assume {{PROJECT_DIR}} means "this part of the string should be the projectDir".
				// This is mainly for absolute paths that depend on the temp projectDir.
				// if strings.Contains(expectedLibPath, "{{PROJECT_DIR}}") { // projectDir no longer used for this
				// 	expectedLibPath = strings.ReplaceAll(expectedLibPath, "{{PROJECT_DIR}}", projectDir) // projectDir no longer used for this
				// } // projectDir no longer used for this
			} else {
				// If not starting with ~ or /, and no {{PROJECT_DIR}}, it's a relative path or a non-path string.
				// It should be stored as-is.
				expectedLibPath = tt.expectedLibraryPath
			}


			assert.Equal(t, expectedLibPath, cfg.LibraryPath, "Unexpected LibraryPath")

			var expectedCmdPath string
			if tt.expectedCmdLinePath != "" {
				var currentHomeDirForTest string
				if tt.setupGlobalConfig != nil {
					currentHomeDirForTest = os.Getenv("HOME")
				} else {
					currentHomeDirForTest = homeDir
				}
				expectedCmdPath = strings.ReplaceAll(tt.expectedCmdLinePath, "{{HOME}}", currentHomeDirForTest)
				// expectedCmdPath = strings.ReplaceAll(expectedCmdPath, "{{PROJECT_DIR}}", projectDir) // projectDir no longer used for this
			}
			assert.Equal(t, expectedCmdPath, cfg.CmdLineLibraryPath, "Unexpected CmdLineLibraryPath for test: "+tt.name)
		})
	}
}

func TestExpandPathWithTilde(t *testing.T) {
	// Capture original HOME to restore it, as some tests might modify it.
	originalHomeEnv := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHomeEnv)

	// Test Case 1: Standard home directory
	t.Run("StandardHome", func(t *testing.T) {
		// Use a temporary directory to simulate HOME only for this sub-test if needed,
		// or rely on the actual user home and ensure no side effects.
		// For ExpandPathWithTilde, it directly calls os.UserHomeDir().
		actualHomeDir, err := os.UserHomeDir()
		assert.NoError(t, err)

		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{"Tilde path", "~/testdir", filepath.Join(actualHomeDir, "testdir")},
			{"Absolute path", "/abs/path", "/abs/path"},
			{"Relative path", "rel/path", "rel/path"},
			{"Empty path", "", ""},
			{"Just tilde", "~", actualHomeDir},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				expanded := ExpandPathWithTilde(tt.input)
				assert.Equal(t, tt.expected, expanded)
			})
		}
	})

	// Test Case 2: Overridden HOME environment variable (if os.UserHomeDir respects it - it often doesn't directly)
	// This test is more about documenting behavior than strict testing of UserHomeDir mocks.
	// Go's os.UserHomeDir() might ignore $HOME on some systems (e.g., macOS when cached).
	t.Run("WithTemporaryHomeEnv", func(t *testing.T) {
		tempHome, err := os.MkdirTemp("", "fakehome_expand_")
		assert.NoError(t, err)
		defer os.RemoveAll(tempHome)

		os.Setenv("HOME", tempHome) // Set $HOME to our temporary directory

		// Re-fetch actual home dir, which *might* now be tempHome if os.UserHomeDir respects $HOME update.
		// Note: This is platform and Go version dependent.
		effectiveHomeDir, err := os.UserHomeDir()
		assert.NoError(t, err)
		// If effectiveHomeDir is not tempHome, it means os.UserHomeDir() did not use the new $HOME.
		// The test proceeds, but its interpretation might vary.

		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{"Tilde path with temp home", "~/testdir_temp", filepath.Join(effectiveHomeDir, "testdir_temp")},
			{"Just tilde with temp home", "~", effectiveHomeDir},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				expanded := ExpandPathWithTilde(tt.input)
				assert.Equal(t, tt.expected, expanded)
			})
		}
		os.Setenv("HOME", originalHomeEnv) // Restore HOME
	})
}

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()
	assert.Equal(t, "templates", cfg.TemplatesDir)
	assert.Equal(t, "actions", cfg.ActionsDir)
	assert.Equal(t, "configs", cfg.ConfigsDir)
	assert.Equal(t, "mappings", cfg.MappingsDir)
	assert.Equal(t, ExpandPathWithTilde(DefaultGlobalLibrary), cfg.LibraryPath) // Use updated function name
	assert.True(t, cfg.UseGlobal)
	assert.False(t, cfg.UseLocal)
	assert.True(t, cfg.GlobalFirst)
	assert.Empty(t, cfg.CmdLineLibraryPath)
}

func TestSaveLoadState(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "state_test_")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	originalState := NewState(tempDir, "1.0.0")
	originalState.LibraryInUse = "/new/lib/path"
	originalState.LastUpdated = "a while ago"

	err = SaveState(originalState, tempDir)
	assert.NoError(t, err)

	loadedState, err := LoadState(tempDir)
	assert.NoError(t, err)
	assert.NotNil(t, loadedState)

	assert.Equal(t, originalState.ProjectDir, loadedState.ProjectDir)
	assert.Equal(t, originalState.LibraryInUse, loadedState.LibraryInUse)
	assert.Equal(t, originalState.LastUpdated, loadedState.LastUpdated) // This will be different due to time.Now() in NewState
	assert.Equal(t, originalState.InitializedAt, loadedState.InitializedAt)
	assert.Equal(t, originalState.Version, loadedState.Version)
}
