// SPDX-License-Identifier: Apache-2.0

// TODO: This file needs a lot of TLC. I couldn't figure out a good way to do the combination of:
// * Parsing yaml
// * Parsing the json schema embedded in the yaml
// * Do some sort of validation of those parameters
// * Merge the parameters with defaults
// * Feed those parameters in to templates

package resolver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kusari-oss/darn/internal/core/action"
	"github.com/kusari-oss/darn/internal/core/library"
)

// Resolver handles finding and loading actions based on configuration
type Resolver struct {
	// Paths to search for actions, in order of precedence
	actionPaths []string

	// Factory for creating actions
	factory *action.Factory
}

// NewResolver creates a new action resolver compatible with the new factory
func NewResolver(factory *action.Factory, projectDir string, useLocal, useGlobal bool, globalFirst bool, localActionsDir, libraryPath string) *Resolver {
	var actionPaths []string

	// Add paths based on configuration and precedence
	if useLocal && useGlobal {
		if globalFirst {
			// Global first, then local
			actionPaths = append(actionPaths, filepath.Join(libraryPath, "actions"))
			actionPaths = append(actionPaths, filepath.Join(projectDir, localActionsDir))
		} else {
			// Local first, then global
			actionPaths = append(actionPaths, filepath.Join(projectDir, localActionsDir))
			actionPaths = append(actionPaths, filepath.Join(libraryPath, "actions"))
		}
	} else if useLocal {
		// Only local
		actionPaths = append(actionPaths, filepath.Join(projectDir, localActionsDir))
	} else if useGlobal {
		// Only global
		actionPaths = append(actionPaths, filepath.Join(libraryPath, "actions"))
	}

	return &Resolver{
		actionPaths: actionPaths,
		factory:     factory,
	}
}

// ResolveAction finds and loads an action by name, using the new factory
func (r *Resolver) ResolveAction(name string) (action.Action, error) {
	var lastErr error

	// Search for the action in each path
	for _, path := range r.actionPaths {
		actionPath := filepath.Join(path, name+".yaml")

		// Check if file exists
		_, err := os.Stat(actionPath)
		if err != nil {
			lastErr = err
			continue
		}

		// File found, load the action
		actionConfig, err := LoadActionConfig(actionPath)
		if err != nil {
			lastErr = err
			continue
		}

		// Create the action using the factory
		action, err := r.factory.Create(*actionConfig)
		if err != nil {
			lastErr = err
			continue
		}

		return action, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("could not resolve action '%s': %w", name, lastErr)
	}

	return nil, fmt.Errorf("action '%s' not found in any configured location", name)
}

// ListAvailableActions lists all available actions from all configured locations
func (r *Resolver) ListAvailableActions() (map[string]action.Config, error) {
	actions := make(map[string]action.Config)

	for _, path := range r.actionPaths {
		// Skip if path doesn't exist
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		// List all YAML files in the directory
		entries, err := os.ReadDir(path)
		if err != nil {
			continue // Skip directories we can't read
		}

		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
				continue
			}

			// Load the action config
			actionPath := filepath.Join(path, entry.Name())
			actionConfig, err := LoadActionConfig(actionPath)
			if err != nil {
				continue // Skip invalid action configs
			}

			// Use filename as name if not specified
			if actionConfig.Name == "" {
				actionConfig.Name = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			}

			// Add to map if not already present (respects precedence order)
			if _, exists := actions[actionConfig.Name]; !exists {
				actions[actionConfig.Name] = *actionConfig
			}
		}
	}

	return actions, nil
}

