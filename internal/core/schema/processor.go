// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var paramRegex = regexp.MustCompile(`\{\{\.([^}]+)\}\}`)

// ProcessParamsWithSchema processes parameters with schema type awareness
func ProcessParamsWithSchema(params map[string]interface{}, data map[string]interface{}, schema map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Get schema properties
	properties, _ := schema["properties"].(map[string]interface{})
	if properties == nil {
		// If no schema properties defined, fall back to untyped processing
		return processParamMapUntyped(params, data)
	}

	for key, value := range params {
		// Get schema property for this parameter
		propSchema, hasSchema := properties[key].(map[string]interface{})

		switch v := value.(type) {
		case string:
			// Process string with template substitution
			processed, err := substituteParameters(v, data)
			if err != nil {
				return nil, fmt.Errorf("error processing parameter %s: %w", key, err)
			}

			// If we have a schema and it specifies this should be an array, try to convert it
			if hasSchema && propSchema["type"] == "array" && strings.HasPrefix(processed, "[") && strings.HasSuffix(processed, "]") {
				// This might be a stringified array - try to convert it
				var arrayValue []interface{}
				if err := json.Unmarshal([]byte(processed), &arrayValue); err == nil {
					result[key] = arrayValue
					continue
				}
			} else if hasSchema && propSchema["type"] == "number" {
				// Try to convert to number if schema specifies
				if num, err := strconv.ParseFloat(processed, 64); err == nil {
					result[key] = num
					continue
				}
			} else if hasSchema && propSchema["type"] == "boolean" {
				// Try to convert to boolean if schema specifies
				if processed == "true" {
					result[key] = true
					continue
				} else if processed == "false" {
					result[key] = false
					continue
				}
			}

			// Default - keep as string
			result[key] = processed

		case []interface{}:
			// Process array items but maintain array structure
			processedArray, err := processArrayItems(v, data)
			if err != nil {
				return nil, fmt.Errorf("error processing array parameter %s: %w", key, err)
			}
			result[key] = processedArray

		case map[string]interface{}:
			// Process nested object
			processedObj, err := processParamMapUntyped(v, data)
			if err != nil {
				return nil, fmt.Errorf("error processing nested object parameter %s: %w", key, err)
			}
			result[key] = processedObj

		default:
			// For other types (bool, number, etc.), keep as is
			result[key] = value
		}
	}

	return result, nil
}

// Process array items, substituting string values
func processArrayItems(array []interface{}, data map[string]interface{}) ([]interface{}, error) {
	result := make([]interface{}, len(array))

	for i, item := range array {
		switch v := item.(type) {
		case string:
			processed, err := substituteParameters(v, data)
			if err != nil {
				return nil, err
			}
			result[i] = processed
		case []interface{}:
			// Handle nested arrays
			processedNested, err := processArrayItems(v, data)
			if err != nil {
				return nil, err
			}
			result[i] = processedNested
		case map[string]interface{}:
			// Handle objects in arrays
			processedObj, err := processParamMapUntyped(v, data)
			if err != nil {
				return nil, err
			}
			result[i] = processedObj
		default:
			// Keep other types as is
			result[i] = item
		}
	}

	return result, nil
}

// processParamMapUntyped processes parameters without schema type awareness
func processParamMapUntyped(params map[string]interface{}, data map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range params {
		switch v := value.(type) {
		case string:
			processed, err := substituteParameters(v, data)
			if err != nil {
				return nil, fmt.Errorf("error processing parameter %s: %w", key, err)
			}
			result[key] = processed

		case []interface{}:
			processedSlice, err := processArrayItems(v, data)
			if err != nil {
				return nil, fmt.Errorf("error processing array parameter %s: %w", key, err)
			}
			result[key] = processedSlice

		case map[string]interface{}:
			processedObj, err := processParamMapUntyped(v, data)
			if err != nil {
				return nil, fmt.Errorf("error processing nested object parameter %s: %w", key, err)
			}
			result[key] = processedObj

		default:
			result[key] = value
		}
	}

	return result, nil
}

// substituteParameters replaces template parameters in a string with values from data
func substituteParameters(template string, data map[string]interface{}) (string, error) {
	result := paramRegex.ReplaceAllStringFunc(template, func(match string) string {
		// Extract key from {{.key}}
		key := match[3 : len(match)-2]

		// Look up value
		value, found := data[key]
		if !found {
			return match // Keep original if not found
		}

		// Convert value to string, but maintain special representations for complex types
		switch v := value.(type) {
		case []interface{}, map[string]interface{}:
			// For complex types, use JSON representation
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return match // Return original on error
			}
			return string(jsonBytes)
		default:
			// For simple types, use string representation
			return fmt.Sprintf("%v", value)
		}
	})

	// If any templates remain, there were missing values
	if paramRegex.MatchString(result) {
		return result, fmt.Errorf("missing values for some parameters: %s", result)
	}

	return result, nil
}
