package library

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"bytes"

	"github.com/kusari-oss/darn/internal/core/config"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// Helper to execute the init command
func executeInitCommand(t *testing.T, args []string) (*cobra.Command, error) {
	t.Helper()
	cmd := newInitCommand()
	cmd.SetArgs(args)
	// Suppress output during tests
	cmd.SetOut(nil)
	cmd.SetErr(nil)
	err := cmd.Execute()
	return cmd, err
}

// TestRunInitCommand_DefaultPath_InitializesGlobalDefaultLocation tests `darn library init` with no arguments.
// It should initialize the library content at the default global library location (~/.darn/library).
func TestRunInitCommand_DefaultPath_InitializesGlobalDefaultLocation(t *testing.T) {
	// Create a temporary directory to act as HOME
	tempHomeDir, err := os.MkdirTemp("", "darn_init_defaulthome_")
	assert.NoError(t, err)
	defer os.RemoveAll(tempHomeDir)

	originalDarnHome := os.Getenv("DARN_HOME")
	os.Setenv("DARN_HOME", tempHomeDir)
	defer func() {
		if originalDarnHome == "" {
			os.Unsetenv("DARN_HOME")
		} else {
			os.Setenv("DARN_HOME", originalDarnHome)
		}
	}()

	// The command is run from a different temporary directory to simulate a generic CWD.
	tempCwd, err := os.MkdirTemp("", "darn_init_defaultcwd_")
	assert.NoError(t, err)
	defer os.RemoveAll(tempCwd)

	originalWd, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(tempCwd)
	assert.NoError(t, err)
	defer os.Chdir(originalWd)

	// Execute 'darn library init' with no arguments
	_, err = executeInitCommand(t, []string{})
	assert.NoError(t, err)

	// Verify library content initialization at the default global path (within tempHomeDir)
	expectedLibPath := filepath.Join(tempHomeDir, config.DefaultConfigDir, "library") // This is config.DefaultGlobalLibrary expanded
	assert.DirExists(t, expectedLibPath, "Library directory should exist at default global location")
	assert.DirExists(t, filepath.Join(expectedLibPath, "actions"), "Actions directory should exist")
	assert.DirExists(t, filepath.Join(expectedLibPath, "templates"), "Templates directory should exist")
	assert.FileExists(t, filepath.Join(expectedLibPath, "actions", "add-security-md.yaml"), "Default action file should exist")

	// Assert no .darn/config.yaml or .darn/state.yaml created in CWD
	assert.NoFileExists(t, filepath.Join(tempCwd, config.DefaultConfigDir, config.DefaultConfigFileName), "No project config.yaml should be created in CWD")
	assert.NoFileExists(t, filepath.Join(tempCwd, config.DefaultConfigDir, config.DefaultStateFileName), "No project state.yaml should be created in CWD")

	// Assert global ~/.darn/config.yaml is not created or modified by 'init'
	globalConfigPath := filepath.Join(tempHomeDir, config.DefaultConfigDir, config.DefaultConfigFileName)
	assert.NoFileExists(t, globalConfigPath, "Global config.yaml should not be created/modified by init command")
}

