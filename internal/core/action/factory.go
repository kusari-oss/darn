// SPDX-License-Identifier: Apache-2.0

package action

import (
	"fmt"
)

// ActionCreator is a function that creates a darn action from a configuration
type ActionCreator func(Config, ActionContext) (Action, error)

// ActionContext provides contextual information for action creation
type ActionContext struct {
	TemplatesDir       string
	GlobalTemplatesDir string
	WorkingDir         string
	VerboseMode        bool
	UseLocal           bool
	UseGlobal          bool
	GlobalFirst        bool
}

// Factory creates actions of different types
type Factory struct {
	actionCreators map[string]ActionCreator
	context        ActionContext
}

// NewFactory creates a new action factory with the given context
func NewFactory(context ActionContext) *Factory {
	return &Factory{
		actionCreators: make(map[string]ActionCreator),
		context:        context,
	}
}

// Register registers a new action type creator
func (f *Factory) Register(typeName string, creator ActionCreator) {
	f.actionCreators[typeName] = creator
}

// Create creates an action of the specified type
func (f *Factory) Create(config Config) (Action, error) {
	creator, ok := f.actionCreators[config.Type]
	if !ok {
		return nil, fmt.Errorf("unknown action type: %s", config.Type)
	}

	return creator(config, f.context)
}

// RegisterDefaultTypes registers all the standard action types
func (f *Factory) RegisterDefaultTypes() {
	// File action creator
	f.Register("file", func(config Config, context ActionContext) (Action, error) {
		if config.TemplatePath == "" {
			return nil, fmt.Errorf("template_path is required for file actions")
		}

		if config.TargetPath == "" {
			return nil, fmt.Errorf("target_path is required for file actions")
		}

		// Create a FileAction that knows about both template locations
		return &FileAction{
			config:             config,
			templatesDir:       context.TemplatesDir,
			globalTemplatesDir: context.GlobalTemplatesDir,
			useLocal:           context.UseLocal,
			useGlobal:          context.UseGlobal,
			globalFirst:        context.GlobalFirst,
		}, nil
	})

	// CLI action creator
	f.Register("cli", func(config Config, context ActionContext) (Action, error) {
		// Handle both regular and output CLI actions based on config
		if config.Outputs != nil {
			return NewOutputCLIAction(config)
		}
		return NewCLIAction(config)
	})
}

// UpdateContext updates the factory's context
func (f *Factory) UpdateContext(context ActionContext) {
	f.context = context
}
