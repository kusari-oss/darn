// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kusari-oss/darn/internal/core/action"
	"github.com/kusari-oss/darn/internal/core/models"
	r "github.com/kusari-oss/darn/internal/darn/resolver"
)

// StepExecutor is responsible for executing a single step
type StepExecutor struct {
	resolver    *r.Resolver
	stepOutputs map[string]map[string]interface{}
	options     models.ExecutionOptions // Use from models package
}

// NewStepExecutor creates a new step executor
func NewStepExecutor(resolver *r.Resolver, options models.ExecutionOptions) *StepExecutor {
	return &StepExecutor{
		resolver:    resolver,
		stepOutputs: make(map[string]map[string]interface{}),
		options:     options,
	}
}

// ExecuteStep executes a single step in the plan
func (e *StepExecutor) ExecuteStep(step *models.RemediationStep) error {
	// Update status
	step.Status = "running"

	if e.options.VerboseLogging {
		fmt.Printf("Executing step: %s (Action: %s)\n", step.ID, step.ActionName)
		fmt.Printf("Reason: %s\n", step.Reason)
	} else {
		fmt.Printf("Executing step: %s\n", step.ID)
	}

	// Process any parameter references from previous steps
	if err := e.processOutputReferences(step); err != nil {
		return err
	}

	// In dry-run mode, just print the parameters
	if e.options.DryRun {
		e.dryRunStep(step)
		return nil
	}

	// Get the action
	act, err := e.resolver.ResolveAction(step.ActionName)
	if err != nil {
		step.Status = "failure"
		step.Error = fmt.Sprintf("error resolving action: %v", err)

		if e.options.VerboseLogging {
			fmt.Printf("Error resolving action '%s': %v\n", step.ActionName, err)
		} else {
			fmt.Printf("Error: %v\n", err)
		}

		return fmt.Errorf("error resolving action '%s': %w", step.ActionName, err)
	}

	// Add verbose flag to params if needed
	if step.Params == nil {
		step.Params = make(map[string]interface{})
	}
	step.Params["verbose"] = e.options.VerboseLogging

	// Execute the step based on its type
	return e.executeAction(step, act)
}

// processOutputReferences processes parameter references from previous steps
func (e *StepExecutor) processOutputReferences(step *models.RemediationStep) error {
	if len(step.OutputRefs) == 0 {
		return nil
	}

	if step.Params == nil {
		step.Params = make(map[string]interface{})
	}

	// For each output reference, get the value from previous step outputs
	for paramName, outputRef := range step.OutputRefs {
		// Format should be "step_id.output_name"
		parts := strings.Split(outputRef, ".")
		if len(parts) != 2 {
			return fmt.Errorf("invalid output reference format: %s", outputRef)
		}

		sourceStepID, outputName := parts[0], parts[1]

		// Check if source step exists and has outputs
		sourceOutputs, exists := e.stepOutputs[sourceStepID]
		if !exists {
			return fmt.Errorf("referenced step %s not found or has not completed successfully", sourceStepID)
		}

		// Get output value
		outputValue, exists := sourceOutputs[outputName]
		if !exists {
			return fmt.Errorf("output %s not found in step %s", outputName, sourceStepID)
		}

		// Get action schema for type checking
		_, err := e.resolver.GetActionConfig(step.ActionName)
		if err != nil {
			return fmt.Errorf("error getting action config: %w", err)
		}

		// Set the parameter value from the output, preserving the correct type
		step.Params[paramName] = outputValue

		if e.options.VerboseLogging {
			fmt.Printf("  Setting parameter %s to value from %s.%s (type: %T)\n",
				paramName, sourceStepID, outputName, outputValue)
		}
	}

	return nil
}

// dryRunStep simulates step execution in dry-run mode
func (e *StepExecutor) dryRunStep(step *models.RemediationStep) {
	paramsJSON, _ := json.MarshalIndent(step.Params, "  ", "  ")
	fmt.Printf("  Would execute action '%s' with parameters:\n  %s\n",
		step.ActionName, string(paramsJSON))

	// In dry-run, simulate outputs for next steps
	if step.Outputs != nil {
		e.stepOutputs[step.ID] = step.Outputs
	}

	// Mark step as successful in dry-run mode
	step.Status = "success"
}

