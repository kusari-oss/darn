// SPDX-License-Identifier: Apache-2.0

package action_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kusari-oss/darn/internal/core/action"
	"github.com/kusari-oss/darn/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test template file
func createTestTemplate(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err, "Failed to create test template file")
	return path
}

func TestFileAction(t *testing.T) {
	// Create temporary directories for templates and output
	tempDir := t.TempDir()
	templatesDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "output")

	err := os.MkdirAll(templatesDir, 0755)
	require.NoError(t, err)

	err = os.MkdirAll(outputDir, 0755)
	require.NoError(t, err)

	// Create a test template
	templateName := "security.md.tmpl"
	templateContent := "# Security Policy for {{.name}}\n\nEmails:\n{{- range .emails}}\n- {{.}}\n{{- end}}"
	createTestTemplate(t, templatesDir, templateName, templateContent)

	// Create a file action
	config := action.Config{
		Name:         "add-security-md",
		Type:         "file",
		Description:  "Add SECURITY.md file",
		TemplatePath: templateName,
		TargetPath:   filepath.Join(outputDir, "SECURITY.md"),
		CreateDirs:   true,
		Schema: map[string]interface{}{
			"type":     "object",
			"required": []interface{}{"name", "emails"},
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
				"emails": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
	}

	// Create the action context
	context := action.ActionContext{
		TemplatesDir: templatesDir,
		WorkingDir:   tempDir,
		VerboseMode:  true,
		UseLocal:     true,
	}

	// Create the action using factory pattern (similar to actual code)
	factory := action.NewFactory(context)
	factory.Register("file", func(cfg action.Config, ctx action.ActionContext) (action.Action, error) {
		fileAction := createTestFileAction(t, cfg, ctx.TemplatesDir, ctx.WorkingDir)
		return fileAction, nil
	})

	act, err := factory.Create(config)
	require.NoError(t, err)

	// Execute the action
	params := map[string]interface{}{
		"name":   "Test Project",
		"emails": []interface{}{"security@example.com", "admin@example.com"},
	}

	err = act.Execute(params)
	assert.NoError(t, err)

	// Verify the output file was created
	outputPath := filepath.Join(outputDir, "SECURITY.md")
	assert.FileExists(t, outputPath)

	// Verify the content of the file
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	expectedContent := "# Security Policy for Test Project\n\nEmails:\n- security@example.com\n- admin@example.com"
	assert.Equal(t, expectedContent, string(content))
}

func TestFileActionWithOutputs(t *testing.T) {
	// Create a mock output action
	mockOutputAction := new(testutil.MockOutputAction)

	// Set expectations for the mock
	params := map[string]interface{}{
		"name": "TestProject",
	}

	expectedOutputs := map[string]interface{}{
		"file_path": "output/TestProject-SECURITY.md",
	}

	mockOutputAction.On("ExecuteWithOutput", params).Return(expectedOutputs, nil)
	mockOutputAction.On("Description").Return("Add SECURITY.md file")

	// Use the mock
	outputs, err := mockOutputAction.ExecuteWithOutput(params)
	assert.NoError(t, err)
	description := mockOutputAction.Description()
	assert.Equal(t, "Add SECURITY.md file", description)

	// Verify the outputs
	assert.Equal(t, expectedOutputs, outputs)
	assert.Equal(t, "output/TestProject-SECURITY.md", outputs["file_path"])

	// Verify expectations were met
	mockOutputAction.AssertExpectations(t)

	// Also do a real test to verify interface compatibility
	tempDir := t.TempDir()
	templatesDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "output")

	err = os.MkdirAll(templatesDir, 0755)
	require.NoError(t, err)

	err = os.MkdirAll(outputDir, 0755)
	require.NoError(t, err)

	templateName := "security.md.tmpl"
	templateContent := "# Security Policy for {{.name}}"
	createTestTemplate(t, templatesDir, templateName, templateContent)

	config := action.Config{
		Name:         "add-security-md",
		Type:         "file",
		Description:  "Add SECURITY.md file",
		TemplatePath: templateName,
		TargetPath:   filepath.Join(outputDir, "{{.name}}-SECURITY.md"),
		CreateDirs:   true,
	}

	fileAction := createTestFileAction(t, config, templatesDir, outputDir)
	outputAction, ok := interface{}(fileAction).(action.OutputAction)
	require.True(t, ok, "FileAction should implement OutputAction interface")

	realOutputs, err := outputAction.ExecuteWithOutput(params)
	assert.NoError(t, err)
	assert.Contains(t, realOutputs, "file_path")
}

func TestFileActionValidation(t *testing.T) {
	// Create a mock action
	mockAction := new(testutil.MockAction)

	// Set up expectations for different validation scenarios
	validParams := map[string]interface{}{
		"name":   "Test Project",
		"emails": []string{"security@example.com"},
	}

	invalidParams := map[string]interface{}{
		"name": "Test Project",
		// Missing "emails" parameter
	}

	wrongTypeParams := map[string]interface{}{
		"name":   "Test Project",
		"emails": "security@example.com", // String instead of array
	}

	// Set expectations
	mockAction.On("Execute", validParams).Return(nil)
	mockAction.On("Execute", invalidParams).Return(assert.AnError)
	mockAction.On("Execute", wrongTypeParams).Return(assert.AnError)

	// Test with valid parameters
	err := mockAction.Execute(validParams)
	assert.NoError(t, err)

	// Test with missing parameter
	err = mockAction.Execute(invalidParams)
	assert.Error(t, err)

	// Test with wrong type
	err = mockAction.Execute(wrongTypeParams)
	assert.Error(t, err)

	// Verify all expectations were met
	mockAction.AssertExpectations(t)

	// TODO: Add a test to create a real file along with cleanup.
	// Now also test with real implementation but force validation errors
	// Create directory with a non-existent template path to isolate validation errors
	tempDir := t.TempDir()

	config := action.Config{
		Name:         "add-security-md",
		Type:         "file",
		Description:  "Add SECURITY.md file",
		TemplatePath: "nonexistent.tmpl", // This will not be found
		TargetPath:   "SECURITY.md",
		Schema: map[string]interface{}{
			"type":     "object",
			"required": []interface{}{"name", "emails"},
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
				"emails": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
	}

	fileAction := createTestFileAction(t, config, tempDir, tempDir)

	// Test missing parameter
	err = fileAction.Execute(invalidParams)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parameter validation failed")

	// Test wrong type
	err = fileAction.Execute(wrongTypeParams)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parameter 'emails' (emails) must be an array")
}

func createTestFileAction(t *testing.T, config action.Config, templatesDir, workingDir string) *action.FileAction {
	// Use testify's require for assertions that would fail the test immediately
	require := require.New(t)

	context := action.ActionContext{
		TemplatesDir:       templatesDir,
		GlobalTemplatesDir: templatesDir, // Often the same in tests
		WorkingDir:         workingDir,
		VerboseMode:        true,
		UseLocal:           true,
	}

	factory := action.NewFactory(context)
	factory.RegisterDefaultTypes()

	// Create action through factory
	act, err := factory.Create(config)
	require.NoError(err)

	// Type assertion to get the FileAction
	fileAction, ok := act.(*action.FileAction)
	require.True(ok, "Expected FileAction type")

	return fileAction
}
