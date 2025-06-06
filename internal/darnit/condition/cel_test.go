// SPDX-License-Identifier: Apache-2.0

package condition_test

import (
	"testing"

	"github.com/kusari-oss/darn/internal/darnit/condition"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCELEvaluator(t *testing.T) {
	// Create a new evaluator
	evaluator, err := condition.NewCELEvaluator()
	require.NoError(t, err, "Error creating CEL evaluator")

	// Test cases
	tests := []struct {
		name       string
		expression string
		data       map[string]interface{}
		expected   bool
		wantErr    bool
	}{
		{
			name:       "simple comparison - true",
			expression: "findings.security_policy == 'missing'",
			data: map[string]interface{}{
				"findings": map[string]interface{}{
					"security_policy": "missing",
				},
			},
			expected: true,
			wantErr:  false,
		},
		{
			name:       "simple comparison - false",
			expression: "findings.security_policy == 'missing'",
			data: map[string]interface{}{
				"findings": map[string]interface{}{
					"security_policy": "present",
				},
			},
			expected: false,
			wantErr:  false,
		},
		{
			name:       "logical AND - true",
			expression: "findings.security_policy == 'missing' && findings.mfa_status == 'disabled'",
			data: map[string]interface{}{
				"findings": map[string]interface{}{
					"security_policy": "missing",
					"mfa_status":      "disabled",
				},
			},
			expected: true,
			wantErr:  false,
		},
		{
			name:       "logical AND - false",
			expression: "findings.security_policy == 'missing' && findings.mfa_status == 'disabled'",
			data: map[string]interface{}{
				"findings": map[string]interface{}{
					"security_policy": "missing",
					"mfa_status":      "enabled",
				},
			},
			expected: false,
			wantErr:  false,
		},
		{
			name:       "logical OR - true",
			expression: "findings.security_policy == 'missing' || findings.mfa_status == 'disabled'",
			data: map[string]interface{}{
				"findings": map[string]interface{}{
					"security_policy": "present",
					"mfa_status":      "disabled",
				},
			},
			expected: true,
			wantErr:  false,
		},
		{
			name:       "logical OR - false",
			expression: "findings.security_policy == 'missing' || findings.mfa_status == 'disabled'",
			data: map[string]interface{}{
				"findings": map[string]interface{}{
					"security_policy": "present",
					"mfa_status":      "enabled",
				},
			},
			expected: false,
			wantErr:  false,
		},
		{
			name:       "complex condition - true",
			expression: "findings.security_policy == 'missing' || (findings.mfa_status == 'disabled' && findings.branch_protection == 'partial')",
			data: map[string]interface{}{
				"findings": map[string]interface{}{
					"security_policy":   "present",
					"mfa_status":        "disabled",
					"branch_protection": "partial",
				},
			},
			expected: true,
			wantErr:  false,
		},
		{
			name:       "invalid expression",
			expression: "findings.security_policy = 'missing'", // Invalid syntax (= instead of ==)
			data: map[string]interface{}{
				"findings": map[string]interface{}{
					"security_policy": "missing",
				},
			},
			expected: false,
			wantErr:  true,
		},
		{
			name:       "non-boolean result",
			expression: "findings.security_policy", // Doesn't evaluate to boolean
			data: map[string]interface{}{
				"findings": map[string]interface{}{
					"security_policy": "missing",
				},
			},
			expected: false,
			wantErr:  true,
		},
		{
			name:       "missing field",
			expression: "findings.nonexistent_field == 'value'",
			data: map[string]interface{}{
				"findings": map[string]interface{}{
					"security_policy": "missing",
				},
			},
			expected: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.EvaluateExpression(tt.expression, tt.data)

			if tt.wantErr {
				assert.Error(t, err, "Expected error for expression: %s", tt.expression)
			} else {
				assert.NoError(t, err, "Unexpected error for expression: %s", tt.expression)
				assert.Equal(t, tt.expected, result, "Unexpected result for expression: %s", tt.expression)
			}
		})
	}
}

func TestCELEvaluatorWithInvalidData(t *testing.T) {
	// Create a new evaluator
	evaluator, err := condition.NewCELEvaluator()
	require.NoError(t, err, "Error creating CEL evaluator")

	// Test with nil data
	_, err = evaluator.EvaluateExpression("findings.security_policy == 'missing'", nil)
	assert.Error(t, err, "Expected error for nil data")

	// Test with missing findings key
	_, err = evaluator.EvaluateExpression("findings.security_policy == 'missing'", map[string]interface{}{})
	assert.Error(t, err, "Expected error for missing findings key")

	// Test with findings not being a map
	_, err = evaluator.EvaluateExpression("findings.security_policy == 'missing'", map[string]interface{}{
		"findings": "not a map",
	})
	assert.Error(t, err, "Expected error for findings not being a map")
}
