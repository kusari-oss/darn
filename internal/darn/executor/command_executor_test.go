// SPDX-License-Identifier: Apache-2.0

package executor_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kusari-oss/darn/internal/darn/executor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandExecutor(t *testing.T) {
	// Skip tests if running on Windows because the commands are different
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}

	// Create a temporary directory for output files
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.txt")

	// Create a test file that will be used for testing file existence
	testFile := filepath.Join(tempDir, "test-file.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err, "Failed to create test file")

	// Test cases for CommandExecutor
	tests := []struct {
		name        string
		command     string
		args        []string
		params      map[string]interface{}
		workingDir  string
		shouldError bool
		outputCheck func(t *testing.T, result *executor.CommandResult)
	}{
		{
			name:    "echo command",
			command: "echo",
			args:    []string{"Hello, {{.name}}!"},
			params: map[string]interface{}{
				"name": "World",
			},
			shouldError: false,
			outputCheck: func(t *testing.T, result *executor.CommandResult) {
				assert.Contains(t, string(result.Output), "Hello, World!")
			},
		},
		{
			name:    "write to file",
			command: "bash",
			args:    []string{"-c", "echo '{{.content}}' > {{.output_file}}"},
			params: map[string]interface{}{
				"content":     "File content",
				"output_file": outputFile,
			},
			shouldError: false,
			outputCheck: func(t *testing.T, result *executor.CommandResult) {
				// Check that the file was created
				content, err := os.ReadFile(outputFile)
				assert.NoError(t, err, "Failed to read output file")
				assert.Contains(t, string(content), "File content")
			},
		},
		{
			name:    "command with working directory",
			command: "pwd",
			args:    []string{},
			params: map[string]interface{}{
				"working_dir": tempDir,
			},
			shouldError: false,
			outputCheck: func(t *testing.T, result *executor.CommandResult) {
				// Should output the tempDir path
				assert.Contains(t, string(result.Output), tempDir)
			},
		},
		{
			name:    "command to check file existence",
			command: "ls",
			args:    []string{"-l", "{{.file_path}}"},
			params: map[string]interface{}{
				"file_path": testFile,
			},
			workingDir:  tempDir,
			shouldError: false,
			outputCheck: func(t *testing.T, result *executor.CommandResult) {
				// Output should contain the test file name
				assert.Contains(t, string(result.Output), "test-file.txt")
			},
		},
		{
			name:    "nonexistent command",
			command: "thiscommanddoesnotexist",
			args:    []string{},
			params:  map[string]interface{}{},
			// This should error because the command doesn't exist
			shouldError: true,
			outputCheck: nil,
		},
		{
			name:    "command with environment variables",
			command: "bash",
			args:    []string{"-c", "echo $TEST_VAR"},
			params: map[string]interface{}{
				"environment": []interface{}{"TEST_VAR={{.test_value}}"},
				"test_value":  "environment test",
			},
			shouldError: false,
			outputCheck: func(t *testing.T, result *executor.CommandResult) {
				assert.Contains(t, string(result.Output), "environment test")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create command executor
			cmdExecutor := executor.NewCommandExecutor(tt.command, tt.args)

			// Set working directory if specified
			if tt.workingDir != "" {
				cmdExecutor.WithWorkingDir(tt.workingDir)
			}

			// Enable verbose mode for testing
			cmdExecutor.WithVerbose(true)

			// Process parameters
			err := cmdExecutor.ProcessParameters(tt.params)
			require.NoError(t, err, "Failed to process parameters")

			// Execute the command
			result, err := cmdExecutor.Execute()

			if tt.shouldError {
				assert.Error(t, err, "Expected error for command: %s", tt.command)
			} else {
				assert.NoError(t, err, "Unexpected error for command: %s %v", tt.command, tt.args)

				// Run output checks if specified
				if tt.outputCheck != nil {
					tt.outputCheck(t, result)
				}
			}
		})
	}
}

func TestCommandExecutorWithInvalidTemplates(t *testing.T) {
	// Test with missing parameter in template
	cmdExecutor := executor.NewCommandExecutor("echo", []string{"Hello, {{.missing_param}}!"})

	err := cmdExecutor.ProcessParameters(map[string]interface{}{
		"name": "World", // Parameter doesn't match template
	})

	assert.Error(t, err, "Expected error for missing parameter")
	assert.Contains(t, err.Error(), "error processing argument", "Error should indicate issue with processing")
}

func TestCommandExecutorWithMultipleArguments(t *testing.T) {
	// Skip tests if running on Windows
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}

	// Test with multiple arguments that have templates
	cmdExecutor := executor.NewCommandExecutor("echo", []string{
		"{{.first}}",
		"{{.second}}",
		"{{.third}}",
	})

	err := cmdExecutor.ProcessParameters(map[string]interface{}{
		"first":  "one",
		"second": "two",
		"third":  "three",
	})
	require.NoError(t, err, "Failed to process parameters")

	result, err := cmdExecutor.Execute()
	assert.NoError(t, err, "Failed to execute command")
	assert.Contains(t, string(result.Output), "one two three")
}
