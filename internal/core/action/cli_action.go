// SPDX-License-Identifier: Apache-2.0

package action

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kusari-oss/darn/internal/darn/executor"
)

// CLIAction executes a command line tool using composition
type CLIAction struct {
	config        Config
	outputParsers map[string]func([]byte) (interface{}, error)
}

// NewCLIAction creates a new CLI action
func NewCLIAction(config Config) (Action, error) {
	// Validate required fields
	if config.Command == "" {
		return nil, fmt.Errorf("command is required for CLI actions")
	}

	/*if config.Args == nil {
		return nil, fmt.Errorf("args is required for CLI actions")
	}*/

	// Initialize with no output parsers by default
	return &CLIAction{
		config:        config,
		outputParsers: make(map[string]func([]byte) (interface{}, error)),
	}, nil
}

// NewOutputCLIAction creates a CLI action with output parsers
func NewOutputCLIAction(config Config) (Action, error) {
	// Create the base CLI action first
	action, err := NewCLIAction(config)
	if err != nil {
		return nil, err
	}

	cliAction := action.(*CLIAction)

	// Add output parsers
	if outputsConfig, ok := config.Outputs.(map[string]interface{}); ok {
		for outputName, parserConfig := range outputsConfig {
			if parserMap, ok := parserConfig.(map[string]interface{}); ok {
				if format, ok := parserMap["format"].(string); ok {
					switch format {
					case "json":
						// JSON parser with optional jq-like path
						path, _ := parserMap["path"].(string)
						cliAction.outputParsers[outputName] = createJSONParser(path)
					case "text":
						// Text parser with optional regex
						pattern, _ := parserMap["pattern"].(string)
						cliAction.outputParsers[outputName] = createTextParser(pattern)
					}
				}
			}
		}
	}

	return cliAction, nil
}

// Execute runs the CLI action
func (a *CLIAction) Execute(params map[string]interface{}) error {
	// Create command com_executor
	com_executor := executor.NewCommandExecutor(a.config.Command, a.config.Args)

	// Get verbose setting from params (default to false)
	verbose, _ := params["verbose"].(bool)
	com_executor.WithVerbose(verbose)

	// Process parameters
	if err := com_executor.ProcessParameters(params); err != nil {
		return err
	}

	// Execute the command
	_, err := com_executor.Execute()
	if err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}

// ExecuteWithOutput runs the CLI action and captures outputs
func (a *CLIAction) ExecuteWithOutput(params map[string]interface{}) (map[string]interface{}, error) {
	// Create command com_executor
	com_executor := executor.NewCommandExecutor(a.config.Command, a.config.Args)

	// Get verbose setting from params (default to false)
	verbose, _ := params["verbose"].(bool)
	com_executor.WithVerbose(verbose)

	// Process parameters
	if err := com_executor.ProcessParameters(params); err != nil {
		return nil, err
	}

	// Execute the command
	result, err := com_executor.Execute()
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}

	// Process outputs if there are parsers
	outputs := make(map[string]interface{})

	if len(a.outputParsers) > 0 {
		// Get output bytes
		outputBytes := result.Output

		// Apply each parser
		for outputName, parser := range a.outputParsers {
			value, err := parser(outputBytes)
			if err != nil {
				fmt.Printf("Warning: Failed to parse output %s: %v\n", outputName, err)
				continue
			}
			outputs[outputName] = value
		}
	}

	return outputs, nil
}

// Description returns the action description
func (a *CLIAction) Description() string {
	if a.config.Description != "" {
		return a.config.Description
	}
	return "Execute a command line tool"
}

// Helper functions for output parsing
func createJSONParser(path string) func([]byte) (interface{}, error) {
	return func(data []byte) (interface{}, error) {
		var result interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err
		}

		// If path is specified, extract the nested value
		if path != "" {
			return extractJSONPath(result, path)
		}

		return result, nil
	}
}

func createTextParser(pattern string) func([]byte) (interface{}, error) {
	return func(data []byte) (interface{}, error) {
		text := string(data)

		// If pattern is specified, extract with regex
		if pattern != "" {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return nil, err
			}

			matches := re.FindStringSubmatch(text)
			if len(matches) > 1 {
				return matches[1], nil // Return first capture group
			} else if len(matches) == 1 {
				return matches[0], nil // Return full match
			}
			return nil, fmt.Errorf("no matches found for pattern: %s", pattern)
		}

		// Otherwise return the whole text, trimmed
		return strings.TrimSpace(text), nil
	}
}

// Helper to extract a value from a JSON object using a dotted path
func extractJSONPath(obj interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	current := obj

	for _, part := range parts {
		// Handle array access
		if strings.Contains(part, "[") && strings.Contains(part, "]") {
			// Split into property name and array index
			openBracket := strings.Index(part, "[")
			closeBracket := strings.Index(part, "]")

			if openBracket > 0 && closeBracket > openBracket {
				propName := part[:openBracket]
				indexStr := part[openBracket+1 : closeBracket]

				// Get the object property
				if mapObj, ok := current.(map[string]interface{}); ok {
					current = mapObj[propName]
				} else {
					return nil, fmt.Errorf("not an object at path: %s", propName)
				}

				// Parse index and access array
				index, err := strconv.Atoi(indexStr)
				if err != nil {
					return nil, fmt.Errorf("invalid array index: %s", indexStr)
				}

				if arr, ok := current.([]interface{}); ok {
					if index >= 0 && index < len(arr) {
						current = arr[index]
					} else {
						return nil, fmt.Errorf("array index out of bounds: %d", index)
					}
				} else {
					return nil, fmt.Errorf("not an array at path: %s[%s]", propName, indexStr)
				}
			}
		} else {
			// Regular property access
			if mapObj, ok := current.(map[string]interface{}); ok {
				current = mapObj[part]
			} else {
				return nil, fmt.Errorf("not an object at path: %s", part)
			}
		}
	}

	return current, nil
}
