// SPDX-License-Identifier: Apache-2.0

package darnit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kusari-oss/darn/internal/core/action"
	"github.com/kusari-oss/darn/internal/core/config"
	"github.com/kusari-oss/darn/internal/core/models"
	"github.com/kusari-oss/darn/internal/darn/resolver"
	"github.com/kusari-oss/darn/internal/darnit/executor"
	"gopkg.in/yaml.v3"
)

// Report represents the parsed report data
type Report struct {
	Findings map[string]interface{}
}

// GenerateOptions contains options for plan generation
type GenerateOptions struct {
	DefaultsPath      string
	RepoPath          string
	MappingsDir       string
	ExtraParams       map[string]interface{}
	SkipDefaults      bool
	SkipRepoInference bool
	NonInteractive    bool
	VerboseLogging    bool
}

// ParseReportFile reads and parses a report file
func ParseReportFile(filePath string) (*Report, error) {
	// Read the report file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading report file: %w", err)
	}

	// First try JSON format
	var reportData map[string]interface{}
	err = json.Unmarshal(data, &reportData)
	if err != nil {
		// If JSON fails, try YAML
		err = yaml.Unmarshal(data, &reportData)
		if err != nil {
			return nil, fmt.Errorf("error parsing report (tried JSON and YAML): %w", err)
		}
	}

	return &Report{Findings: reportData}, nil
}

// LoadPlanFile loads a remediation plan from a file
func LoadPlanFile(filePath string) (*models.RemediationPlan, error) {
	// Read the plan file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading plan file: %w", err)
	}

	// Parse JSON
	var plan models.RemediationPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("error parsing plan JSON: %w", err)
	}

	return &plan, nil
}

// ExecutePlan executes a remediation plan
func ExecutePlan(plan *models.RemediationPlan, options models.ExecutionOptions) error {
	// Create action factory and resolver
	factory, resolver, err := CreateActionResolver(options.WorkingDir)
	if err != nil {
		return fmt.Errorf("error creating action resolver: %w", err)
	}

	// Create and use the plan executor
	planExecutor := executor.NewPlanExecutor(factory, resolver, options)

	// Execute the plan
	if err := planExecutor.ExecutePlan(plan); err != nil {
		return err
	}

	return nil
}

// CreateActionResolver creates the action factory and resolver
func CreateActionResolver(workingDir string) (*action.Factory, *resolver.Resolver, error) {
	// If working directory not specified, use current directory
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			return nil, nil, fmt.Errorf("error getting working directory: %w", err)
		}
	}

	// Load configuration
	cfg, err := config.LoadConfig(workingDir)
	if err != nil {
		return nil, nil, fmt.Errorf("error loading configuration: %w", err)
	}

	// Create action context with both local and global template directories
	context := action.ActionContext{
		TemplatesDir:       filepath.Join(workingDir, cfg.TemplatesDir),
		GlobalTemplatesDir: filepath.Join(cfg.LibraryPath, "templates"),
		WorkingDir:         workingDir,
		VerboseMode:        false, // Default value, will be overridden by options
		UseLocal:           cfg.UseLocal,
		UseGlobal:          cfg.UseGlobal,
		GlobalFirst:        cfg.GlobalFirst,
	}

	// Create action factory with context
	factory := action.NewFactory(context)

	// Register default action types
	factory.RegisterDefaultTypes()

	// Create resolver
	resolver := resolver.NewResolver(
		factory,
		workingDir,
		cfg.UseLocal,
		cfg.UseGlobal,
		cfg.GlobalFirst,
		cfg.ActionsDir,
		cfg.LibraryPath,
	)

	return factory, resolver, nil
}