// TestRunInitCommand_SpecificPath_InitializesAtGivenLocation tests `darn library init /path/to/lib`.
// It should initialize library content at the specified path.
func TestRunInitCommand_SpecificPath_InitializesAtGivenLocation(t *testing.T) {
	// Create a temporary directory to hold the new library
	customLibBaseDir, err := os.MkdirTemp("", "darn_init_customlib_base_")
	assert.NoError(t, err)
	defer os.RemoveAll(customLibBaseDir)
	customLibPath := filepath.Join(customLibBaseDir, "mytestlib")

	// The command is run from a different temporary directory to simulate a generic CWD.
	tempCwd, err := os.MkdirTemp("", "darn_init_specificcwd_")
	assert.NoError(t, err)
	defer os.RemoveAll(tempCwd)

	originalWd, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(tempCwd)
	assert.NoError(t, err)
	defer os.Chdir(originalWd)

	// Execute 'darn library init /path/to/lib'
	_, err = executeInitCommand(t, []string{customLibPath})
	assert.NoError(t, err)

	// Verify library content initialization at the specified path
	assert.DirExists(t, customLibPath, "Library directory should exist at specified path")
	assert.DirExists(t, filepath.Join(customLibPath, "actions"), "Actions directory should exist in custom lib")
	assert.DirExists(t, filepath.Join(customLibPath, "templates"), "Templates directory should exist in custom lib")
	assert.FileExists(t, filepath.Join(customLibPath, "actions", "add-security-md.yaml"), "Default action file should exist in custom lib")

	// Assert no .darn/config.yaml or .darn/state.yaml created in CWD
	assert.NoFileExists(t, filepath.Join(tempCwd, config.DefaultConfigDir, config.DefaultConfigFileName), "No project config.yaml should be created in CWD")
	assert.NoFileExists(t, filepath.Join(tempCwd, config.DefaultConfigDir, config.DefaultStateFileName), "No project state.yaml should be created in CWD")

	// Assert global ~/.darn/config.yaml is not created or modified
	// Use a temporary HOME to check this, ensuring no interference with user's actual global config.
	tempHomeDir, err := os.MkdirTemp("", "darn_init_specific_home_")
	assert.NoError(t, err)
	defer os.RemoveAll(tempHomeDir)
	originalDarnHome := os.Getenv("DARN_HOME")
	os.Setenv("DARN_HOME", tempHomeDir)
	defer func() {
		if originalDarnHome == "" {
			os.Unsetenv("DARN_HOME")
		} else {
			os.Setenv("DARN_HOME", originalDarnHome)
		}
	}()
	globalConfigPath := filepath.Join(tempHomeDir, config.DefaultConfigDir, config.DefaultConfigFileName)
	assert.NoFileExists(t, globalConfigPath, "Global config.yaml should not be created/modified by init command")
}

// TestRunInitCommand_CustomDirNames tests using custom directory names during library initialization.
func TestRunInitCommand_CustomDirNames(t *testing.T) {
	// Create a temporary directory to hold the new library with custom dir names
	customLibBaseDir, err := os.MkdirTemp("", "darn_init_custom_names_base_")
	assert.NoError(t, err)
	defer os.RemoveAll(customLibBaseDir)
	customLibPath := filepath.Join(customLibBaseDir, "my-custom-layout-lib")

	// The command is run from a different temporary directory
	tempCwd, err := os.MkdirTemp("", "darn_init_custom_names_cwd_")
	assert.NoError(t, err)
	defer os.RemoveAll(tempCwd)
	originalWd, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(tempCwd)
	assert.NoError(t, err)
	defer os.Chdir(originalWd)

	args := []string{
		customLibPath, // Target path for the library
		"--actions-dir", "my-bespoke-actions",
		"--templates-dir", "my-lovely-templates",
		"--configs-dir", "my-standard-configs",
		"--mappings-dir", "my-data-maps",
	}
	_, err = executeInitCommand(t, args)
	assert.NoError(t, err)

	assert.DirExists(t, customLibPath)
	assert.DirExists(t, filepath.Join(customLibPath, "my-bespoke-actions"))
	assert.DirExists(t, filepath.Join(customLibPath, "my-lovely-templates"))
	assert.DirExists(t, filepath.Join(customLibPath, "my-standard-configs"))
	assert.DirExists(t, filepath.Join(customLibPath, "my-data-maps"))

	// Check that default files are still copied to these custom-named directories
	assert.FileExists(t, filepath.Join(customLibPath, "my-bespoke-actions", "add-security-md.yaml"))
	assert.FileExists(t, filepath.Join(customLibPath, "my-lovely-templates", "security.md.tmpl"))

	// Ensure no project-specific config is created
	assert.NoFileExists(t, filepath.Join(tempCwd, config.DefaultConfigDir, config.DefaultConfigFileName))
}

