// SPDX-License-Identifier: Apache-2.0

package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kusari-oss/darn/internal/core/models"
	"github.com/kusari-oss/darn/internal/core/schema"
	"github.com/kusari-oss/darn/internal/darn/resolver"
	. "github.com/kusari-oss/darn/internal/darnit"
	"github.com/kusari-oss/darn/internal/darnit/condition"
	"gopkg.in/yaml.v3"
)

var paramRegex = regexp.MustCompile(`\{\{\.([^}]+)\}\}`)

// MappingRule defines a rule for mapping a finding to an action or other mapping.
type MappingRule struct {
	ID         string                 `yaml:"id"`
	Condition  string                 `yaml:"condition"`             // CEL expression
	MappingRef string                 `yaml:"mapping_ref,omitempty"` // New field for allowing submappings
	Action     string                 `yaml:"action,omitempty"`      // Make optional
	Reason     string                 `yaml:"reason"`
	Labels     map[string][]string    `yaml:"labels,omitempty"`
	Parameters map[string]interface{} `yaml:"parameters,omitempty"`
	DependsOn  []string               `yaml:"depends_on,omitempty"`
	Once       bool                   `yaml:"once,omitempty"`
	Steps      []MappingRule          `yaml:"steps,omitempty"` // New field for sub-steps
}

// MappingConfig contains all mapping rules
type MappingConfig struct {
	Mappings []MappingRule `yaml:"mappings"`
}

// LoadMappingConfig loads the mapping configuration from a file
func LoadMappingConfig(filePath string) (*MappingConfig, error) {
	// Read the mapping file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading mapping file: %w", err)
	}

	// Parse YAML
	var config MappingConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing mapping file: %w", err)
	}

	return &config, nil
}

// getRequiredParametersFromMappings extracts all required parameters from mapping rules
func getRequiredParametersFromMappings(mappingConfig *MappingConfig) []string {
	// Use a map to deduplicate parameters
	requiredParams := make(map[string]bool)

	for _, rule := range mappingConfig.Mappings {
		// Extract parameters from template strings in rule parameters
		for _, paramValue := range rule.Parameters {
			switch v := paramValue.(type) {
			case string:
				// Find all {{.param}} patterns
				for _, match := range paramRegex.FindAllStringSubmatch(v, -1) {
					if len(match) > 1 {
						requiredParams[match[1]] = true
					}
				}
			case []interface{}:
				// Check array items
				for _, item := range v {
					if str, ok := item.(string); ok {
						for _, match := range paramRegex.FindAllStringSubmatch(str, -1) {
							if len(match) > 1 {
								requiredParams[match[1]] = true
							}
						}
					}
				}
			}
		}
	}

	// Convert map keys to slice
	result := make([]string, 0, len(requiredParams))
	for param := range requiredParams {
		result = append(result, param)
	}

	return result
}