// LoadDefaultParameters loads default parameters from config
func LoadDefaultParameters(configPath string) (map[string]interface{}, error) {
	if configPath == "" {
		// Look in standard locations
		candidates := []string{
			"./params.yaml",
			"./.darn/params.yaml",
			os.ExpandEnv("$HOME/.darn/params.yaml"),
		}

		for _, path := range candidates {
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}

		if configPath == "" {
			return make(map[string]interface{}), nil
		}
	}

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse YAML
	var config struct {
		DefaultParameters map[string]interface{} `yaml:"default_parameters"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config.DefaultParameters, nil
}

// PromptForMissingParameters asks the user for any missing required parameters
func PromptForMissingParameters(data map[string]interface{}, requiredParams []string) error {
	for _, param := range requiredParams {
		if _, exists := data[param]; !exists {
			fmt.Printf("Required parameter '%s' is missing. Please enter a value: ", param)
			var value string
			if _, err := fmt.Scanln(&value); err != nil {
				return fmt.Errorf("error reading input: %w", err)
			}
			data[param] = value
		}
	}
	return nil
}

// ValidatePlan checks if a remediation plan is valid
func ValidatePlan(plan *models.RemediationPlan) error {
	// Check for empty plan
	if len(plan.Steps) == 0 {
		return fmt.Errorf("plan contains no steps")
	}

	// Check for duplicate step IDs
	stepIDs := make(map[string]bool)
	for _, step := range plan.Steps {
		if step.ID == "" {
			return fmt.Errorf("step has empty ID")
		}

		if stepIDs[step.ID] {
			return fmt.Errorf("duplicate step ID: %s", step.ID)
		}

		stepIDs[step.ID] = true
	}

	// Check for missing action names
	for _, step := range plan.Steps {
		if step.ActionName == "" {
			return fmt.Errorf("step '%s' has empty action name", step.ID)
		}
	}

	// Check for valid dependencies
	for _, step := range plan.Steps {
		for _, depID := range step.DependsOn {
			if !stepIDs[depID] {
				return fmt.Errorf("step '%s' depends on non-existent step '%s'", step.ID, depID)
			}
		}
	}

	// Check for circular dependencies
	if err := DetectCycles(plan.Steps); err != nil {
		return err
	}

	return nil
}

// SavePlanToFile saves a remediation plan to a file
func SavePlanToFile(plan *models.RemediationPlan, filePath string) error {
	// Validate the plan
	if err := ValidatePlan(plan); err != nil {
		return fmt.Errorf("invalid plan: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling plan to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing plan to file: %w", err)
	}

	return nil
}

// DetectCycles checks for circular dependencies in the steps
func DetectCycles(steps []models.RemediationStep) error {
	// Build dependency graph
	stepMap := make(map[string]models.RemediationStep)
	for _, step := range steps {
		stepMap[step.ID] = step
	}

	// Check each step for cycles
	for _, step := range steps {
		visited := make(map[string]bool)
		path := make(map[string]bool)

		if cycle := findCycle(step.ID, stepMap, visited, path); cycle != "" {
			return fmt.Errorf("circular dependency detected: %s", cycle)
		}
	}

	return nil
}

// findCycle performs DFS to find cycles in the dependency graph
func findCycle(
	nodeID string,
	graph map[string]models.RemediationStep,
	visited map[string]bool,
	path map[string]bool,
) string {
	if path[nodeID] {
		// Found a cycle
		return nodeID
	}

	if visited[nodeID] {
		// Already checked, no cycle
		return ""
	}

	// Mark this node as part of current path
	visited[nodeID] = true
	path[nodeID] = true

	// Visit all dependencies
	node, exists := graph[nodeID]
	if exists {
		for _, depID := range node.DependsOn {
			if cycle := findCycle(depID, graph, visited, path); cycle != "" {
				// Format the cycle path nicely
				if cycle == nodeID {
					return fmt.Sprintf("%s -> %s", nodeID, cycle)
				} else {
					return fmt.Sprintf("%s -> %s", nodeID, cycle)
				}
			}
		}
	}

	// Remove from current path
	path[nodeID] = false
	return ""
}