func TestRunInitCommand_LocalOnly(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "darn_init_local_only_")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	originalWd, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(tempDir)
	assert.NoError(t, err)
	defer os.Chdir(originalWd)

	// The defaults.CopyDefaults function in the actual code will try to fetch from remote if not localOnly.
	// For this test, we primarily ensure the flag is passed and the command runs.
	// A deeper test would involve mocking the network call in defaults.CopyDefaults.
	// Here, we just check if the command completes and basic structure is created.
	// If the remote URL was invalid and localOnly=false, it might fail.
	// If localOnly=true, it should succeed regardless of remote state.

	// Define a specific temporary path for the library initialization
	targetLibPath := filepath.Join(tempDir, "my-local-only-lib")

	_, err = executeInitCommand(t, []string{targetLibPath, "--local-only"})
	assert.NoError(t, err, "Init with --local-only should succeed")

	// Verify library structure at the targetLibPath
	assert.DirExists(t, targetLibPath, "Target library path should exist")
	assert.DirExists(t, filepath.Join(targetLibPath, "actions"), "Actions directory should exist in target library")
	assert.DirExists(t, filepath.Join(targetLibPath, "templates"), "Templates directory should exist in target library")

	// Check that default files are still copied (from embedded defaults)
	assert.FileExists(t, filepath.Join(targetLibPath, "actions", "add-security-md.yaml"))
	assert.FileExists(t, filepath.Join(targetLibPath, "templates", "security.md.tmpl"))

	// Ensure no project-specific config is created in CWD (which is tempDir)
	assert.NoFileExists(t, filepath.Join(tempDir, config.DefaultConfigDir, config.DefaultConfigFileName))
}

// Helper to execute the set-global command
func executeSetGlobalCommand(t *testing.T, args []string, tempHomeDir string) (*cobra.Command, string, error) {
	t.Helper()

	// Override user home directory for this test
	originalUserHomeDir := os.Getenv("HOME")
	if tempHomeDir != "" {
		os.Setenv("HOME", tempHomeDir)
		// Ensure the GlobalConfigFilePath uses the overridden HOME
		// This might require a way to reset or re-initialize parts of your config package
		// if it caches the home directory path on startup.
		// For this example, we assume config.GlobalConfigFilePath() will pick up the new HOME.
	}
	defer func() {
		if tempHomeDir != "" {
			os.Setenv("HOME", originalUserHomeDir)
		}
	}()

	cmd := newSetGlobalCommand()
	cmd.SetArgs(args)

	// Capture output
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	err := cmd.Execute()
	
	// Combine both stdout and stderr for complete output
	output := outBuf.String() + errBuf.String()
	return cmd, output, err
}

// TestRunSetGlobalCommand_NoExistingConfig tests setting global library when no config exists.
func TestRunSetGlobalCommand_NoExistingConfig(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "darn_testhome_no_config_")
	assert.NoError(t, err)
	defer os.RemoveAll(tempHome)

	// Path for the new global library (doesn't need to exist for this part of the test)
	newLibPath := filepath.Join(tempHome, "my-global-darn-lib")
	_ = os.MkdirAll(newLibPath, 0755) // Create it so the "exists" message is correct

	_, _, err = executeSetGlobalCommand(t, []string{newLibPath}, tempHome)
	assert.NoError(t, err)

	// Verify ~/.darn/config.yaml was created in the tempHome
	expectedConfigPath := filepath.Join(tempHome, config.DefaultConfigDir, config.DefaultConfigFileName)
	assert.FileExists(t, expectedConfigPath)

	// Verify its contents
	cfgData, err := os.ReadFile(expectedConfigPath)
	assert.NoError(t, err)
	var globalCfg config.Config
	err = yaml.Unmarshal(cfgData, &globalCfg)
	assert.NoError(t, err)

	assert.Equal(t, newLibPath, globalCfg.LibraryPath)
	assert.True(t, globalCfg.UseGlobal)
	assert.False(t, globalCfg.UseLocal)
}