func LoadActionConfig(path string) (*action.Config, error) {
	// Read the action file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading action file: %w", err)
	}

	// First convert YAML to a generic map
	var rawMap map[string]interface{}
	if err := yaml.Unmarshal(data, &rawMap); err != nil {
		return nil, fmt.Errorf("error parsing action file: %w", err)
	}

	// Use the helper recursive conversion function
	sanitizedInterface := sanitizeMapKeys(rawMap)

	// Perform type assertion
	sanitizedMap, ok := sanitizedInterface.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error: sanitized map is not of expected type")
	}

	// Create the config manually from the sanitized map
	actionConfig := &action.Config{
		Name:         getStringValue(sanitizedMap, "name"),
		Type:         getStringValue(sanitizedMap, "type"),
		Description:  getStringValue(sanitizedMap, "description"),
		TemplatePath: getStringValue(sanitizedMap, "template_path"),
		TargetPath:   getStringValue(sanitizedMap, "target_path"),
		CreateDirs:   getBoolValue(sanitizedMap, "create_dirs"),
		Command:      getStringValue(sanitizedMap, "command"),
		Args:         getStringSlice(sanitizedMap, "args"),
		Schema:       getMap(sanitizedMap, "schema"),
		Defaults:     getMap(sanitizedMap, "defaults"),
		Outputs:      getValue(sanitizedMap, "outputs"),
	}

	// Handle labels specifically
	if labelsMap, ok := sanitizedMap["labels"].(map[string]interface{}); ok {
		actionConfig.Labels = make(map[string][]string)
		for key, value := range labelsMap {
			if valueSlice, ok := value.([]interface{}); ok {
				strSlice := make([]string, 0, len(valueSlice))
				for _, v := range valueSlice {
					if strVal, ok := v.(string); ok {
						strSlice = append(strSlice, strVal)
					}
				}
				actionConfig.Labels[key] = strSlice
			}
		}
	}

	// If no name is specified, use the filename (without extension)
	if actionConfig.Name == "" {
		actionConfig.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	// TODO: Handle debug logging
	/*
		fmt.Printf("DEBUG: Loaded action '%s', type: %s, template_path: %s\n",
			actionConfig.Name, actionConfig.Type, actionConfig.TemplatePath)*/

	return actionConfig, nil
}

// NOTE: I wonder if something like reflection or using some sort of tagging with types would be better here.
// Helper functions to extract values from the map

