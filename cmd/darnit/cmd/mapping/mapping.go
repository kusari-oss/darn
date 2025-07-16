// SPDX-License-Identifier: Apache-2.0

package mapping

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kusari-oss/darn/internal/core/config"
	"github.com/kusari-oss/darn/internal/darnit/plan"
	"github.com/spf13/cobra"
)

// GetMappingCmd returns the mapping command
func GetMappingCmd() *cobra.Command {
	mappingCmd := &cobra.Command{
		Use:   "mapping",
		Short: "Manage remediation mappings",
		Long:  `Commands for working with remediation mappings.`,
	}

	mappingCmd.AddCommand(getListCmd())
	mappingCmd.AddCommand(getValidateCmd())
	mappingCmd.AddCommand(getAddCmd())

	return mappingCmd
}

func getListCmd() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available mappings",
		Run: func(cmd *cobra.Command, args []string) {
			// Get working directory
			workingDir, err := os.Getwd()
			if err != nil {
				fmt.Printf("Error getting working directory")
				os.Exit(1)
			}

			// Load configuration
			// For darnit mapping list, cmdLineLibraryPath is not directly applicable from this subcommand's flags.
			// It would be inherited if darnit was called with --library-path.
			// globalConfigPathOverride is also not applicable here.
			// The projectDir argument has been removed from LoadConfig.
			cfg, err := config.LoadConfig("", "")
			if err != nil {
				fmt.Printf("Error loading configuration: %v\n", err)
				os.Exit(1)
			}

			// List mappings in the mappings directory
			mappingsDir := filepath.Join(workingDir, cfg.MappingsDir)
			if _, err := os.Stat(mappingsDir); os.IsNotExist(err) {
				fmt.Printf("Mappings directory %s does not exist\n", mappingsDir)
				os.Exit(1)
			}

			files, err := os.ReadDir(mappingsDir)
			if err != nil {
				fmt.Printf("Error reading mappings directory: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("Available mapping files:")
			for _, file := range files {
				if !file.IsDir() && (filepath.Ext(file.Name()) == ".yaml" || filepath.Ext(file.Name()) == ".yml") {
					// Load mapping to show description
					mappingPath := filepath.Join(mappingsDir, file.Name())
					mapping, err := plan.LoadMappingConfig(mappingPath)
					if err != nil {
						fmt.Printf("  %s (Error: %v)\n", file.Name(), err)
						continue
					}

					fmt.Printf("  %s - %d rule(s)\n", file.Name(), len(mapping.Mappings))
					for _, rule := range mapping.Mappings {
						fmt.Printf("    - %s: %s\n", rule.ID, rule.Reason)
					}
				}
			}
		},
	}

	return listCmd
}

func getValidateCmd() *cobra.Command {
	validateCmd := &cobra.Command{
		Use:   "validate [mapping-file]",
		Short: "Validate a mapping file",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			mappingPath := args[0]

			// Load the mapping file
			mapping, err := plan.LoadMappingConfig(mappingPath)
			if err != nil {
				fmt.Printf("Error loading mapping file: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Mapping file %s is valid\n", mappingPath)
			fmt.Printf("Contains %d rule(s)\n", len(mapping.Mappings))

			// TODO: Add more validation checks
		},
	}

	return validateCmd
}

