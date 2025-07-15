// SPDX-License-Identifier: Apache-2.0

package condition

import (
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// CELEvaluator handles evaluation of CEL expressions
type CELEvaluator struct {
	baseEnv *cel.Env
}

// NewCELEvaluator creates a new CEL evaluator
func NewCELEvaluator() (*CELEvaluator, error) {
	// Create a new CEL environment with standard env and dynamic variable support
	// Use standard library which already includes string functions like contains, startsWith
	env, err := cel.NewEnv(
		cel.StdLib(), // Include standard library of functions
		// Only add custom functions that aren't in the standard library
		cel.Function("split",
			cel.MemberOverload("string_split_string",
				[]*cel.Type{cel.StringType, cel.StringType},
				cel.ListType(cel.StringType),
				cel.BinaryBinding(func(lhs, rhs ref.Val) ref.Val {
					s1, ok1 := lhs.Value().(string)
					s2, ok2 := rhs.Value().(string)
					if !ok1 || !ok2 {
						return types.NewErr("split: unexpected type")
					}
					parts := strings.Split(s1, s2)
					return types.NewStringList(types.DefaultTypeAdapter, parts)
				}),
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating CEL environment: %w", err)
	}

	return &CELEvaluator{baseEnv: env}, nil
}

// EvaluateExpression evaluates a CEL expression against data
func (e *CELEvaluator) EvaluateExpression(expression string, data map[string]any) (bool, error) {
	// Create a dynamic environment with variables for all data keys
	dynamicEnv, err := e.createDynamicEnv(data)
	if err != nil {
		return false, fmt.Errorf("error creating dynamic environment: %w", err)
	}

	// Parse the expression
	ast, issues := dynamicEnv.Parse(expression)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("error parsing expression: %w", issues.Err())
	}

	// Type-check the expression
	checked, issues := dynamicEnv.Check(ast)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("error type-checking expression: %w", issues.Err())
	}

	// Compile the expression
	program, err := dynamicEnv.Program(checked)
	if err != nil {
		return false, fmt.Errorf("error compiling expression: %w", err)
	}

	// Create the variable map from data (flat access only)
	vars := make(map[string]any)
	
	// Add all fields from data directly for flat access
	for key, value := range data {
		vars[key] = value
	}

	// Evaluate the expression
	result, _, err := program.Eval(vars)
	if err != nil {
		return false, fmt.Errorf("error evaluating expression: %w", err)
	}

	// Convert result to boolean
	if result.Type() != types.BoolType {
		return false, fmt.Errorf("expression did not evaluate to a boolean")
	}

	return result.Value().(bool), nil
}

// createDynamicEnv creates a CEL environment with variables for all data keys
func (e *CELEvaluator) createDynamicEnv(data map[string]any) (*cel.Env, error) {
	// Start with base environment options
	opts := []cel.EnvOption{cel.StdLib()}
	
	// Add custom functions from base environment
	opts = append(opts, cel.Function("split",
		cel.MemberOverload("string_split_string",
			[]*cel.Type{cel.StringType, cel.StringType},
			cel.ListType(cel.StringType),
			cel.BinaryBinding(func(lhs, rhs ref.Val) ref.Val {
				s1, ok1 := lhs.Value().(string)
				s2, ok2 := rhs.Value().(string)
				if !ok1 || !ok2 {
					return types.NewErr("split: unexpected type")
				}
				parts := strings.Split(s1, s2)
				return types.NewStringList(types.DefaultTypeAdapter, parts)
			}),
		),
	))
	
	// Add variables for all data keys
	for key := range data {
		opts = append(opts, cel.Variable(key, cel.DynType))
	}
	
	return cel.NewEnv(opts...)
}

// EvaluateStringArrayExpression evaluates a CEL expression that returns a string array
func (e *CELEvaluator) EvaluateStringArrayExpression(expression string, data map[string]any) ([]string, error) {
	// Create a dynamic environment with variables for all data keys
	dynamicEnv, err := e.createDynamicEnv(data)
	if err != nil {
		return nil, fmt.Errorf("error creating dynamic environment: %w", err)
	}

	// Parse the expression
	ast, issues := dynamicEnv.Parse(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("error parsing expression: %w", issues.Err())
	}

	// Type-check the expression
	checked, issues := dynamicEnv.Check(ast)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("error type-checking expression: %w", issues.Err())
	}

	// Compile the expression
	program, err := dynamicEnv.Program(checked)
	if err != nil {
		return nil, fmt.Errorf("error compiling expression: %w", err)
	}

	// Create the variable map from data (flat access only)
	vars := make(map[string]any)
	
	// Add all fields from data directly for flat access
	for key, value := range data {
		vars[key] = value
	}

	// Evaluate the expression
	result, _, err := program.Eval(vars)
	if err != nil {
		return nil, fmt.Errorf("error evaluating expression: %w", err)
	}

	// Extract string array from result
	val := result.Value()
	if val == nil {
		return []string{}, nil
	}

	// Handle different return types
	switch v := val.(type) {
	case []interface{}:
		// Convert []interface{} to []string
		strArray := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				strArray = append(strArray, str)
			} else {
				strArray = append(strArray, fmt.Sprintf("%v", item))
			}
		}
		return strArray, nil
	case []string:
		return v, nil
	case string:
		// Single string - return as one-element array
		return []string{v}, nil
	default:
		// Handle CEL-specific list types ([]ref.Val)
		if list, ok := val.([]ref.Val); ok {
			strArray := make([]string, 0, len(list))
			for _, item := range list {
				if str, ok := item.Value().(string); ok {
					strArray = append(strArray, str)
				} else {
					strArray = append(strArray, fmt.Sprintf("%v", item.Value()))
				}
			}
			return strArray, nil
		}
		return nil, fmt.Errorf("expression did not evaluate to a string array or string, got: %T", val)
	}
}