// TestRunSetGlobalCommand_UpdateExistingConfig tests updating an existing global config.
func TestRunSetGlobalCommand_UpdateExistingConfig(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "darn_testhome_update_")
	assert.NoError(t, err)
	defer os.RemoveAll(tempHome)

	// Create a dummy existing global config
	initialLibPath := filepath.Join(tempHome, "initial-global-lib")
	darnConfigDir := filepath.Join(tempHome, config.DefaultConfigDir)
	_ = os.MkdirAll(darnConfigDir, 0755)
	initialConfigPath := filepath.Join(darnConfigDir, config.DefaultConfigFileName)

	initialCfg := config.Config{
		LibraryPath:  initialLibPath,
		TemplatesDir: "original_templates",
		UseGlobal:    true,
		UseLocal:     true, // Deliberately different to check it gets set to false
	}
	cfgBytes, _ := yaml.Marshal(initialCfg)
	err = os.WriteFile(initialConfigPath, cfgBytes, 0644)
	assert.NoError(t, err)

	// New path to set
	updatedLibPath := filepath.Join(tempHome, "updated-global-darn-lib")
	_ = os.MkdirAll(updatedLibPath, 0755)

	_, _, err = executeSetGlobalCommand(t, []string{updatedLibPath}, tempHome)
	assert.NoError(t, err)

	// Verify config is updated
	assert.FileExists(t, initialConfigPath) // Should still be the same file
	updatedCfgData, err := os.ReadFile(initialConfigPath)
	assert.NoError(t, err)
	var updatedGlobalCfg config.Config
	err = yaml.Unmarshal(updatedCfgData, &updatedGlobalCfg)
	assert.NoError(t, err)

	assert.Equal(t, updatedLibPath, updatedGlobalCfg.LibraryPath)
	assert.True(t, updatedGlobalCfg.UseGlobal, "UseGlobal should be true after set-global")
	assert.False(t, updatedGlobalCfg.UseLocal, "UseLocal should be false after set-global")
	assert.Equal(t, "original_templates", updatedGlobalCfg.TemplatesDir, "Other settings should be preserved")
}

// TestRunSetGlobalCommand_PathExpansion tests tilde expansion for paths.
func TestRunSetGlobalCommand_PathExpansion(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "darn_testhome_expansion_")
	assert.NoError(t, err)
	defer os.RemoveAll(tempHome)

	// Path with tilde
	libPathWithTilde := "~/my_custom_darn_library" // This will resolve relative to tempHome due to env var override
	expectedExpandedPath := filepath.Join(tempHome, "my_custom_darn_library")
	_ = os.MkdirAll(expectedExpandedPath, 0755) // Create for "exists" check in command output

	_, _, err = executeSetGlobalCommand(t, []string{libPathWithTilde}, tempHome)
	assert.NoError(t, err)

	expectedConfigPath := filepath.Join(tempHome, config.DefaultConfigDir, config.DefaultConfigFileName)
	assert.FileExists(t, expectedConfigPath)
	cfgData, err := os.ReadFile(expectedConfigPath)
	assert.NoError(t, err)
	var globalCfg config.Config
	err = yaml.Unmarshal(cfgData, &globalCfg)
	assert.NoError(t, err)

	assert.Equal(t, expectedExpandedPath, globalCfg.LibraryPath, "Path should be expanded and absolute")
	assert.True(t, filepath.IsAbs(globalCfg.LibraryPath), "Stored path should be absolute")
}

// TestRunSetGlobalCommand_PathDoesNotExistInformational tests behavior when the target path doesn't exist.
func TestRunSetGlobalCommand_PathDoesNotExistInformational(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "darn_testhome_nonexistent_")
	assert.NoError(t, err)
	defer os.RemoveAll(tempHome)

	nonExistentLibPath := filepath.Join(tempHome, "non-existent-darn-lib") // Do NOT create this path

	_, output, err := executeSetGlobalCommand(t, []string{nonExistentLibPath}, tempHome)
	assert.NoError(t, err) // Command should still succeed

	// Verify config is created/updated
	expectedConfigPath := filepath.Join(tempHome, config.DefaultConfigDir, config.DefaultConfigFileName)
	assert.FileExists(t, expectedConfigPath)
	cfgData, err := os.ReadFile(expectedConfigPath)
	assert.NoError(t, err)
	var globalCfg config.Config
	err = yaml.Unmarshal(cfgData, &globalCfg)
	assert.NoError(t, err)
	assert.Equal(t, nonExistentLibPath, globalCfg.LibraryPath) // Path should still be set

	// Check for informational message in output
	assert.Contains(t, output, "does not exist (it may need to be initialized or created)", "Output should inform that the path does not exist")
}