func getAddCmd() *cobra.Command {
	var interactive bool
	var condition string
	var action string
	var reason string
	var mappingID string

	addCmd := &cobra.Command{
		Use:   "add [mapping-file]",
		Short: "Create a new mapping file or add rules to existing mapping",
		Long: `Create a new mapping file or add mapping rules to an existing file.
This command helps you create custom mapping rules that link security findings to remediation actions.

Examples:
  # Create a new mapping file interactively
  darnit mapping add security-rules.yaml --interactive

  # Add a rule with flags
  darnit mapping add security-rules.yaml --id missing-policy --condition "security_policy == 'missing'" --action add-security-md --reason "Add security policy"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mappingFile := args[0]

			// Load configuration to get mappings directory
			cfg, err := config.LoadConfig("", "")
			if err != nil {
				return fmt.Errorf("error loading configuration: %w", err)
			}

			if interactive {
				return createMappingInteractive(mappingFile, cfg)
			}

			// Validate required flags for non-interactive mode
			if mappingID == "" {
				return fmt.Errorf("mapping ID is required (use --id)")
			}
			if condition == "" {
				return fmt.Errorf("condition is required (use --condition)")
			}
			if action == "" {
				return fmt.Errorf("action is required (use --action)")
			}
			if reason == "" {
				return fmt.Errorf("reason is required (use --reason)")
			}

			return createMapping(mappingFile, mappingID, condition, action, reason, cfg)
		},
	}

	// Add flags
	addCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Use interactive mode")
	addCmd.Flags().StringVar(&mappingID, "id", "", "Mapping rule ID")
	addCmd.Flags().StringVarP(&condition, "condition", "c", "", "CEL condition expression")
	addCmd.Flags().StringVarP(&action, "action", "a", "", "Action to execute")
	addCmd.Flags().StringVarP(&reason, "reason", "r", "", "Reason for this mapping")

	return addCmd
}

func createMappingInteractive(mappingFile string, cfg *config.Config) error {
	fmt.Printf("Creating mapping '%s' interactively...\n\n", mappingFile)

	var rules []plan.MappingRule

	for {
		fmt.Println("Add a new mapping rule:")

		// Prompt for mapping ID
		fmt.Print("Rule ID: ")
		var mappingID string
		fmt.Scanln(&mappingID)

		// Prompt for condition
		fmt.Print("Condition (CEL expression): ")
		var condition string
		fmt.Scanln(&condition)

		// Prompt for action
		fmt.Print("Action name: ")
		var action string
		fmt.Scanln(&action)

		// Prompt for reason
		fmt.Print("Reason: ")
		var reason string
		fmt.Scanln(&reason)

		// Create the rule
		rule := plan.MappingRule{
			ID:        mappingID,
			Condition: condition,
			Action:    action,
			Reason:    reason,
		}

		rules = append(rules, rule)

		// Ask if user wants to add more rules
		fmt.Print("Add another rule? (y/n): ")
		var continueStr string
		fmt.Scanln(&continueStr)
		if continueStr != "y" && continueStr != "yes" {
			break
		}
		fmt.Println()
	}

	return writeMappingFile(mappingFile, rules, cfg)
}

func createMapping(mappingFile, mappingID, condition, action, reason string, cfg *config.Config) error {
	rule := plan.MappingRule{
		ID:        mappingID,
		Condition: condition,
		Action:    action,
		Reason:    reason,
	}

	return writeMappingFile(mappingFile, []plan.MappingRule{rule}, cfg)
}

func writeMappingFile(mappingFile string, rules []plan.MappingRule, cfg *config.Config) error {
	// Resolve mappings directory
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting working directory: %w", err)
	}

	mappingsDir := filepath.Join(workingDir, cfg.MappingsDir)
	if cfg.MappingsDir == "" {
		mappingsDir = filepath.Join(workingDir, "mappings")
	}

	// Ensure mappings directory exists
	if err := os.MkdirAll(mappingsDir, 0755); err != nil {
		return fmt.Errorf("error creating mappings directory: %w", err)
	}

	// Determine full file path
	var mappingFilePath string
	if filepath.IsAbs(mappingFile) {
		mappingFilePath = mappingFile
	} else {
		mappingFilePath = filepath.Join(mappingsDir, mappingFile)
	}

	// Ensure .yaml extension
	if !strings.HasSuffix(mappingFilePath, ".yaml") && !strings.HasSuffix(mappingFilePath, ".yml") {
		mappingFilePath += ".yaml"
	}

	// Check if file exists and load existing rules
	var existingRules []plan.MappingRule
	if _, err := os.Stat(mappingFilePath); err == nil {
		// File exists, load existing rules
		existingMapping, err := plan.LoadMappingConfig(mappingFilePath)
		if err != nil {
			return fmt.Errorf("error loading existing mapping file: %w", err)
		}
		existingRules = existingMapping.Mappings
	}

	// Combine existing and new rules
	allRules := append(existingRules, rules...)

	// Create mapping content
	var content strings.Builder
	content.WriteString("# Mapping rules for security remediation\n")
	content.WriteString("# This file defines conditions that trigger specific remediation actions\n\n")
	content.WriteString("mappings:\n")

	for _, rule := range allRules {
		content.WriteString(fmt.Sprintf("  - id: \"%s\"\n", rule.ID))
		content.WriteString(fmt.Sprintf("    condition: \"%s\"\n", rule.Condition))
		content.WriteString(fmt.Sprintf("    action: \"%s\"\n", rule.Action))
		content.WriteString(fmt.Sprintf("    reason: \"%s\"\n", rule.Reason))

		// Add parameters section with example
		content.WriteString("    parameters:\n")
		content.WriteString("      # Add your parameters here\n")
		content.WriteString("      # Example:\n")
		content.WriteString("      # project_name: \"{{.project_name}}\"\n")
		content.WriteString("      # security_email: \"{{.security_email}}\"\n")
		content.WriteString("\n")
	}

	// Write mapping file
	if err := os.WriteFile(mappingFilePath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("error writing mapping file: %w", err)
	}

	if len(existingRules) > 0 {
		fmt.Printf("✅ Added %d new rule(s) to existing mapping file: %s\n", len(rules), mappingFilePath)
		fmt.Printf("📊 File now contains %d total rule(s)\n", len(allRules))
	} else {
		fmt.Printf("✅ Mapping file '%s' created successfully at %s\n", filepath.Base(mappingFilePath), mappingFilePath)
		fmt.Printf("📊 File contains %d rule(s)\n", len(rules))
	}

	fmt.Printf("\n📝 Edit the mapping file to add parameters and customize behavior\n")
	fmt.Printf("🔍 Use 'darnit mapping validate %s' to validate the mapping\n", mappingFilePath)

	return nil
}
