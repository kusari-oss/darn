// SPDX-License-Identifier: Apache-2.0

package darnit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kusari-oss/darn/internal/darnit/condition"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReportFile_FlatStructure(t *testing.T) {
	// Test parsing a flat report structure without "findings" wrapper
	tempDir, err := os.MkdirTemp("", "darnit_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a flat report file
	reportFile := filepath.Join(tempDir, "flat_report.json")
	flatReportContent := `{
  "security_policy": "missing",
  "mfa_status": "enabled", 
  "project_name": "Test Project",
  "organization": "test-org"
}`
	err = os.WriteFile(reportFile, []byte(flatReportContent), 0644)
	require.NoError(t, err)

	// Parse the report
	report, err := ParseReportFile(reportFile)
	require.NoError(t, err)

	// Verify flat access is available
	assert.Equal(t, "missing", report.Findings["security_policy"])
	assert.Equal(t, "enabled", report.Findings["mfa_status"])
	assert.Equal(t, "Test Project", report.Findings["project_name"])
	assert.Equal(t, "test-org", report.Findings["organization"])
}

func TestParseReportFile_WithFindingsStructure(t *testing.T) {
	// Test parsing a report that already has "findings" structure - should extract the findings content
	tempDir, err := os.MkdirTemp("", "darnit_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a report file with findings structure
	reportFile := filepath.Join(tempDir, "findings_report.json")
	findingsReportContent := `{
  "findings": {
    "security_policy": "missing",
    "mfa_status": "enabled",
    "project_name": "Test Project"
  },
  "metadata": {
    "scan_date": "2024-01-01",
    "tool_version": "1.0"
  }
}`
	err = os.WriteFile(reportFile, []byte(findingsReportContent), 0644)
	require.NoError(t, err)

	// Parse the report
	report, err := ParseReportFile(reportFile)
	require.NoError(t, err)

	// Verify the findings content is extracted for flat access
	assert.Equal(t, "missing", report.Findings["security_policy"])
	assert.Equal(t, "enabled", report.Findings["mfa_status"])
	assert.Equal(t, "Test Project", report.Findings["project_name"])

	// Metadata should not be present since only findings content is extracted
	_, hasMetadata := report.Findings["metadata"]
	assert.False(t, hasMetadata, "metadata should not be present when findings structure is extracted")
}

func TestParseReportFile_YAMLFormat(t *testing.T) {
	// Test parsing a YAML report file with flat structure
	tempDir, err := os.MkdirTemp("", "darnit_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a YAML report file
	reportFile := filepath.Join(tempDir, "report.yaml")
	yamlReportContent := `security_policy: missing
mfa_status: enabled
project_name: YAML Test Project
organization: yaml-org
failed_controls:
  - CTRL-001
  - CTRL-002
has_failed_control:
  CTRL-001: true
  CTRL-002: true
`
	err = os.WriteFile(reportFile, []byte(yamlReportContent), 0644)
	require.NoError(t, err)

	// Parse the report
	report, err := ParseReportFile(reportFile)
	require.NoError(t, err)

	// Verify flat access
	assert.Equal(t, "missing", report.Findings["security_policy"])
	assert.Equal(t, "enabled", report.Findings["mfa_status"])
	assert.Equal(t, "YAML Test Project", report.Findings["project_name"])
	assert.Equal(t, "yaml-org", report.Findings["organization"])

	// Verify array access
	failedControls, ok := report.Findings["failed_controls"].([]any)
	require.True(t, ok)
	assert.Contains(t, failedControls, "CTRL-001")
	assert.Contains(t, failedControls, "CTRL-002")

	// Verify map access
	hasFailedControl, ok := report.Findings["has_failed_control"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, hasFailedControl["CTRL-001"])
	assert.Equal(t, true, hasFailedControl["CTRL-002"])
}

func TestIntegratedCELEvaluation(t *testing.T) {
	// Test the complete workflow: parse report -> evaluate CEL expressions
	tempDir, err := os.MkdirTemp("", "darnit_cel_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a flat report file (no findings wrapper)
	reportFile := filepath.Join(tempDir, "flat_report.json")
	flatReportContent := `{
  "security_policy": "missing",
  "mfa_status": "disabled",
  "project_name": "Test Project"
}`
	err = os.WriteFile(reportFile, []byte(flatReportContent), 0644)
	require.NoError(t, err)

	// Parse the report
	report, err := ParseReportFile(reportFile)
	require.NoError(t, err)

	// Create CEL evaluator
	evaluator, err := condition.NewCELEvaluator()
	require.NoError(t, err)

	// Test flat access works
	result, err := evaluator.EvaluateExpression("security_policy == 'missing'", report.Findings)
	assert.NoError(t, err)
	assert.True(t, result, "Flat access should work after parsing")

	// Test complex expressions
	result, err = evaluator.EvaluateExpression("security_policy == 'missing' && mfa_status == 'disabled'", report.Findings)
	assert.NoError(t, err)
	assert.True(t, result, "Complex expressions should work")

	// Now test with a findings-wrapped report
	wrappedReportFile := filepath.Join(tempDir, "wrapped_report.json")
	wrappedReportContent := `{
  "findings": {
    "security_policy": "present",
    "mfa_status": "enabled"
  },
  "metadata": {
    "scan_date": "2024-01-01"
  }
}`
	err = os.WriteFile(wrappedReportFile, []byte(wrappedReportContent), 0644)
	require.NoError(t, err)

	// Parse the wrapped report (should extract findings content)
	wrappedReport, err := ParseReportFile(wrappedReportFile)
	require.NoError(t, err)

	// Test flat access works with wrapped data after extraction
	result, err = evaluator.EvaluateExpression("security_policy == 'present'", wrappedReport.Findings)
	assert.NoError(t, err)
	assert.True(t, result, "Flat access should work with findings-wrapped data after extraction")

	// Test another field from the extracted findings
	result, err = evaluator.EvaluateExpression("mfa_status == 'enabled'", wrappedReport.Findings)
	assert.NoError(t, err)
	assert.True(t, result, "Flat access to all extracted findings fields should work")
}