// Helper to execute the library update command
func executeLibraryUpdateCommand(t *testing.T, commandArgs []string, tempHomeDir string, globalConfigContent *config.Config) (string, error) {
	t.Helper()

	originalUserHomeDir := os.Getenv("HOME")
	if tempHomeDir != "" {
		os.Setenv("HOME", tempHomeDir)
		if globalConfigContent != nil {
			// Create the global config file if content is provided
			globalConfigDir := filepath.Join(tempHomeDir, config.DefaultConfigDir)
			err := os.MkdirAll(globalConfigDir, 0755)
			assert.NoError(t, err)
			globalConfigPath := filepath.Join(globalConfigDir, config.DefaultConfigFileName)
			cfgBytes, err := yaml.Marshal(globalConfigContent)
			assert.NoError(t, err)
			err = os.WriteFile(globalConfigPath, cfgBytes, 0644)
			assert.NoError(t, err)
		}
	}
	defer func() {
		if tempHomeDir != "" {
			os.Setenv("HOME", originalUserHomeDir)
		}
	}()

	cmd := newUpdateCommand() // Assuming newUpdateCommand() exists and is the constructor
	cmd.SetArgs(commandArgs)

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	err := cmd.Execute()
	
	// Combine both stdout and stderr for complete output
	output := outBuf.String() + errBuf.String()
	return output, err
}

// TestRunUpdateCommand_NoPathFlag_GlobalConfigActive tests `darn library update`
// when --library-path is not used, but an active global library is set in ~/.darn/config.yaml.
func TestRunUpdateCommand_NoPathFlag_GlobalConfigActive(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "darn_update_activeglobal_home_")
	assert.NoError(t, err)
	defer os.RemoveAll(tempHome)

	// Prepare the target library directory that global config will point to
	activeGlobalLibPath := filepath.Join(tempHome, "my-active-global-library")
	err = os.MkdirAll(filepath.Join(activeGlobalLibPath, "actions"), 0755) // Ensure "actions" subdir exists
	assert.NoError(t, err)

	// Prepare global config pointing to this library
	globalCfg := &config.Config{
		LibraryPath: activeGlobalLibPath,
		UseGlobal:   true,
	}

	// Prepare a source directory with proper library structure
	sourceFilesDir, err := os.MkdirTemp("", "darn_update_source_")
	assert.NoError(t, err)
	defer os.RemoveAll(sourceFilesDir)
	
	// Create actions subdirectory and file
	sourceActionsDir := filepath.Join(sourceFilesDir, "actions")
	err = os.MkdirAll(sourceActionsDir, 0755)
	assert.NoError(t, err)
	sourceActionFile := filepath.Join(sourceActionsDir, "test_action.yaml")
	err = os.WriteFile(sourceActionFile, []byte("name: test_action\ndescription: A test action."), 0644)
	assert.NoError(t, err)

	// Execute 'darn library update <sourceDir>' (implicitly uses global config)
	// Add --verbose to parse output for targeted path
	output, err := executeLibraryUpdateCommand(t, []string{sourceFilesDir, "--verbose"}, tempHome, globalCfg)
	assert.NoError(t, err)

	// Verify from output that it targets the activeGlobalLibPath
	assert.Contains(t, output, "Attempting to update active global library (from ~/.darn/config.yaml or default ~/.darn/library)...", "Output should indicate attempt to use global config")
	assert.Contains(t, output, fmt.Sprintf("Updating active global library configured at: %s", activeGlobalLibPath), "Output should confirm update of active global library")

	// Verify file was copied to the active global library
	expectedDestFile := filepath.Join(activeGlobalLibPath, "actions", "test_action.yaml")
	assert.FileExists(t, expectedDestFile, "Test action file should be copied to the active global library's actions directory")
}