// ProcessMappingRule processes a single mapping rule and adds it to the plan
func ProcessMappingRule(rule MappingRule, plan *models.RemediationPlan,
	combinedData map[string]interface{}, resolver *resolver.Resolver,
	addedSteps map[string]bool, options GenerateOptions, mappingRefHistory []string) error {

	// Check for circular references
	if rule.MappingRef != "" {
		for _, ref := range mappingRefHistory {
			if ref == rule.MappingRef {
				return fmt.Errorf("circular mapping reference detected: %s -> %s",
					strings.Join(mappingRefHistory, " -> "), rule.MappingRef)
			}
		}
	}

	// Check if rule matches using CEL expressions
	matches, err := EvaluateRuleMatch(rule, combinedData, options)
	if err != nil {
		return fmt.Errorf("error evaluating rule %s: %w", rule.ID, err)
	}

	if !matches {
		return nil // No match, skip this rule
	}

	// Check for mapping reference
	if rule.MappingRef != "" {
		if options.VerboseLogging {
			fmt.Printf("Processing mapping reference: %s\n", rule.MappingRef)
		}

		// Construct the mapping file path
		mappingPath := rule.MappingRef
		if !filepath.IsAbs(mappingPath) && options.MappingsDir != "" {
			mappingPath = filepath.Join(options.MappingsDir, mappingPath)
		}

		// Load the referenced mapping file
		referencedMapping, err := LoadMappingConfig(mappingPath)
		if err != nil {
			return fmt.Errorf("error loading referenced mapping %s: %w", rule.MappingRef, err)
		}

		// Create new history slice with current reference
		newHistory := append(mappingRefHistory, rule.MappingRef)

		// Process each mapping in the referenced file
		for _, mapping := range referencedMapping.Mappings {
			if len(mapping.Steps) > 0 {
				// Process each step with parent context
				for i, step := range mapping.Steps {
					// Create a copy of the step to avoid modifying the original
					stepCopy := step

					// Generate a unique ID by combining parent and step IDs
					if rule.ID != "" && stepCopy.ID != "" {
						stepCopy.ID = fmt.Sprintf("%s-%s", rule.ID, stepCopy.ID)
					}

					// Merge parameters from parent
					if stepCopy.Parameters == nil {
						stepCopy.Parameters = make(map[string]interface{})
					}

					// Apply parent parameters (override child parameters)
					for k, v := range rule.Parameters {
						stepCopy.Parameters[k] = v
					}

					// Update dependencies for the step
					if len(stepCopy.DependsOn) > 0 {
						updatedDeps := make([]string, len(stepCopy.DependsOn))
						for j, dep := range stepCopy.DependsOn {
							// Prefix dependency with parent ID if needed
							if rule.ID != "" {
								updatedDeps[j] = fmt.Sprintf("%s-%s", rule.ID, dep)
							} else {
								updatedDeps[j] = dep
							}
						}
						stepCopy.DependsOn = updatedDeps
					} else if i > 0 && rule.ID != "" {
						// If no explicit dependencies and not the first step,
						// depend on the previous step from this mapping
						prevStepID := mapping.Steps[i-1].ID
						if prevStepID != "" {
							stepCopy.DependsOn = []string{fmt.Sprintf("%s-%s", rule.ID, prevStepID)}
						}
					}

					// Process the step recursively
					if err := ProcessMappingRule(stepCopy, plan, combinedData,
						resolver, addedSteps, options, newHistory); err != nil {
						return err
					}
				}

				// Steps were processed, continue to next mapping
				continue
			}

			// If the mapping has no steps but has an action, process it directly
			if mapping.Action != "" {
				actionRule := mapping

				// Generate a unique ID
				if rule.ID != "" && actionRule.ID != "" {
					actionRule.ID = fmt.Sprintf("%s-%s", rule.ID, actionRule.ID)
				}

				// Merge parameters
				if actionRule.Parameters == nil {
					actionRule.Parameters = make(map[string]interface{})
				}
				for k, v := range rule.Parameters {
					actionRule.Parameters[k] = v
				}

				// Process the action rule
				if err := ProcessMappingRule(actionRule, plan, combinedData,
					resolver, addedSteps, options, newHistory); err != nil {
					return err
				}
			}
		}

		return nil
	}

	// Process sub-steps if available
	if len(rule.Steps) > 0 {
		// Process each sub-step
		for _, subStep := range rule.Steps {
			// Process the sub-step (no condition inheritance needed anymore)
			if err := ProcessMappingRule(subStep, plan, combinedData, resolver,
				addedSteps, options, mappingRefHistory); err != nil {
				return err
			}
		}
		return nil
	}

	// This is a regular action step, process it
	if rule.Action == "" {
		return fmt.Errorf("rule '%s' has no action, steps, or mapping reference", rule.ID)
	}

	// If this is a "once" rule and we've already added it, skip
	if rule.Once && addedSteps[rule.Action] {
		if options.VerboseLogging {
			fmt.Printf("Skipping duplicate action '%s' (once: true)\n", rule.Action)
		}
		return nil
	}

	// Get action schema for type-aware parameter processing
	actionConfig, err := resolver.GetActionConfig(rule.Action)
	if err != nil {
		return fmt.Errorf("error getting action config for rule %s: %w", rule.ID, err)
	}

	// Process the parameters with schema awareness
	processedParams, err := schema.ProcessParamsWithSchema(rule.Parameters, combinedData, actionConfig.Schema)
	if err != nil {
		return fmt.Errorf("error processing parameters for rule %s: %w", rule.ID, err)
	}

	// Add the step to the plan
	plan.Steps = append(plan.Steps, models.RemediationStep{
		ID:         rule.ID,
		ActionName: rule.Action,
		Params:     processedParams,
		Reason:     rule.Reason,
		DependsOn:  rule.DependsOn,
	})

	// Mark this action as added (for "once: true" handling)
	addedSteps[rule.Action] = true

	return nil
}

// TODO: See if we should remove this? Originally I had a custom condition evaluator. If we include Rego or similar, this function makes sense.
// If we don't, we can remove this function.
// EvaluateRuleMatch checks if the rule matches using either CEL or a different type of condition
func EvaluateRuleMatch(rule MappingRule, data map[string]interface{}, options GenerateOptions) (bool, error) {
	if rule.Condition != "" {
		// Create evaluator (consider caching this)
		evaluator, err := condition.NewCELEvaluator()
		if err != nil {
			return false, err
		}

		return evaluator.EvaluateExpression(rule.Condition, data)
	}

	// No conditions specified, match by default
	return true, nil
}

