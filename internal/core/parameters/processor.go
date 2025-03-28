// SPDX-License-Identifier: Apache-2.0

package parameters

import (
	"fmt"
	"regexp"
)

// ParameterProcessor handles parameter substitution and processing
type ParameterProcessor struct {
	// paramRegex matches template parameters like {{.name}}
	paramRegex *regexp.Regexp
}

// NewParameterProcessor creates a new parameter processor
func NewParameterProcessor() *ParameterProcessor {
	return &ParameterProcessor{
		paramRegex: regexp.MustCompile(`\{\{\.([^}]+)\}\}`),
	}
}

// SubstituteString replaces template parameters in a string with values from data
func (p *ParameterProcessor) SubstituteString(template string, data map[string]interface{}) (string, error) {
	result := p.paramRegex.ReplaceAllStringFunc(template, func(match string) string {
		// Extract key from {{.key}}
		key := match[3 : len(match)-2]

		// Look up value
		value, found := data[key]
		if !found {
			return match // Keep original if not found
		}

		// Convert value to string
		return fmt.Sprintf("%v", value)
	})

	// If any templates remain, there were missing values
	if p.paramRegex.MatchString(result) {
		return result, fmt.Errorf("missing values for some parameters: %s", result)
	}

	return result, nil
}

// ProcessMap substitutes parameters in all string values in a map
func (p *ParameterProcessor) ProcessMap(params map[string]interface{}, data map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range params {
		switch v := value.(type) {
		case string:
			processed, err := p.SubstituteString(v, data)
			if err != nil {
				return nil, fmt.Errorf("error processing parameter %s: %w", key, err)
			}
			result[key] = processed

		case []interface{}:
			processedSlice := make([]interface{}, len(v))
			for i, item := range v {
				if str, ok := item.(string); ok {
					processed, err := p.SubstituteString(str, data)
					if err != nil {
						return nil, fmt.Errorf("error processing array item: %w", err)
					}
					processedSlice[i] = processed
				} else {
					processedSlice[i] = item
				}
			}
			result[key] = processedSlice

		case map[string]interface{}:
			// Recursively process nested maps
			processed, err := p.ProcessMap(v, data)
			if err != nil {
				return nil, fmt.Errorf("error processing nested map %s: %w", key, err)
			}
			result[key] = processed

		default:
			result[key] = value
		}
	}

	return result, nil
}

// ExtractRequiredParameters finds all required parameters in template strings
func (p *ParameterProcessor) ExtractRequiredParameters(template string) []string {
	params := make(map[string]bool)

	matches := p.paramRegex.FindAllStringSubmatch(template, -1)
	for _, match := range matches {
		if len(match) > 1 {
			params[match[1]] = true
		}
	}

	// Convert map keys to slice
	result := make([]string, 0, len(params))
	for param := range params {
		result = append(result, param)
	}

	return result
}
