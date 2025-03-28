// SPDX-License-Identifier: Apache-2.0

// TODO: Some of the logic here is duplicated in the resolver. I wonder if we can share some of that and also simplify it.

package action

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kusari-oss/darn/internal/core/schema"
	"github.com/kusari-oss/darn/internal/core/template"
)

// FileProcessor handles file operations
type FileProcessor struct {
	templatePath string
}

// NewFileProcessor creates a new file processor
func NewFileProcessor(templatePath string) *FileProcessor {
	return &FileProcessor{
		templatePath: templatePath,
	}
}

// ProcessAndWriteFile processes a template and writes it to the target path
func (p *FileProcessor) ProcessAndWriteFile(targetPath string, params map[string]interface{}, createDirs bool) error {
	// Process the target path (it might contain template variables)
	processedTargetPath, err := template.ProcessString(targetPath, params)
	if err != nil {
		return fmt.Errorf("error processing target path: %w", err)
	}

	targetPathStr := string(processedTargetPath)

	// Create directories if needed
	if createDirs {
		dir := filepath.Dir(targetPathStr)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creating directories: %w", err)
		}
	}

	// Process the template file
	processedContent, err := template.ProcessFile(p.templatePath, params)
	if err != nil {
		return fmt.Errorf("error processing template: %w", err)
	}

	// Write the file
	err = os.WriteFile(targetPathStr, processedContent, 0644)
	if err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	fmt.Printf("Created file: %s\n", targetPathStr)
	return nil
}

// FileAction creates a file from a template
type FileAction struct {
	config                 Config
	templatesDir           string
	globalTemplatesDir     string
	additionalTemplateDirs []string
	useLocal               bool
	useGlobal              bool
	globalFirst            bool
}

// Execute runs the file action with enhanced parameter validation
func (a *FileAction) Execute(params map[string]interface{}) error {
	// First, validate parameters against the schema if available
	if a.config.Schema != nil {
		// Custom validation for common issues
		if err := a.validateCommonIssues(params); err != nil {
			return err
		}

		// Then do standard schema validation
		if err := schema.ValidateParams(a.config.Schema, params); err != nil {
			return fmt.Errorf("parameter validation failed: %w", err)
		}
	}

	// Resolve template path
	// Build a list of paths to check in order
	var pathsToCheck []string

	// Add local and global paths according to configuration
	if a.useLocal && a.useGlobal {
		if a.globalFirst {
			// Global first, then local
			pathsToCheck = append(pathsToCheck, filepath.Join(a.globalTemplatesDir, a.config.TemplatePath))
			pathsToCheck = append(pathsToCheck, filepath.Join(a.templatesDir, a.config.TemplatePath))
		} else {
			// Local first, then global
			pathsToCheck = append(pathsToCheck, filepath.Join(a.templatesDir, a.config.TemplatePath))
			pathsToCheck = append(pathsToCheck, filepath.Join(a.globalTemplatesDir, a.config.TemplatePath))
		}
	} else if a.useLocal {
		// Only local
		pathsToCheck = append(pathsToCheck, filepath.Join(a.templatesDir, a.config.TemplatePath))
	} else if a.useGlobal {
		// Only global
		pathsToCheck = append(pathsToCheck, filepath.Join(a.globalTemplatesDir, a.config.TemplatePath))
	} else {
		return fmt.Errorf("neither local nor global templates enabled")
	}

	// Add additional template directories
	for _, additionalDir := range a.additionalTemplateDirs {
		pathsToCheck = append(pathsToCheck, filepath.Join(additionalDir, a.config.TemplatePath))
	}

	// Check each path in order
	var templatePath string
	for _, path := range pathsToCheck {
		if _, err := os.Stat(path); err == nil {
			templatePath = path
			break
		}
	}

	// If no template was found
	if templatePath == "" {
		return fmt.Errorf("template '%s' not found in any configured location", a.config.TemplatePath)
	}

	// Use the found template path
	processor := NewFileProcessor(templatePath)

	// Log the path if verbose mode is enabled
	if verbose, ok := params["verbose"].(bool); ok && verbose {
		fmt.Printf("Using template: %s\n", templatePath)
	}

	return processor.ProcessAndWriteFile(
		a.config.TargetPath,
		params,
		a.config.CreateDirs,
	)
}

// validateCommonIssues checks for common parameter issues and provides clear error messages
func (a *FileAction) validateCommonIssues(params map[string]interface{}) error {
	// Only check if we have a schema
	if a.config.Schema == nil {
		return nil
	}

	// Check schema for array properties
	if props, ok := a.config.Schema["properties"].(map[string]interface{}); ok {
		for propName, propSpec := range props {
			if specMap, ok := propSpec.(map[string]interface{}); ok {
				// Check if property should be an array
				if propType, ok := specMap["type"].(string); ok && propType == "array" {
					// Check if we have this property
					if value, exists := params[propName]; exists {
						// Verify it's actually an array
						switch value.(type) {
						case []interface{}, []string:
							// It's an array, so it's fine
						case string:
							// It's a string, check if it looks like a JSON array
							strValue := value.(string)
							if strings.HasPrefix(strValue, "[") && strings.HasSuffix(strValue, "]") {
								// Try to parse as JSON array
								var array []interface{}
								if err := json.Unmarshal([]byte(strValue), &array); err == nil {
									// Replace with parsed array
									params[propName] = array
								} else {
									// Not a valid JSON array
									description := propName
									if desc, ok := specMap["description"].(string); ok {
										description = desc
									}
									return fmt.Errorf("parameter '%s' (%s) contains a string that looks like an array but isn't valid JSON: %v\nPlease modify your parameters to use a proper array format",
										propName, description, value)
								}
							} else {
								// It's not an array - provide a helpful error
								description := propName
								if desc, ok := specMap["description"].(string); ok {
									description = desc
								}
								return fmt.Errorf("parameter '%s' (%s) must be an array, but got a string value: %v\nPlease modify your parameters to use an array format, e.g., [%v]",
									propName, description, value, value)
							}
						default:
							// It's not an array - provide a helpful error
							description := propName
							if desc, ok := specMap["description"].(string); ok {
								description = desc
							}
							return fmt.Errorf("parameter '%s' (%s) must be an array, but got a single value: %v\nPlease modify your parameters to use an array format, e.g., [%v]",
								propName, description, value, value)
						}
					}
				}
			}
		}
	}

	return nil
}

// ExecuteWithOutput runs the file action and returns outputs
func (a *FileAction) ExecuteWithOutput(params map[string]interface{}) (map[string]interface{}, error) {
	// Use the same template resolution logic as Execute
	err := a.Execute(params)
	if err != nil {
		return nil, err
	}

	// Process the target path for output
	processedTargetPath, err := template.ProcessString(a.config.TargetPath, params)
	if err != nil {
		return nil, fmt.Errorf("error processing target path for output: %w", err)
	}

	// Return output with the processed path
	outputs := make(map[string]interface{})
	outputs["file_path"] = string(processedTargetPath)

	return outputs, nil
}

// Description returns the action description
func (a *FileAction) Description() string {
	if a.config.Description != "" {
		return a.config.Description
	}
	return "Create a file from a template"
}
