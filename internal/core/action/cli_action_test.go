// SPDX-License-Identifier: Apache-2.0

package action_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kusari-oss/darn/internal/core/action"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCLIAction(t *testing.T) {
	tests := []struct {
		name        string
		config      action.Config
		shouldError bool
	}{
		{
			name: "valid config",
			config: action.Config{
				Name:        "test-action",
				Type:        "cli",
				Description: "Test CLI action",
				Command:     "echo",
				Args:        []string{"Hello, World!"},
			},
			shouldError: false,
		},
		{
			name: "missing command",
			config: action.Config{
				Name:        "test-action",
				Type:        "cli",
				Description: "Test CLI action",
				// Command is missing
				Args: []string{"Hello, World!"},
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			act, err := action.NewCLIAction(tt.config)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Nil(t, act)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, act)
				// Check that the action has the correct type
				assert.Implements(t, (*action.Action)(nil), act)
				assert.Equal(t, tt.config.Description, act.Description())
			}
		})
	}
}

func TestNewOutputCLIAction(t *testing.T) {
	// Create a valid config with outputs
	config := action.Config{
		Name:        "test-output-action",
		Type:        "cli",
		Description: "Test CLI action with outputs",
		Command:     "echo",
		Args:        []string{"Hello, World!"},
		Outputs: map[string]interface{}{
			"greeting": map[string]interface{}{
				"format":  "text",
				"pattern": "(Hello.*)",
			},
		},
	}

	// Test creation
	act, err := action.NewOutputCLIAction(config)
	assert.NoError(t, err)
	assert.NotNil(t, act)

	// Verify it implements the OutputAction interface
	_, ok := act.(action.OutputAction)
	assert.True(t, ok, "Action should implement OutputAction interface")
}

// This test requires the ability to execute a real command
func TestCLIActionExecute(t *testing.T) {
	// Create a temporary file to write output to
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.txt")

	// Create the CLI action config
	config := action.Config{
		Name:        "test-file-write",
		Type:        "cli",
		Description: "Test writing to a file",
		Command:     "bash",
		Args:        []string{"-c", "echo '{{.content}}' > {{.output_file}}"},
	}

	// Create the action
	act, err := action.NewCLIAction(config)
	require.NoError(t, err)

	// Execute the action
	params := map[string]interface{}{
		"content":     "Hello, World!",
		"output_file": outputFile,
	}

	err = act.Execute(params)
	assert.NoError(t, err)

	// Verify the output file was created with the correct content
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Hello, World!")
}

// Mock implementation for testing CLI actions without actual command execution
type MockExecutor struct {
	ExecuteCalled bool
	ReturnError   error
	ReturnOutput  []byte
	LastCommand   string
	LastArgs      []string
	LastParams    map[string]interface{}
}

func (m *MockExecutor) Execute(command string, args []string, params map[string]interface{}) ([]byte, error) {
	m.ExecuteCalled = true
	m.LastCommand = command
	m.LastArgs = args
	m.LastParams = params
	return m.ReturnOutput, m.ReturnError
}

// TODO: Implement a test for the CLI action using a mock executor
