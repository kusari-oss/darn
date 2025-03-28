// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

// TODO: The gojsonschema library is quite old with no updates. It might be worth looking to see if there's a newer maintained
// alternative.
// ValidateParams validates parameters against a JSON schema
func ValidateParams(schema map[string]interface{}, params map[string]interface{}) error {
	// Convert the schema to JSON
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("schema validation error: failed to serialize schema: %w", err)
	}

	// Use gojsonschema's loader to ensure proper schema format
	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)

	// Convert params to JSON too for consistency
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("schema validation error: failed to serialize params: %w", err)
	}
	documentLoader := gojsonschema.NewBytesLoader(paramsBytes)

	// Validate using the properly formatted schema
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	// Check validation result
	if !result.Valid() {
		errorMsg := "Parameter validation failed:\n"
		for _, err := range result.Errors() {
			errorMsg += fmt.Sprintf("- %s\n", err)
		}
		return fmt.Errorf("%s", errorMsg)
	}

	return nil
}

// MergeWithDefaults merges params with default values
func MergeWithDefaults(params map[string]interface{}, defaults map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// First copy defaults
	for k, v := range defaults {
		result[k] = v
	}

	// Then override with actual params
	for k, v := range params {
		result[k] = v
	}

	return result
}
