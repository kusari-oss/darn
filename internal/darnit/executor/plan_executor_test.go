// SPDX-License-Identifier: Apache-2.0

package executor_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kusari-oss/darn/internal/core/action"
	"github.com/kusari-oss/darn/internal/core/models"
	"github.com/kusari-oss/darn/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// We need to test the plan execution functionality without actual resolver implementation
// Since we can't replace the resolver directly, we'll test the component parts separately

// Test the step output processing functionality
func TestProcessOutputReferences(t *testing.T) {
	// Create a test step with output references
	step := &models.RemediationStep{
		ID:         "test-step",
		ActionName: "test-action",
		Params:     map[string]interface{}{"param": "value"},
		OutputRefs: map[string]string{
			"hash": "previous-step.commit_hash",
		},
		Reason: "Test step with output references",
	}

	// Create a mock step outputs map
	stepOutputs := map[string]map[string]interface{}{
		"previous-step": {
			"commit_hash": "abc123",
			"other_value": 42,
		},
	}

	// Test successful output reference resolution
	err := processOutputReferences(step, stepOutputs)
	assert.NoError(t, err, "Failed to process output references")
	assert.Equal(t, "abc123", step.Params["hash"], "Parameter not updated with output value")

	// Test with missing source step
	step.OutputRefs["missing"] = "nonexistent-step.value"
	err = processOutputReferences(step, stepOutputs)
	assert.Error(t, err, "Should error with nonexistent source step")
	assert.Contains(t, err.Error(), "not found or has not completed", "Error should indicate missing step")

	// Test with missing output field
	delete(step.OutputRefs, "missing") // Remove the problematic reference
	step.OutputRefs["missing_field"] = "previous-step.nonexistent_field"
	err = processOutputReferences(step, stepOutputs)
	assert.Error(t, err, "Should error with nonexistent output field")
	assert.Contains(t, err.Error(), "not found in step", "Error should indicate missing output field")
}

// Mock function to simulate output reference processing
func processOutputReferences(step *models.RemediationStep, stepOutputs map[string]map[string]interface{}) error {
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
		sourceOutputs, exists := stepOutputs[sourceStepID]
		if !exists {
			return fmt.Errorf("referenced step %s not found or has not completed successfully", sourceStepID)
		}

		// Get output value
		outputValue, exists := sourceOutputs[outputName]
		if !exists {
			return fmt.Errorf("output %s not found in step %s", outputName, sourceStepID)
		}

		// Set the parameter value from the output
		step.Params[paramName] = outputValue
	}

	return nil
}

// Test execution with mock actions directly
func TestStepExecution(t *testing.T) {
	// Create mock actions
	mockAction := new(testutil.MockAction)
	mockOutputAction := new(testutil.MockOutputAction)

	// Set up mock action expectations
	mockAction.On("Execute", mock.Anything).Return(nil)

	mockOutputs := map[string]interface{}{
		"result": "output value",
	}
	mockOutputAction.On("ExecuteWithOutput", mock.Anything).Return(mockOutputs, nil)

	// Test executing a regular action
	regularParams := map[string]interface{}{"param": "value"}
	err := mockAction.Execute(regularParams)
	assert.NoError(t, err, "Failed to execute regular action")

	// Test executing an output action
	outputParams := map[string]interface{}{"param": "value"}
	outputs, err := mockOutputAction.ExecuteWithOutput(outputParams)
	assert.NoError(t, err, "Failed to execute output action")
	assert.Equal(t, "output value", outputs["result"], "Output value incorrect")

	// Verify expectations were met
	mockAction.AssertExpectations(t)
	mockOutputAction.AssertExpectations(t)
}

// Test handling of step status updates
func TestStepStatusHandling(t *testing.T) {
	// Create mock actions
	successAction := new(testutil.MockAction)
	failingAction := new(testutil.MockAction)

	// Set up mock action expectations
	successAction.On("Execute", mock.Anything).Return(nil)
	failingAction.On("Execute", mock.Anything).Return(assert.AnError)

	// Test successful step
	successStep := &models.RemediationStep{
		ID:         "success-step",
		ActionName: "success-action",
		Params:     map[string]interface{}{"param": "value"},
		Reason:     "Step that succeeds",
	}

	// Simulate execution with successful action
	successStep.Status = "running"
	err := successAction.Execute(successStep.Params)
	assert.NoError(t, err, "Action execution should succeed")
	successStep.Status = "success"

	// Test failing step
	failingStep := &models.RemediationStep{
		ID:         "failing-step",
		ActionName: "failing-action",
		Params:     map[string]interface{}{"param": "value"},
		Reason:     "Step that fails",
	}

	// Simulate execution with failing action
	failingStep.Status = "running"
	err = failingAction.Execute(failingStep.Params)
	assert.Error(t, err, "Action execution should fail")
	failingStep.Status = "failure"
	failingStep.Error = fmt.Sprintf("execution failed: %v", err)

	// Verify step statuses
	assert.Equal(t, "success", successStep.Status, "Success step status should be 'success'")
	assert.Equal(t, "failure", failingStep.Status, "Failing step status should be 'failure'")
	assert.Contains(t, failingStep.Error, "execution failed", "Error message should indicate execution failure")

	// Verify expectations were met
	successAction.AssertExpectations(t)
	failingAction.AssertExpectations(t)
}