// TestRunUpdateCommand_NoPathFlag_NoGlobalConfig tests `darn library update`
// when --library-path is not used and no global config exists (or it's empty).
func TestRunUpdateCommand_NoPathFlag_NoGlobalConfig(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "darn_update_noglobal_home_")
	assert.NoError(t, err)
	defer os.RemoveAll(tempHome)

	// Prepare the default global library directory
	defaultGlobalLibPath := config.ExpandPathWithTilde(config.DefaultGlobalLibrary) // This will be under tempHome
	err = os.MkdirAll(filepath.Join(defaultGlobalLibPath, "actions"), 0755)
	assert.NoError(t, err)

	// Prepare a source directory with proper library structure
	sourceFilesDir, err := os.MkdirTemp("", "darn_update_source_")
	assert.NoError(t, err)
	defer os.RemoveAll(sourceFilesDir)
	
	// Create actions subdirectory and file
	sourceActionsDir := filepath.Join(sourceFilesDir, "actions")
	err = os.MkdirAll(sourceActionsDir, 0755)
	assert.NoError(t, err)
	sourceActionFile := filepath.Join(sourceActionsDir, "another_action.yaml")
	err = os.WriteFile(sourceActionFile, []byte("name: another_action\ndescription: Another test action."), 0644)
	assert.NoError(t, err)

	// Execute 'darn library update <sourceDir>' (no global config file provided to helper)
	output, err := executeLibraryUpdateCommand(t, []string{sourceFilesDir, "--verbose"}, tempHome, nil)
	assert.NoError(t, err)

	// Verify from output that it targets the default global library path (this will use tempHome)
	expectedPath := filepath.Join(tempHome, ".darn", "library")
	assert.Contains(t, output, fmt.Sprintf("Updating active global library configured at: %s", expectedPath), "Output should confirm update of configured global library")

	// Verify file was copied to the expected global library path
	expectedDestFile := filepath.Join(expectedPath, "actions", "another_action.yaml")
	assert.FileExists(t, expectedDestFile, "Test action file should be copied to the global library's actions directory")
}

// TestRunUpdateCommand_WithPathFlag tests `darn library update --library-path /custom/path`.
func TestRunUpdateCommand_WithPathFlag(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "darn_update_withpath_home_") // Home dir, though not strictly used for config in this test
	assert.NoError(t, err)
	defer os.RemoveAll(tempHome)

	// Prepare the custom library directory specified by the flag
	customLibTargetDir, err := os.MkdirTemp("", "darn_update_customtarget_")
	assert.NoError(t, err)
	defer os.RemoveAll(customLibTargetDir)
	err = os.MkdirAll(filepath.Join(customLibTargetDir, "actions"), 0755)
	assert.NoError(t, err)

	// Prepare a source directory with proper library structure
	sourceFilesDir, err := os.MkdirTemp("", "darn_update_source_")
	assert.NoError(t, err)
	defer os.RemoveAll(sourceFilesDir)
	
	// Create actions subdirectory and file
	sourceActionsDir := filepath.Join(sourceFilesDir, "actions")
	err = os.MkdirAll(sourceActionsDir, 0755)
	assert.NoError(t, err)
	sourceActionFile := filepath.Join(sourceActionsDir, "custom_action.yaml")
	err = os.WriteFile(sourceActionFile, []byte("name: custom_action\ndescription: A custom test action."), 0644)
	assert.NoError(t, err)

	// Execute 'darn library update --library-path <customLibTargetDir> <sourceDir>'
	output, err := executeLibraryUpdateCommand(t, []string{"--library-path", customLibTargetDir, sourceFilesDir, "--verbose"}, tempHome, nil)
	assert.NoError(t, err)

	// Verify from output that it targets the customLibTargetDir
	assert.Contains(t, output, fmt.Sprintf("Updating library specified by --library-path flag: %s", customLibTargetDir))

	// Verify file was copied to the custom library
	expectedDestFile := filepath.Join(customLibTargetDir, "actions", "custom_action.yaml")
	assert.FileExists(t, expectedDestFile, "Test action file should be copied to the custom library's actions directory")
}
