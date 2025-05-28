// SPDX-License-Identifier: Apache-2.0

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemediationStep(t *testing.T) {
	t.Run("BasicStepCreation", func(t *testing.T) {
		step := RemediationStep{
			ID:         "test-step",
			ActionName: "test-action",
			Params: map[string]interface{}{
				"param1": "value1",
				"param2": 42,
				"param3": true,
			},
			Reason: "Test step for validation",
		}

		assert.Equal(t, "test-step", step.ID)
		assert.Equal(t, "test-action", step.ActionName)
		assert.Equal(t, "Test step for validation", step.Reason)
		assert.Equal(t, "value1", step.Params["param1"])
		assert.Equal(t, 42, step.Params["param2"])
		assert.Equal(t, true, step.Params["param3"])
	})

	t.Run("StepWithDependencies", func(t *testing.T) {
		step := RemediationStep{
			ID:         "dependent-step",
			ActionName: "test-action",
			Params:     map[string]interface{}{},
			Reason:     "This step depends on others",
			DependsOn:  []string{"step1", "step2"},
		}

		assert.Equal(t, "dependent-step", step.ID)
		assert.Len(t, step.DependsOn, 2)
		assert.Contains(t, step.DependsOn, "step1")
		assert.Contains(t, step.DependsOn, "step2")
	})

	t.Run("StepWithOutputs", func(t *testing.T) {
		step := RemediationStep{
			ID:         "output-step",
			ActionName: "test-action",
			Params:     map[string]interface{}{},
			Reason:     "This step produces outputs",
			Outputs: map[string]interface{}{
				"result":   "success",
				"file_path": "/tmp/output.txt",
			},
		}

		assert.Equal(t, "output-step", step.ID)
		assert.Equal(t, "success", step.Outputs["result"])
		assert.Equal(t, "/tmp/output.txt", step.Outputs["file_path"])
	})

	t.Run("StepStatus", func(t *testing.T) {
		step := RemediationStep{
			ID:         "status-step",
			ActionName: "test-action",
			Params:     map[string]interface{}{},
			Reason:     "Testing status tracking",
			Status:     "pending",
		}

		assert.Equal(t, "pending", step.Status)

		// Test status transitions
		step.Status = "running"
		assert.Equal(t, "running", step.Status)

		step.Status = "success"
		assert.Equal(t, "success", step.Status)

		step.Status = "failure"
		step.Error = "Something went wrong"
		assert.Equal(t, "failure", step.Status)
		assert.Equal(t, "Something went wrong", step.Error)
	})

	t.Run("StepWithOutputRefs", func(t *testing.T) {
		step := RemediationStep{
			ID:         "ref-step",
			ActionName: "test-action",
			Params:     map[string]interface{}{},
			Reason:     "This step references outputs from other steps",
			OutputRefs: map[string]string{
				"input_file": "previous-step.file_path",
				"config":     "config-step.config_data",
			},
		}

		assert.Equal(t, "ref-step", step.ID)
		assert.Equal(t, "previous-step.file_path", step.OutputRefs["input_file"])
		assert.Equal(t, "config-step.config_data", step.OutputRefs["config"])
	})
}

