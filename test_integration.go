// +build integration

// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/kusari-oss/darn/internal/core/config"
	"github.com/kusari-oss/darn/internal/core/models"
	"github.com/kusari-oss/darn/internal/darnit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBasicWorkflow tests the basic darn workflow end-to-end
func TestBasicWorkflow(t *testing.T) {
	// Create a temporary directory for this test
	tempDir, err := ioutil.TempDir("", "darn_integration_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 1. Test configuration loading
	t.Run("ConfigurationLoad", func(t *testing.T) {
		cfg, err := config.LoadConfig("", "")
		require.NoError(t, err)
		require.NotNil(t, cfg)
		
		// Verify default configuration
		assert.Equal(t, "templates", cfg.TemplatesDir)
		assert.Equal(t, "actions", cfg.ActionsDir)
		assert.Equal(t, "configs", cfg.ConfigsDir)
		assert.Equal(t, "mappings", cfg.MappingsDir)
		assert.True(t, cfg.UseGlobal)
		assert.False(t, cfg.UseLocal)
		
		fmt.Printf("✓ Configuration loaded successfully\n")
		fmt.Printf("  Library Path: %s\n", cfg.LibraryPath)
		fmt.Printf("  Templates Dir: %s\n", cfg.TemplatesDir)
		fmt.Printf("  Actions Dir: %s\n", cfg.ActionsDir)
	})

	// 2. Test report parsing
	t.Run("ReportParsing", func(t *testing.T) {
		// Create a sample findings file
		sampleFindings := map[string]interface{}{
			"security_policy": "missing",
			"mfa_status":      "disabled",
			"repository":      "test-repo",
			"organization":    "test-org",
		}
		
		findingsFile := filepath.Join(tempDir, "findings.json")
		findingsData, err := json.MarshalIndent(sampleFindings, "", "  ")
		require.NoError(t, err)
		
		err = ioutil.WriteFile(findingsFile, findingsData, 0644)
		require.NoError(t, err)
		
		// Parse the report
		report, err := darnit.ParseReportFile(findingsFile)
		require.NoError(t, err)
		require.NotNil(t, report)
		
		// Verify parsed data
		assert.Equal(t, "missing", report.Findings["security_policy"])
		assert.Equal(t, "disabled", report.Findings["mfa_status"])
		assert.Equal(t, "test-repo", report.Findings["repository"])
		assert.Equal(t, "test-org", report.Findings["organization"])
		
		fmt.Printf("✓ Report parsed successfully\n")
		fmt.Printf("  Security Policy: %v\n", report.Findings["security_policy"])
		fmt.Printf("  MFA Status: %v\n", report.Findings["mfa_status"])
	})

	// 3. Test plan validation
	t.Run("PlanValidation", func(t *testing.T) {
		// Create a valid plan
		validPlan := &models.RemediationPlan{
			ProjectName: "test-project",
			Repository:  "test-repo",
			Steps: []models.RemediationStep{
				{
					ID:         "step1",
					ActionName: "add-security-md",
					Params: map[string]interface{}{
						"name":   "Test Project",
						"emails": []string{"security@example.com"},
					},
					Reason: "Add security documentation",
				},
				{
					ID:         "step2",
					ActionName: "enable-mfa",
					Params: map[string]interface{}{
						"organization": "test-org",
					},
					Reason:    "Enable MFA for organization",
					DependsOn: []string{"step1"},
				},
			},
		}
		
		err := darnit.ValidatePlan(validPlan)
		require.NoError(t, err)
		
		fmt.Printf("✓ Valid plan validation passed\n")
		fmt.Printf("  Project: %s\n", validPlan.ProjectName)
		fmt.Printf("  Steps: %d\n", len(validPlan.Steps))
		
		// Test invalid plan (circular dependency)
		invalidPlan := &models.RemediationPlan{
			ProjectName: "test-project",
			Repository:  "test-repo",
			Steps: []models.RemediationStep{
				{
					ID:         "step1",
					ActionName: "action1",
					DependsOn:  []string{"step2"},
				},
				{
					ID:         "step2",
					ActionName: "action2",
					DependsOn:  []string{"step1"},
				},
			},
		}
		
		err = darnit.ValidatePlan(invalidPlan)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circular dependency")
		
		fmt.Printf("✓ Invalid plan validation correctly failed\n")
	})

	// 4. Test plan file operations
	t.Run("PlanFileOperations", func(t *testing.T) {
		plan := &models.RemediationPlan{
			ProjectName: "test-project",
			Repository:  "test-repo",
			Steps: []models.RemediationStep{
				{
					ID:         "test-step",
					ActionName: "test-action",
					Params: map[string]interface{}{
						"param1": "value1",
						"param2": 42,
					},
					Reason: "Test step",
				},
			},
		}
		
		// Save plan to file
		planFile := filepath.Join(tempDir, "test-plan.json")
		err := darnit.SavePlanToFile(plan, planFile)
		require.NoError(t, err)
		
		// Load plan from file
		loadedPlan, err := darnit.LoadPlanFile(planFile)
		require.NoError(t, err)
		require.NotNil(t, loadedPlan)
		
		// Verify loaded plan
		assert.Equal(t, plan.ProjectName, loadedPlan.ProjectName)
		assert.Equal(t, plan.Repository, loadedPlan.Repository)
		assert.Len(t, loadedPlan.Steps, 1)
		assert.Equal(t, plan.Steps[0].ID, loadedPlan.Steps[0].ID)
		assert.Equal(t, plan.Steps[0].ActionName, loadedPlan.Steps[0].ActionName)
		assert.Equal(t, plan.Steps[0].Reason, loadedPlan.Steps[0].Reason)
		
		fmt.Printf("✓ Plan file operations successful\n")
		fmt.Printf("  Plan saved to: %s\n", planFile)
		fmt.Printf("  Plan loaded with %d steps\n", len(loadedPlan.Steps))
	})

	// 5. Test path expansion
	t.Run("PathExpansion", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)
		
		// Test tilde expansion
		expanded := config.ExpandPathWithTilde("~/test/path")
		expected := filepath.Join(homeDir, "test/path")
		assert.Equal(t, expected, expanded)
		
		// Test absolute path (should not change)
		absolutePath := "/absolute/path"
		expanded = config.ExpandPathWithTilde(absolutePath)
		assert.Equal(t, absolutePath, expanded)
		
		// Test relative path (should not change)
		relativePath := "relative/path"
		expanded = config.ExpandPathWithTilde(relativePath)
		assert.Equal(t, relativePath, expanded)
		
		fmt.Printf("✓ Path expansion working correctly\n")
		fmt.Printf("  Home dir: %s\n", homeDir)
		fmt.Printf("  Tilde expansion: ~/test/path -> %s\n", expanded)
	})

	fmt.Printf("\n✅ All integration tests passed successfully!\n")
}

