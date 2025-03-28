// SPDX-License-Identifier: Apache-2.0

package action

// Action defines the interface that all actions must implement
type Action interface {
	// Execute runs the action with the given parameters
	Execute(params map[string]interface{}) error

	// Description returns a human-readable description of the action
	Description() string
}

// OutputAction extends Action to support returning outputs
type OutputAction interface {
	Action

	// ExecuteWithOutput runs the action and returns outputs
	ExecuteWithOutput(params map[string]interface{}) (map[string]interface{}, error)
}

// TODO: Move Template, Args to an action type specific struct for each action type.
// Config holds the configuration for an action
type Config struct {
	Name         string                 `yaml:"name"`
	Type         string                 `yaml:"type"`
	Description  string                 `yaml:"description"`
	Labels       map[string][]string    `yaml:"labels,omitempty"`
	TemplatePath string                 `yaml:"template_path,omitempty"`
	TargetPath   string                 `yaml:"target_path,omitempty"`
	CreateDirs   bool                   `yaml:"create_dirs,omitempty"`
	Command      string                 `yaml:"command,omitempty"`
	Args         []string               `yaml:"args,omitempty"`
	Schema       map[string]interface{} `yaml:"schema"`
	Defaults     map[string]interface{} `yaml:"defaults,omitempty"`
	Outputs      interface{}            `yaml:"outputs,omitempty"`
}

// LoadConfig loads a Config from a map of data
func LoadConfig(data map[string]interface{}) (Config, error) {
	var config Config

	// Map fields from the data to the config
	if name, ok := data["name"].(string); ok {
		config.Name = name
	}

	if typeName, ok := data["type"].(string); ok {
		config.Type = typeName
	}

	if description, ok := data["description"].(string); ok {
		config.Description = description
	}

	// Handle labels
	if labels, ok := data["labels"].(map[string]interface{}); ok {
		config.Labels = make(map[string][]string)
		for key, value := range labels {
			if valueSlice, ok := value.([]interface{}); ok {
				strSlice := make([]string, 0, len(valueSlice))
				for _, v := range valueSlice {
					if strVal, ok := v.(string); ok {
						strSlice = append(strSlice, strVal)
					}
				}
				config.Labels[key] = strSlice
			}
		}
	}

	// Handle template-related fields
	if templatePath, ok := data["template_path"].(string); ok {
		config.TemplatePath = templatePath
	}

	if targetPath, ok := data["target_path"].(string); ok {
		config.TargetPath = targetPath
	}

	if createDirs, ok := data["create_dirs"].(bool); ok {
		config.CreateDirs = createDirs
	}

	// Handle command-related fields
	if command, ok := data["command"].(string); ok {
		config.Command = command
	}

	if argsData, ok := data["args"].([]interface{}); ok {
		config.Args = make([]string, 0, len(argsData))
		for _, arg := range argsData {
			if strArg, ok := arg.(string); ok {
				config.Args = append(config.Args, strArg)
			}
		}
	}

	// Copy schema and defaults as is
	if schema, ok := data["schema"].(map[string]interface{}); ok {
		config.Schema = schema
	}

	if defaults, ok := data["defaults"].(map[string]interface{}); ok {
		config.Defaults = defaults
	}

	// Copy outputs as is
	if outputs, ok := data["outputs"]; ok {
		config.Outputs = outputs
	}

	return config, nil
}