func TestRemediationPlan(t *testing.T) {
	t.Run("BasicPlanCreation", func(t *testing.T) {
		plan := RemediationPlan{
			ProjectName: "test-project",
			Repository:  "github.com/example/test-repo",
			Steps: []RemediationStep{
				{
					ID:         "step1",
					ActionName: "action1",
					Params:     map[string]interface{}{"param": "value"},
					Reason:     "First step",
				},
				{
					ID:         "step2",
					ActionName: "action2",
					Params:     map[string]interface{}{"param": "value"},
					Reason:     "Second step",
					DependsOn:  []string{"step1"},
				},
			},
		}

		assert.Equal(t, "test-project", plan.ProjectName)
		assert.Equal(t, "github.com/example/test-repo", plan.Repository)
		assert.Len(t, plan.Steps, 2)
		assert.Equal(t, "step1", plan.Steps[0].ID)
		assert.Equal(t, "step2", plan.Steps[1].ID)
		assert.Contains(t, plan.Steps[1].DependsOn, "step1")
	})

	t.Run("EmptyPlan", func(t *testing.T) {
		plan := RemediationPlan{
			ProjectName: "empty-project",
			Repository:  "github.com/example/empty-repo",
			Steps:       []RemediationStep{},
		}

		assert.Equal(t, "empty-project", plan.ProjectName)
		assert.Equal(t, "github.com/example/empty-repo", plan.Repository)
		assert.Len(t, plan.Steps, 0)
	})

	t.Run("PlanWithComplexDependencies", func(t *testing.T) {
		plan := RemediationPlan{
			ProjectName: "complex-project",
			Repository:  "github.com/example/complex-repo",
			Steps: []RemediationStep{
				{
					ID:         "init",
					ActionName: "initialize",
					Params:     map[string]interface{}{},
					Reason:     "Initialize project",
				},
				{
					ID:         "setup-security",
					ActionName: "add-security-md",
					Params:     map[string]interface{}{},
					Reason:     "Add security documentation",
					DependsOn:  []string{"init"},
				},
				{
					ID:         "setup-mfa",
					ActionName: "enable-mfa",
					Params:     map[string]interface{}{},
					Reason:     "Enable MFA",
					DependsOn:  []string{"init"},
				},
				{
					ID:         "finalize",
					ActionName: "finalize",
					Params:     map[string]interface{}{},
					Reason:     "Finalize setup",
					DependsOn:  []string{"setup-security", "setup-mfa"},
				},
			},
		}

		assert.Equal(t, "complex-project", plan.ProjectName)
		assert.Len(t, plan.Steps, 4)
		
		// Verify dependency structure
		stepMap := make(map[string]RemediationStep)
		for _, step := range plan.Steps {
			stepMap[step.ID] = step
		}

		initStep := stepMap["init"]
		assert.Len(t, initStep.DependsOn, 0)

		securityStep := stepMap["setup-security"]
		assert.Len(t, securityStep.DependsOn, 1)
		assert.Contains(t, securityStep.DependsOn, "init")

		mfaStep := stepMap["setup-mfa"]
		assert.Len(t, mfaStep.DependsOn, 1)
		assert.Contains(t, mfaStep.DependsOn, "init")

		finalizeStep := stepMap["finalize"]
		assert.Len(t, finalizeStep.DependsOn, 2)
		assert.Contains(t, finalizeStep.DependsOn, "setup-security")
		assert.Contains(t, finalizeStep.DependsOn, "setup-mfa")
	})
}

func TestExecutionOptions(t *testing.T) {
	t.Run("DefaultOptions", func(t *testing.T) {
		options := ExecutionOptions{}

		assert.False(t, options.DryRun)
		assert.False(t, options.VerboseLogging)
		assert.False(t, options.ContinueOnError)
		assert.Empty(t, options.WorkingDir)
	})

	t.Run("CustomOptions", func(t *testing.T) {
		options := ExecutionOptions{
			DryRun:          true,
			VerboseLogging:  true,
			ContinueOnError: true,
			WorkingDir:      "/tmp/test",
		}

		assert.True(t, options.DryRun)
		assert.True(t, options.VerboseLogging)
		assert.True(t, options.ContinueOnError)
		assert.Equal(t, "/tmp/test", options.WorkingDir)
	})
}

func TestRemediationStepValidation(t *testing.T) {
	t.Run("ValidStep", func(t *testing.T) {
		step := RemediationStep{
			ID:         "valid-step",
			ActionName: "valid-action",
			Params:     map[string]interface{}{"key": "value"},
			Reason:     "Valid step for testing",
		}

		// Basic validation checks
		assert.NotEmpty(t, step.ID)
		assert.NotEmpty(t, step.ActionName)
		assert.NotEmpty(t, step.Reason)
		assert.NotNil(t, step.Params)
	})

	t.Run("StepWithRequiredFields", func(t *testing.T) {
		step := RemediationStep{
			ID:         "required-step",
			ActionName: "required-action",
			Params:     map[string]interface{}{},
			Reason:     "Step with all required fields",
		}

		// Verify required fields are present
		assert.NotEmpty(t, step.ID, "ID should not be empty")
		assert.NotEmpty(t, step.ActionName, "ActionName should not be empty")
		assert.NotEmpty(t, step.Reason, "Reason should not be empty")
		assert.NotNil(t, step.Params, "Params should not be nil")
	})
}