// TestDefaultsAvailable tests that default resources are available
func TestDefaultsAvailable(t *testing.T) {
	// This test verifies that the embedded defaults are accessible
	// It doesn't test the full library system but ensures basic resources exist
	
	t.Run("DefaultConfiguration", func(t *testing.T) {
		cfg := config.NewDefaultConfig()
		require.NotNil(t, cfg)
		
		// Check that default values are sensible
		assert.Equal(t, "templates", cfg.TemplatesDir)
		assert.Equal(t, "actions", cfg.ActionsDir)
		assert.Equal(t, "configs", cfg.ConfigsDir)
		assert.Equal(t, "mappings", cfg.MappingsDir)
		assert.True(t, cfg.UseGlobal)
		assert.False(t, cfg.UseLocal)
		
		fmt.Printf("✓ Default configuration is valid\n")
	})
	
	t.Run("StateManagement", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Create a new state
		state := config.NewState(tempDir, "test-version")
		require.NotNil(t, state)
		
		// Verify state fields
		assert.Equal(t, tempDir, state.ProjectDir)
		assert.Equal(t, "test-version", state.Version)
		assert.NotEmpty(t, state.LastUpdated)
		assert.NotEmpty(t, state.InitializedAt)
		
		// Test save/load cycle
		err := config.SaveState(state, tempDir)
		require.NoError(t, err)
		
		loadedState, err := config.LoadState(tempDir)
		require.NoError(t, err)
		
		assert.Equal(t, state.ProjectDir, loadedState.ProjectDir)
		assert.Equal(t, state.Version, loadedState.Version)
		assert.Equal(t, state.LastUpdated, loadedState.LastUpdated)
		assert.Equal(t, state.InitializedAt, loadedState.InitializedAt)
		
		fmt.Printf("✓ State management working correctly\n")
	})
}