// executeAction executes the action and processes its outputs
func (e *StepExecutor) executeAction(step *models.RemediationStep, act action.Action) error {
	// Get the action configuration to access the schema
	_, err := e.resolver.GetActionConfig(step.ActionName)
	if err != nil {
		e.handleExecutionError(step, fmt.Errorf("error getting action config: %w", err))
		return err
	}

	// Check if this is an OutputAction
	if outputAct, ok := act.(action.OutputAction); ok {
		// Execute with output capturing
		stepResult, err := outputAct.ExecuteWithOutput(step.Params)
		if err != nil {
			e.handleExecutionError(step, err)
			return err
		}

		// Store outputs for future steps to use
		if stepResult != nil {
			e.stepOutputs[step.ID] = stepResult
			step.Outputs = stepResult
		}
	} else {
		// Regular execution
		if err := act.Execute(step.Params); err != nil {
			e.handleExecutionError(step, err)
			return err
		}
	}

	// If this step defines static outputs, save them
	if step.Outputs != nil && len(e.stepOutputs[step.ID]) == 0 {
		e.stepOutputs[step.ID] = step.Outputs
	}

	// Step succeeded
	step.Status = "success"

	if e.options.VerboseLogging {
		fmt.Printf("Step completed successfully\n")
		if len(e.stepOutputs[step.ID]) > 0 {
			outputJSON, _ := json.MarshalIndent(e.stepOutputs[step.ID], "  ", "  ")
			fmt.Printf("  Outputs: %s\n", string(outputJSON))
		}
	}

	return nil
}

// handleExecutionError processes an execution error
func (e *StepExecutor) handleExecutionError(step *models.RemediationStep, err error) {
	step.Status = "failure"
	step.Error = fmt.Sprintf("execution failed: %v", err)

	if e.options.VerboseLogging {
		fmt.Printf("Error executing action '%s': %v\n", step.ActionName, err)
	} else {
		fmt.Printf("Error: %v\n", err)
	}
}

// PlanExecutor executes a remediation plan
type PlanExecutor struct {
	factory      *action.Factory
	resolver     *r.Resolver
	stepExecutor *StepExecutor
	options      models.ExecutionOptions
}

// NewPlanExecutor creates a new plan executor
func NewPlanExecutor(factory *action.Factory, resolver *r.Resolver, options models.ExecutionOptions) *PlanExecutor {
	stepExecutor := NewStepExecutor(resolver, options)

	return &PlanExecutor{
		factory:      factory,
		resolver:     resolver,
		stepExecutor: stepExecutor,
		options:      options,
	}
}

// ExecutePlan executes a remediation plan
func (e *PlanExecutor) ExecutePlan(plan *models.RemediationPlan) error {
	successCount := 0
	failedCount := 0

	// Execute each step
	for i := range plan.Steps {
		// Get a pointer to the step to allow modifications
		step := &plan.Steps[i]

		fmt.Printf("Executing step %d/%d: %s\n", i+1, len(plan.Steps), step.ID)

		err := e.stepExecutor.ExecuteStep(step)

		if err != nil {
			failedCount++
			if !e.options.ContinueOnError {
				return err
			}
		} else {
			successCount++
		}
	}

	// Print summary
	fmt.Printf("\nExecution summary: %d successful, %d failed (out of %d total steps)\n",
		successCount, failedCount, len(plan.Steps))

	if failedCount > 0 && !e.options.ContinueOnError {
		return fmt.Errorf("%d steps failed during execution", failedCount)
	}

	return nil
}

// GetStepOutputs returns the outputs from all executed steps
func (e *PlanExecutor) GetStepOutputs() map[string]map[string]interface{} {
	return e.stepExecutor.stepOutputs
}
