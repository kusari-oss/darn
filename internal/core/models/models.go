// SPDX-License-Identifier: Apache-2.0

// TODO: To avoid circular dependencies and other issues, we can move most of the internal API logic to the models package.

package models

// RemediationStep represents a single step in the remediation plan
type RemediationStep struct {
	ID         string                 `json:"id"`
	ActionName string                 `json:"action_name"`
	Params     map[string]interface{} `json:"params"`
	Reason     string                 `json:"reason"`
	DependsOn  []string               `json:"depends_on,omitempty"`
	Outputs    map[string]interface{} `json:"outputs,omitempty"`     // Capture outputs from this step
	Status     string                 `json:"status,omitempty"`      // For tracking execution: pending, running, success, failure
	Error      string                 `json:"error,omitempty"`       // Stores error message if execution fails
	OutputRefs map[string]string      `json:"output_refs,omitempty"` // References to outputs from other steps
}

// RemediationPlan represents the generated plan
type RemediationPlan struct {
	ProjectName string            `json:"project_name"`
	Repository  string            `json:"repository"`
	Steps       []RemediationStep `json:"steps"`
}

// ExecutionOptions contains options for plan execution
type ExecutionOptions struct {
	DryRun          bool
	VerboseLogging  bool
	ContinueOnError bool
	WorkingDir      string
}