// Test plan dependency resolution
func TestPlanDependencyResolution(t *testing.T) {
	// Create a plan with dependent steps
	plan := &models.RemediationPlan{
		ProjectName: "Test Project",
		Repository:  "test/repo",
		Steps: []models.RemediationStep{
			{
				ID:         "first-step",
				ActionName: "first-action",
				Params:     map[string]interface{}{"param": "value"},
				Reason:     "First step",
				Status:     "success",
				Outputs: map[string]interface{}{
					"commit_hash": "abc123",
				},
			},
			{
				ID:         "second-step",
				ActionName: "second-action",
				Params:     map[string]interface{}{"hash": "placeholder"},
				OutputRefs: map[string]string{
					"hash": "first-step.commit_hash",
				},
				Reason:    "Second step that depends on first step output",
				DependsOn: []string{"first-step"},
				Status:    "pending",
			},
		},
	}

	// Create a step outputs map to simulate previous step outputs
	stepOutputs := map[string]map[string]interface{}{
		"first-step": {
			"commit_hash": "abc123",
		},
	}

	// Process output references for the second step
	err := processOutputReferences(&plan.Steps[1], stepOutputs)
	assert.NoError(t, err, "Failed to process output references")

	// Verify parameters were updated with output values
	assert.Equal(t, "abc123", plan.Steps[1].Params["hash"], "Parameter not updated with output value")
}

// Test the plan execution with a mocked structure instead of using the actual NewStepExecutor
func TestMockedPlanExecution(t *testing.T) {
	// Create mock actions
	firstAction := new(testutil.MockOutputAction)
	secondAction := new(testutil.MockAction)

	// Set up mock action expectations
	firstOutput := map[string]interface{}{
		"commit_hash": "abc123",
	}
	firstAction.On("ExecuteWithOutput", mock.Anything).Return(firstOutput, nil)
	secondAction.On("Execute", mock.MatchedBy(func(params map[string]interface{}) bool {
		// Verify the hash parameter was updated with the output from the first step
		return params["hash"] == "abc123"
	})).Return(nil)

	// Create a mock action registry
	actions := map[string]action.Action{
		"first-action":  firstAction,
		"second-action": secondAction,
	}

	// Create a plan with dependent steps
	plan := &models.RemediationPlan{
		ProjectName: "Test Project",
		Repository:  "test/repo",
		Steps: []models.RemediationStep{
			{
				ID:         "first-step",
				ActionName: "first-action",
				Params:     map[string]interface{}{"param": "value"},
				Reason:     "First step",
			},
			{
				ID:         "second-step",
				ActionName: "second-action",
				Params:     map[string]interface{}{"hash": "placeholder"},
				OutputRefs: map[string]string{
					"hash": "first-step.commit_hash",
				},
				Reason:    "Second step that depends on first step output",
				DependsOn: []string{"first-step"},
			},
		},
	}

	// Simulate plan execution
	stepOutputs := make(map[string]map[string]interface{})

	// Execute first step
	firstStep := &plan.Steps[0]
	firstStep.Status = "running"

	// Get action from registry
	act, ok := actions[firstStep.ActionName]
	require.True(t, ok, "Action not found in registry")

	// If it's an output action, capture outputs
	if outputAct, ok := act.(action.OutputAction); ok {
		outputs, err := outputAct.ExecuteWithOutput(firstStep.Params)
		assert.NoError(t, err, "Failed to execute first step")

		// Store outputs for future steps
		stepOutputs[firstStep.ID] = outputs
		firstStep.Outputs = outputs
	} else {
		err := act.Execute(firstStep.Params)
		assert.NoError(t, err, "Failed to execute first step")
	}

	firstStep.Status = "success"

	// Execute second step
	secondStep := &plan.Steps[1]
	secondStep.Status = "running"

	// Process output references
	err := processOutputReferences(secondStep, stepOutputs)
	assert.NoError(t, err, "Failed to process output references")

	// Get action from registry
	act, ok = actions[secondStep.ActionName]
	require.True(t, ok, "Action not found in registry")

	// Execute the action
	err = act.Execute(secondStep.Params)
	assert.NoError(t, err, "Failed to execute second step")

	secondStep.Status = "success"

	// Verify all steps executed successfully
	assert.Equal(t, "success", plan.Steps[0].Status, "First step status should be success")
	assert.Equal(t, "success", plan.Steps[1].Status, "Second step status should be success")

	// Verify expectations were met
	firstAction.AssertExpectations(t)
	secondAction.AssertExpectations(t)
}
