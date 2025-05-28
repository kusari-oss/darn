// SPDX-License-Identifier: Apache-2.0

package format

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseFile reads and parses a file, trying YAML first, then JSON
func ParseFile(filePath string, v interface{}) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return ParseData(data, v)
}

// ParseData parses data, trying YAML first, then JSON
func ParseData(data []byte, v interface{}) error {
	// Try YAML first (preferred format)
	err := yaml.Unmarshal(data, v)
	if err == nil {
		return nil
	}

	// If YAML fails, try JSON for backward compatibility
	jsonErr := json.Unmarshal(data, v)
	if jsonErr == nil {
		return nil
	}

	// Both failed - return the more informative error
	return fmt.Errorf("failed to parse as YAML (%v) or JSON (%v)", err, jsonErr)
}

// WriteFile writes data to a file in the specified format based on file extension
func WriteFile(filePath string, v interface{}) error {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	var data []byte
	var err error

	switch ext {
	case ".json":
		data, err = json.MarshalIndent(v, "", "  ")
	case ".yaml", ".yml", "":
		// Default to YAML for no extension or explicit YAML extensions
		data, err = yaml.Marshal(v)
	default:
		// For unknown extensions, use YAML as default
		data, err = yaml.Marshal(v)
	}

	if err != nil {
		return fmt.Errorf("error marshaling data: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

// WriteYAML writes data to a file in YAML format
func WriteYAML(filePath string, v interface{}) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("error marshaling YAML: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

// WriteJSON writes data to a file in JSON format (for backward compatibility)
func WriteJSON(filePath string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

// FormatData formats data as YAML or JSON string
func FormatData(v interface{}, useYAML bool) (string, error) {
	var data []byte
	var err error

	if useYAML {
		data, err = yaml.Marshal(v)
	} else {
		data, err = json.MarshalIndent(v, "", "  ")
	}

	if err != nil {
		return "", fmt.Errorf("error formatting data: %w", err)
	}

	return string(data), nil
}

// IsYAMLFile returns true if the file extension suggests it's a YAML file
func IsYAMLFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".yaml" || ext == ".yml"
}

// IsJSONFile returns true if the file extension suggests it's a JSON file
func IsJSONFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".json"
}