// GenerateRemediationPlan creates a remediation plan based on the report and additional sources
func GenerateRemediationPlan(report *Report, mappingFilePath string, options GenerateOptions) (*models.RemediationPlan, error) {
	// Load mapping configuration
	mappingConfig, err := LoadMappingConfig(mappingFilePath)
	if err != nil {
		return nil, fmt.Errorf("error loading mapping configuration: %w", err)
	}

	// Create action resolver for schema access
	_, resolver, err := CreateActionResolver(options.RepoPath)
	if err != nil {
		return nil, fmt.Errorf("error creating action resolver: %w", err)
	}

	// Create combined data from all sources
	combinedData := make(map[string]interface{})

	// 1. Start with default parameters (lowest priority)
	if !options.SkipDefaults {
		if options.VerboseLogging {
			fmt.Println("Loading default parameters...")
		}
		defaults, err := LoadDefaultParameters(options.DefaultsPath)
		if err != nil {
			// Just log this error but continue
			fmt.Fprintf(os.Stderr, "Warning: Error loading default parameters: %v\n", err)
		} else {
			for k, v := range defaults {
				combinedData[k] = v
				if options.VerboseLogging {
					fmt.Printf("  Default parameter: %s = %v\n", k, v)
				}
			}
		}
	}

	// 2. Add repository-inferred parameters
	if !options.SkipRepoInference {
		if options.VerboseLogging {
			fmt.Printf("Inferring parameters from repository: %s\n",
				options.RepoPath)
		}
		repoParams, err := InferParametersFromRepo(options.RepoPath)
		if err != nil {
			// Just log this error but continue
			fmt.Fprintf(os.Stderr, "Warning: Error inferring parameters from repository: %v\n", err)
		} else {
			for k, v := range repoParams {
				combinedData[k] = v
				if options.VerboseLogging {
					fmt.Printf("  Inferred parameter: %s = %v\n", k, v)
				}
			}
		}
	}

	// 3. Add report data (higher priority)
	if options.VerboseLogging {
		fmt.Println("Adding parameters from report...")
	}
	for k, v := range report.Findings {
		combinedData[k] = v
		if options.VerboseLogging {
			fmt.Printf("  Report parameter: %s = %v\n", k, v)
		}
	}

	// 4. Add explicitly provided parameters (highest priority)
	if options.VerboseLogging && len(options.ExtraParams) > 0 {
		fmt.Println("Adding explicitly provided parameters...")
	}
	for k, v := range options.ExtraParams {
		combinedData[k] = v
		if options.VerboseLogging {
			fmt.Printf("  Explicit parameter: %s = %v\n", k, v)
		}
	}

	// 5. Prompt for any missing required parameters
	if !options.NonInteractive {
		// Determine required parameters from mapping rules
		requiredParams := getRequiredParametersFromMappings(mappingConfig)
		if options.VerboseLogging {
			fmt.Printf("Required parameters: %s\n", strings.Join(requiredParams, ", "))
		}

		if err := PromptForMissingParameters(combinedData, requiredParams); err != nil {
			return nil, fmt.Errorf("error prompting for parameters: %w", err)
		}
	}

	// Extract values for plan metadata
	projectName, _ := combinedData["project_name"].(string)
	if projectName == "" {
		projectName = "Unknown Project"
	}

	repository, _ := combinedData["project_repo"].(string)
	if repository == "" {
		repository = "Unknown Repository"
	}

	// Create the plan
	plan := &models.RemediationPlan{
		ProjectName: projectName,
		Repository:  repository,
		Steps:       []models.RemediationStep{},
	}

	// Keep track of steps we've already added (for "once: true" steps)
	addedSteps := make(map[string]bool)

	// Process each mapping rule
	for _, rule := range mappingConfig.Mappings {
		if err := ProcessMappingRule(rule, plan, combinedData, resolver,
			addedSteps, options, []string{}); err != nil {
			return nil, err
		}
	}

	// Sort steps based on dependencies
	if err := sortStepsByDependencies(plan); err != nil {
		return nil, fmt.Errorf("error sorting steps by dependencies: %w", err)
	}

	if options.VerboseLogging {
		fmt.Printf("Generated remediation plan with %d steps\n", len(plan.Steps))
	}

	return plan, nil
}

// sortStepsByDependencies sorts the steps based on their dependencies
func sortStepsByDependencies(plan *models.RemediationPlan) error {
	// Check for circular dependencies
	if err := DetectCycles(plan.Steps); err != nil {
		return err
	}

	// Build dependency graph
	stepMap := make(map[string]int) // Maps step ID to its index
	for i, step := range plan.Steps {
		stepMap[step.ID] = i
	}

	// Check if all dependencies exist
	for _, step := range plan.Steps {
		for _, depID := range step.DependsOn {
			if _, exists := stepMap[depID]; !exists {
				return fmt.Errorf("step '%s' depends on non-existent step '%s'", step.ID, depID)
			}
		}
	}

	// Perform topological sort
	// This is a simple implementation - a more sophisticated one might be needed
	// for complex dependency graphs

	// Create a new sorted slice
	numSteps := len(plan.Steps)
	sortedSteps := make([]models.RemediationStep, 0, numSteps)
	visited := make(map[string]bool)

	// Helper function to visit nodes in DFS order
	var visit func(step models.RemediationStep) error
	visit = func(step models.RemediationStep) error {
		if visited[step.ID] {
			return nil // Already visited
		}

		visited[step.ID] = true

		// Visit all dependencies first
		for _, depID := range step.DependsOn {
			depStep := plan.Steps[stepMap[depID]]
			if err := visit(depStep); err != nil {
				return err
			}
		}

		// Add this step
		sortedSteps = append(sortedSteps, step)
		return nil
	}

	// Visit all steps
	for _, step := range plan.Steps {
		if !visited[step.ID] {
			if err := visit(step); err != nil {
				return err
			}
		}
	}

	// Update the plan with sorted steps
	plan.Steps = sortedSteps
	return nil
}