func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBoolValue(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getStringSlice(m map[string]interface{}, key string) []string {
	if v, ok := m[key]; ok {
		if slice, ok := v.([]interface{}); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if mapVal, ok := v.(map[string]interface{}); ok {
			return mapVal
		}
	}
	return nil
}

func getValue(m map[string]interface{}, key string) interface{} {
	if v, ok := m[key]; ok {
		return v
	}
	return nil
}

// NOTE: I do not know if I need this, but couldn't figure out a better way to do this.
// Helper function to convert map[interface{}]interface{} to map[string]interface{}
func sanitizeMapKeys(i interface{}) interface{} {
	switch v := i.(type) {
	case map[interface{}]interface{}:
		// Convert map[interface{}]interface{} to map[string]interface{}
		result := make(map[string]interface{})
		for k, val := range v {
			// Convert key to string
			strKey := fmt.Sprintf("%v", k)
			// Recursively sanitize the value
			result[strKey] = sanitizeMapKeys(val)
		}
		return result
	case map[string]interface{}:
		// Recursively sanitize map values
		for k, val := range v {
			v[k] = sanitizeMapKeys(val)
		}
		return v
	case []interface{}:
		// Recursively sanitize slice elements
		for i, val := range v {
			v[i] = sanitizeMapKeys(val)
		}
		return v
	default:
		// Return as is for other types
		return v
	}
}

// ResolveTemplatePath resolves a template path based on configuration
func (r *Resolver) ResolveTemplatePath(templatePath string, projectDir string, useLocal, useGlobal bool,
	globalFirst bool, localTemplatesDir, libraryPath string) (string, error) {
	var templatePaths []string

	// Construct template paths based on configuration and precedence
	if useLocal && useGlobal {
		if globalFirst {
			// Global first, then local
			templatePaths = append(templatePaths, filepath.Join(libraryPath, "templates", templatePath))
			templatePaths = append(templatePaths, filepath.Join(projectDir, localTemplatesDir, templatePath))
		} else {
			// Local first, then global
			templatePaths = append(templatePaths, filepath.Join(projectDir, localTemplatesDir, templatePath))
			templatePaths = append(templatePaths, filepath.Join(libraryPath, "templates", templatePath))
		}
	} else if useLocal {
		// Only local
		templatePaths = append(templatePaths, filepath.Join(projectDir, localTemplatesDir, templatePath))
	} else if useGlobal {
		// Only global
		templatePaths = append(templatePaths, filepath.Join(libraryPath, "templates", templatePath))
	}

	// Check each path
	for _, path := range templatePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("template '%s' not found in any configured location", templatePath)
}

// GetActionConfig retrieves the action configuration without creating the action
// This is needed for schema validation and defaults
func (r *Resolver) GetActionConfig(name string) (*action.Config, error) {
	var lastErr error

	// Search for the action in each path
	for _, path := range r.actionPaths {
		actionPath := filepath.Join(path, name+".yaml")

		// Check if file exists
		_, err := os.Stat(actionPath)
		if err != nil {
			lastErr = err
			continue
		}

		// File found, load the action config
		actionConfig, err := LoadActionConfig(actionPath)
		if err != nil {
			lastErr = err
			continue
		}

		return actionConfig, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("could not find action configuration '%s': %w", name, lastErr)
	}

	return nil, fmt.Errorf("action configuration '%s' not found in any configured location", name)
}

// FilterActionsByLabels filters actions based on provided key-value label selectors
// labelSelectors format: "key1=value1,key2=value2" or "key1 in (value1,value2),key2=value3"
func (r *Resolver) FilterActionsByLabels(actions map[string]action.Config, labelSelectors map[string][]string) map[string]action.Config {
	if len(labelSelectors) == 0 {
		return actions // No filtering needed
	}

	filtered := make(map[string]action.Config)

	for name, action := range actions {
		if matchesLabelSelectors(action.Labels, labelSelectors) {
			filtered[name] = action
		}
	}

	return filtered
}

// matchesLabelSelectors checks if the action's labels match the selectors
func matchesLabelSelectors(actionLabels map[string][]string, selectors map[string][]string) bool {
	for selectorKey, selectorValues := range selectors {
		// Check if the action has this label category
		actionValues, hasCategory := actionLabels[selectorKey]
		if !hasCategory {
			return false
		}

		// Check if any of the action's values for this category match any of the selector values
		if !hasAnyMatchingValue(actionValues, selectorValues) {
			return false
		}
	}

	return true
}

// hasAnyMatchingValue checks if two string slices have any common element
func hasAnyMatchingValue(actionValues, selectorValues []string) bool {
	for _, selectorValue := range selectorValues {
		for _, actionValue := range actionValues {
			if strings.EqualFold(actionValue, selectorValue) {
				return true
			}
		}
	}
	return false
}

// ValidateLibraryPaths validates that all configured library paths exist and are accessible
func (r *Resolver) ValidateLibraryPaths() []error {
	var errors []error
	
	for _, path := range r.actionPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			errors = append(errors, fmt.Errorf("action path does not exist: %s", path))
		} else if err != nil {
			errors = append(errors, fmt.Errorf("cannot access action path %s: %w", path, err))
		}
	}
	
	return errors
}

// ValidateActionCommand validates that an action's shell command exists and is executable
func (r *Resolver) ValidateActionCommand(actionConfig *action.Config) error {
	if actionConfig.Type != "shell" && actionConfig.Type != "cli" {
		return nil // Only validate shell/cli commands
	}
	
	if actionConfig.Command == "" {
		return fmt.Errorf("action '%s' has empty command", actionConfig.Name)
	}
	
	// Use library manager for validation
	manager := library.NewManager("", "", false)
	return manager.ValidateShellCommand(actionConfig.Command)
}
