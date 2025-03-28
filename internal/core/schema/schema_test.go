// SPDX-License-Identifier: Apache-2.0

package schema_test

import (
	"testing"

	"github.com/kusari-oss/darn/internal/core/schema"
	"github.com/stretchr/testify/assert"
)

func TestValidateParams(t *testing.T) {
	tests := []struct {
		name       string
		schema     map[string]interface{}
		params     map[string]interface{}
		shouldPass bool
	}{
		{
			name: "valid simple parameters",
			schema: map[string]interface{}{
				"type":     "object",
				"required": []interface{}{"name", "repo"},
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
					"repo": map[string]interface{}{
						"type": "string",
					},
				},
			},
			params: map[string]interface{}{
				"name": "Darn Project",
				"repo": "github.com/kusari-oss/darn",
			},
			shouldPass: true,
		},
		{
			name: "missing required parameter",
			schema: map[string]interface{}{
				"type":     "object",
				"required": []interface{}{"name", "repo"},
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
					"repo": map[string]interface{}{
						"type": "string",
					},
				},
			},
			params: map[string]interface{}{
				"name": "Darn Project",
				// missing "repo"
			},
			shouldPass: false,
		},
		{
			name: "array type parameter validation",
			schema: map[string]interface{}{
				"type":     "object",
				"required": []interface{}{"emails"},
				"properties": map[string]interface{}{
					"emails": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
			params: map[string]interface{}{
				"emails": []interface{}{"user1@example.com", "user2@example.com"},
			},
			shouldPass: true,
		},
		{
			name: "array type parameter with wrong type",
			schema: map[string]interface{}{
				"type":     "object",
				"required": []interface{}{"emails"},
				"properties": map[string]interface{}{
					"emails": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
			params: map[string]interface{}{
				"emails": "user1@example.com", // Not an array
			},
			shouldPass: false,
		},
		{
			name: "numerical constraints",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"count": map[string]interface{}{
						"type":    "number",
						"minimum": float64(1),
						"maximum": float64(10),
					},
				},
			},
			params: map[string]interface{}{
				"count": float64(5),
			},
			shouldPass: true,
		},
		{
			name: "numerical constraints - out of range",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"count": map[string]interface{}{
						"type":    "number",
						"minimum": float64(1),
						"maximum": float64(10),
					},
				},
			},
			params: map[string]interface{}{
				"count": float64(20),
			},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.ValidateParams(tt.schema, tt.params)

			if tt.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestMergeWithDefaults(t *testing.T) {
	// Test cases for merging parameters with defaults
	tests := []struct {
		name     string
		params   map[string]interface{}
		defaults map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "merge with no conflicts",
			params: map[string]interface{}{
				"name": "Darn Project",
				"repo": "github.com/kusari-oss/darn",
			},
			defaults: map[string]interface{}{
				"create_dirs": true,
				"branch":      "main",
			},
			expected: map[string]interface{}{
				"name":        "Darn Project",
				"repo":        "github.com/kusari-oss/darn",
				"create_dirs": true,
				"branch":      "main",
			},
		},
		{
			name: "params override defaults",
			params: map[string]interface{}{
				"name":   "Darn Project",
				"branch": "feature-branch",
			},
			defaults: map[string]interface{}{
				"name":        "Default Project",
				"branch":      "main",
				"create_dirs": true,
			},
			expected: map[string]interface{}{
				"name":        "Darn Project",
				"branch":      "feature-branch",
				"create_dirs": true,
			},
		},
		{
			name:   "empty params",
			params: map[string]interface{}{},
			defaults: map[string]interface{}{
				"name":   "Default Project",
				"branch": "main",
			},
			expected: map[string]interface{}{
				"name":   "Default Project",
				"branch": "main",
			},
		},
		{
			name: "empty defaults",
			params: map[string]interface{}{
				"name":   "Darn Project",
				"branch": "feature-branch",
			},
			defaults: map[string]interface{}{},
			expected: map[string]interface{}{
				"name":   "Darn Project",
				"branch": "feature-branch",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := schema.MergeWithDefaults(tt.params, tt.defaults)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProcessParamsWithSchema(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		data        map[string]interface{}
		schema      map[string]interface{}
		expected    map[string]interface{}
		shouldError bool
	}{
		{
			name: "template substitution with schema typing",
			params: map[string]interface{}{
				"message": "Hello, {{.name}}!",
				"count":   "{{.number}}",
				"enabled": "{{.flag}}",
			},
			data: map[string]interface{}{
				"name":   "World",
				"number": 42,
				"flag":   true,
			},
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type": "string",
					},
					"count": map[string]interface{}{
						"type": "number",
					},
					"enabled": map[string]interface{}{
						"type": "boolean",
					},
				},
			},
			expected: map[string]interface{}{
				"message": "Hello, World!",
				"count":   float64(42),
				"enabled": true,
			},
			shouldError: false,
		},
		{
			name: "array substitution",
			params: map[string]interface{}{
				"items": "{{.list}}",
			},
			data: map[string]interface{}{
				"list": []interface{}{"one", "two", "three"},
			},
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"items": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"items": []interface{}{"one", "two", "three"},
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := schema.ProcessParamsWithSchema(tt.params, tt.data, tt.schema)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
