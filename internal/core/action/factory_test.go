// SPDX-License-Identifier: Apache-2.0

package action_test

import (
	"testing"

	"github.com/kusari-oss/darn/internal/core/action"
	"github.com/kusari-oss/darn/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFactory(t *testing.T) {
	// Create action context
	context := action.ActionContext{
		TemplatesDir:       "/tmp/templates",
		GlobalTemplatesDir: "/tmp/global/templates",
		WorkingDir:         "/tmp/work",
		VerboseMode:        false,
		UseLocal:           true,
		UseGlobal:          false,
	}

	// Create factory
	factory := action.NewFactory(context)

	// Register a test action creator
	factory.Register("test", testutil.NewMockActionCreator())

	// Test creating an action
	config := action.Config{
		Name:        "test-action",
		Type:        "test",
		Description: "Test action",
	}

	act, err := factory.Create(config)
	require.NoError(t, err)
	require.NotNil(t, act)

	// Verify the action was created with the correct configuration
	mockAction, ok := act.(*testutil.MockAction)
	require.True(t, ok)
	assert.Equal(t, config, mockAction.Config)
	assert.Equal(t, context, mockAction.Context)

	// Test creating an action with unknown type
	unknownConfig := action.Config{
		Name:        "unknown-action",
		Type:        "unknown",
		Description: "Unknown action type",
	}

	act, err = factory.Create(unknownConfig)
	assert.Error(t, err)
	assert.Nil(t, act)
	assert.Contains(t, err.Error(), "unknown action type")
}

func TestRegisterDefaultTypes(t *testing.T) {
	// Create action context
	context := action.ActionContext{
		TemplatesDir:       "/tmp/templates",
		GlobalTemplatesDir: "/tmp/global/templates",
		WorkingDir:         "/tmp/work",
		VerboseMode:        false,
		UseLocal:           true,
		UseGlobal:          false,
	}

	// Create factory and register default types
	factory := action.NewFactory(context)
	factory.RegisterDefaultTypes()

	// Test creating a file action
	fileConfig := action.Config{
		Name:         "test-file",
		Type:         "file",
		Description:  "Test file action",
		TemplatePath: "template.tmpl",
		TargetPath:   "output.txt",
	}

	fileAction, err := factory.Create(fileConfig)
	require.NoError(t, err)
	require.NotNil(t, fileAction)
	assert.Equal(t, "Test file action", fileAction.Description())

	// Test creating a CLI action
	cliConfig := action.Config{
		Name:        "test-cli",
		Type:        "cli",
		Description: "Test CLI action",
		Command:     "echo",
		Args:        []string{"Hello, World!"},
	}

	cliAction, err := factory.Create(cliConfig)
	require.NoError(t, err)
	require.NotNil(t, cliAction)
	assert.Equal(t, "Test CLI action", cliAction.Description())

	// Test validation for file action (missing required fields)
	invalidFileConfig := action.Config{
		Name:        "invalid-file",
		Type:        "file",
		Description: "Invalid file action",
		// Missing TemplatePath and TargetPath
	}

	invalidAction, err := factory.Create(invalidFileConfig)
	assert.Error(t, err)
	assert.Nil(t, invalidAction)
	assert.Contains(t, err.Error(), "template_path is required")

	// Test validation for CLI action (missing required fields)
	invalidCliConfig := action.Config{
		Name:        "invalid-cli",
		Type:        "cli",
		Description: "Invalid CLI action",
		// Missing Command
	}

	invalidAction, err = factory.Create(invalidCliConfig)
	assert.Error(t, err)
	assert.Nil(t, invalidAction)
	assert.Contains(t, err.Error(), "command is required")
}

func TestUpdateContext(t *testing.T) {
	// Create initial context
	initialContext := action.ActionContext{
		TemplatesDir:       "/tmp/templates",
		GlobalTemplatesDir: "/tmp/global/templates",
		WorkingDir:         "/tmp/work",
		VerboseMode:        false,
	}

	// Create factory with initial context
	factory := action.NewFactory(initialContext)

	// Register a test action creator
	factory.Register("test", testutil.NewMockActionCreator())

	// Update the context
	updatedContext := action.ActionContext{
		TemplatesDir:       "/tmp/new/templates",
		GlobalTemplatesDir: "/tmp/new/global/templates",
		WorkingDir:         "/tmp/new/work",
		VerboseMode:        true,
	}
	factory.UpdateContext(updatedContext)

	// Create an action with the updated context
	config := action.Config{
		Name:        "test-action",
		Type:        "test",
		Description: "Test action",
	}

	act, err := factory.Create(config)
	require.NoError(t, err)

	// Verify the action was created with the updated context
	mockAction, ok := act.(*testutil.MockAction)
	require.True(t, ok)
	assert.Equal(t, updatedContext, mockAction.Context)
	assert.Equal(t, "/tmp/new/templates", mockAction.Context.TemplatesDir)
	assert.Equal(t, true, mockAction.Context.VerboseMode)
}
