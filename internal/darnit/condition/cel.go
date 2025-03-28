// SPDX-License-Identifier: Apache-2.0

package condition

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
)

// CELEvaluator handles evaluation of CEL expressions
type CELEvaluator struct {
	env *cel.Env
}

// NewCELEvaluator creates a new CEL evaluator
func NewCELEvaluator() (*CELEvaluator, error) {
	// Create a new CEL environment with standard env and findings variable
	env, err := cel.NewEnv(
		cel.Variable("findings", cel.MapType(cel.StringType, cel.DynType)),
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
