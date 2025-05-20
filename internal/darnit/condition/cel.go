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
	env *cel.Env
}

// NewCELEvaluator creates a new CEL evaluator
func NewCELEvaluator() (*CELEvaluator, error) {
	// Create a new CEL environment with standard env and findings variable
	// Use standard library which already includes string functions like contains, startsWith
	env, err := cel.NewEnv(
		cel.StdLib(), // Include standard library of functions
		cel.Variable("findings", cel.MapType(cel.StringType, cel.DynType)),
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

	return &CELEvaluator{env: env}, nil
}

// EvaluateExpression evaluates a CEL expression against data
func (e *CELEvaluator) EvaluateExpression(expression string, data map[string]interface{}) (bool, error) {
	// Parse the expression
	ast, issues := e.env.Parse(expression)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("error parsing expression: %w", issues.Err())
	}

	// Type-check the expression
	checked, issues := e.env.Check(ast)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("error type-checking expression: %w", issues.Err())
	}

	// Compile the expression
	program, err := e.env.Program(checked)
	if err != nil {
		return false, fmt.Errorf("error compiling expression: %w", err)
	}

	// Create the variable map from data
	vars := map[string]interface{}{
		"findings": data["findings"],
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

// EvaluateStringArrayExpression evaluates a CEL expression that returns a string array
func (e *CELEvaluator) EvaluateStringArrayExpression(expression string, data map[string]interface{}) ([]string, error) {
	// Parse the expression
	ast, issues := e.env.Parse(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("error parsing expression: %w", issues.Err())
	}

	// Type-check the expression
	checked, issues := e.env.Check(ast)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("error type-checking expression: %w", issues.Err())
	}

	// Compile the expression
	program, err := e.env.Program(checked)
	if err != nil {
		return nil, fmt.Errorf("error compiling expression: %w", err)
	}

	// Create the variable map from data
	vars := map[string]interface{}{
		"findings": data["findings"],
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
		return nil, fmt.Errorf("expression did not evaluate to a string array or string, got: %T", val)
	}
}
