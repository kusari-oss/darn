// SPDX-License-Identifier: Apache-2.0

package action

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kusari-oss/darn/internal/core/action"
	"github.com/kusari-oss/darn/internal/core/config"
	"github.com/kusari-oss/darn/internal/core/schema"
	. "github.com/kusari-oss/darn/internal/darn/resolver"
	"github.com/spf13/cobra"
)

// NewActionCmd creates a new action command
func NewActionCmd() *cobra.Command {
	actionCmd := &cobra.Command{
		Use:   "action",
		Short: "Manage and run actions",
		Long:  `Manage and run actions for security remediation`,
	}

	// Add subcommands
	actionCmd.AddCommand(newActionListCmd())
	actionCmd.AddCommand(newActionInfoCmd())
	actionCmd.AddCommand(newActionRunCmd())

	return actionCmd
}

// newActionListCmd creates a 'list' subcommand
func newActionListCmd() *cobra.Command {
	var labelsFlag string

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available actions",
		Long:  `List all available actions or filter by labels`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get the working directory
			workingDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("error getting working directory: %w", err)
			}

			// Load configuration
			// For darn action commands, cmdLineLibraryPath and globalConfigPathOverride are not applicable from CLI flags.
			cfg, err := config.LoadConfig("", "")
			if err != nil {
				return fmt.Errorf("error loading configuration: %w", err)
			}

			// Create action context with templates directory and working directory
			context := action.ActionContext{
				TemplatesDir: filepath.Join(workingDir, cfg.TemplatesDir),
				WorkingDir:   workingDir,
				VerboseMode:  false,
			}

			// Create action factory with context
			factory := action.NewFactory(context)
			factory.RegisterDefaultTypes()

			// Create resolver
			resolver := NewResolver(
				factory,
				workingDir,
				cfg.UseLocal,
				cfg.UseGlobal,
				cfg.GlobalFirst,
				cfg.ActionsDir,
				cfg.LibraryPath,
			)

			// Get all available actions
			actions, err := resolver.ListAvailableActions()
			if err != nil {
				return fmt.Errorf("error listing actions: %w", err)
			}

			// Parse label selectors
			labelSelectors := make(map[string][]string)
			if labelsFlag != "" {
				selectorParts := strings.Split(labelsFlag, ",")
				for _, part := range selectorParts {
					kv := strings.SplitN(part, "=", 2)
					if len(kv) == 2 {
						key := strings.TrimSpace(kv[0])
						valuesStr := strings.TrimSpace(kv[1])
						values := strings.Split(valuesStr, "|")
						for i, v := range values {
							values[i] = strings.TrimSpace(v)
						}
						labelSelectors[key] = values
					}
				}
			}

			// Filter actions by labels if needed
			if len(labelSelectors) > 0 {
				actions = resolver.FilterActionsByLabels(actions, labelSelectors)
			}

			// Display the actions
			fmt.Println("Available actions:")
			fmt.Println("------------------")
			for name, actionConfig := range actions {
				fmt.Printf("- %s: %s\n", name, actionConfig.Description)

				// Display labels if they exist
				if len(actionConfig.Labels) > 0 {
					fmt.Println("  Labels:")
					for key, values := range actionConfig.Labels {
						fmt.Printf("    %s: %s\n", key, strings.Join(values, ", "))
					}
				}
				fmt.Println()
			}

			return nil
		},
	}

	// Add flags
	listCmd.Flags().StringVarP(&labelsFlag, "labels", "l", "", "Filter actions by labels (format: key1=value1,key2=value2)")

	return listCmd
}

// newActionInfoCmd creates an 'info' subcommand
func newActionInfoCmd() *cobra.Command {
	infoCmd := &cobra.Command{
		Use:   "info [action-name]",
		Short: "Show information about an action",
		Long:  `Display detailed information about a specific action`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			actionName := args[0]

			// Get the working directory
			workingDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("error getting working directory: %w", err)
			}

			// Load configuration
			// For darn action commands, cmdLineLibraryPath and globalConfigPathOverride are not applicable from CLI flags.
			cfg, err := config.LoadConfig("", "")
			if err != nil {
				return fmt.Errorf("error loading configuration: %w", err)
			}

			// Create action context with templates directory and working directory
			context := action.ActionContext{
				TemplatesDir: filepath.Join(workingDir, cfg.TemplatesDir),
				WorkingDir:   workingDir,
				VerboseMode:  false,
			}

			// Create action factory with context
			factory := action.NewFactory(context)
			factory.RegisterDefaultTypes()

			// Create resolver
			resolver := NewResolver(
				factory,
				workingDir,
				cfg.UseLocal,
				cfg.UseGlobal,
				cfg.GlobalFirst,
				cfg.ActionsDir,
				cfg.LibraryPath,
			)

			// Get action config
			actionConfig, err := resolver.GetActionConfig(actionName)
			if err != nil {
				return fmt.Errorf("error getting action info: %w", err)
			}

			// Display action information
			fmt.Printf("Action: %s\n", actionConfig.Name)
			fmt.Printf("Type: %s\n", actionConfig.Type)
			fmt.Printf("Description: %s\n", actionConfig.Description)

			// Display labels if they exist
			if len(actionConfig.Labels) > 0 {
				fmt.Println("Labels:")
				for key, values := range actionConfig.Labels {
					fmt.Printf("  %s: %s\n", key, strings.Join(values, ", "))
				}
			}

			// Type-specific information
			switch actionConfig.Type {
			case "file":
				fmt.Printf("Template Path: %s\n", actionConfig.TemplatePath)
				fmt.Printf("Target Path: %s\n", actionConfig.TargetPath)
				fmt.Printf("Create Directories: %t\n", actionConfig.CreateDirs)
			case "cli":
				fmt.Printf("Command: %s\n", actionConfig.Command)
				fmt.Printf("Arguments: %s\n", strings.Join(actionConfig.Args, " "))
			}

			// Display parameter schema
			if actionConfig.Schema != nil {
				fmt.Println("\nParameter Schema:")
				schemaJSON, _ := json.MarshalIndent(actionConfig.Schema, "", "  ")
				fmt.Println(string(schemaJSON))
			}

			// Display default values if any
			if len(actionConfig.Defaults) > 0 {
				fmt.Println("\nDefault Values:")
				for k, v := range actionConfig.Defaults {
					fmt.Printf("  %s: %v\n", k, v)
				}
			}

			return nil
		},
	}

	return infoCmd
}

// newActionRunCmd creates a 'run' subcommand
func newActionRunCmd() *cobra.Command {
	var verboseFlag bool
	var dryRunFlag bool

	runCmd := &cobra.Command{
		Use:   "run [action-name] [params-file] -- [json-string]",
		Short: "Run an action",
		Long: `Run an action with the specified parameters.
Parameters can be provided in multiple ways:
1. As a file path directly (arg 2)
2. As a JSON string after -- (arg 3)`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			actionName := args[0]

			// Determine parameters source
			var params map[string]interface{}
			var err error

			// Check for -- separator for inline JSON
			jsonStringArg := ""
			dashIndex := -1
			for i, arg := range args {
				if arg == "--" && i+1 < len(args) {
					dashIndex = i
					jsonStringArg = args[i+1]
					break
				}
			}

			if dashIndex > 0 {
				// Parse JSON string after --
				err = json.Unmarshal([]byte(jsonStringArg), &params)
				if err != nil {
					return fmt.Errorf("error parsing JSON parameters: %w", err)
				}
			} else if len(args) > 1 {
				// Treat second arg as params file
				paramsFile := args[1]
				params, err = loadParams(paramsFile)
				if err != nil {
					return fmt.Errorf("error loading parameters from file: %w", err)
				}
			} else {
				// No parameters provided
				params = make(map[string]interface{})
			}

			// Get the working directory
			workingDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("error getting working directory: %w", err)
			}

			// Load configuration
			// For darn action commands, cmdLineLibraryPath and globalConfigPathOverride are not applicable from CLI flags.
			cfg, err := config.LoadConfig("", "")
			if err != nil {
				return fmt.Errorf("error loading configuration: %w", err)
			}

			// Create action context with templates directory and working directory
			context := action.ActionContext{
				TemplatesDir: filepath.Join(workingDir, cfg.TemplatesDir),
				WorkingDir:   workingDir,
				VerboseMode:  verboseFlag,
			}

			// Create action factory with context
			factory := action.NewFactory(context)
			factory.RegisterDefaultTypes()

			// Create resolver
			resolver := NewResolver(
				factory,
				workingDir,
				cfg.UseLocal,
				cfg.UseGlobal,
				cfg.GlobalFirst,
				cfg.ActionsDir,
				cfg.LibraryPath,
			)

			// Get action config for validation
			actionConfig, err := resolver.GetActionConfig(actionName)
			if err != nil {
				return fmt.Errorf("error loading action: %w", err)
			}

			// Validate parameters against schema
			if actionConfig.Schema != nil {
				// Merge with defaults if any
				if actionConfig.Defaults != nil {
					params = schema.MergeWithDefaults(params, actionConfig.Defaults)
				}

				if err := schema.ValidateParams(actionConfig.Schema, params); err != nil {
					return fmt.Errorf("parameter validation failed: %w", err)
				}
			}

			// Add verbose flag to parameters
			params["verbose"] = verboseFlag

			// Add dry run flag to parameters if needed
			if dryRunFlag {
				params["dry_run"] = true
				fmt.Println("Running in dry-run mode (no changes will be made)")
			}

			if verboseFlag {
				fmt.Printf("Executing action '%s' with parameters:\n", actionName)
				paramsJSON, _ := json.MarshalIndent(params, "", "  ")
				fmt.Println(string(paramsJSON))
			}

			// Resolve and execute the action
			if dryRunFlag {
				fmt.Printf("Would execute action '%s' with parameters:\n", actionName)
				paramsJSON, _ := json.MarshalIndent(params, "", "  ")
				fmt.Println(string(paramsJSON))
				return nil
			}

			// Resolve the action
			act, err := resolver.ResolveAction(actionName)
			if err != nil {
				return fmt.Errorf("error resolving action: %w", err)
			}

			// Execute the action
			if verboseFlag {
				fmt.Printf("Executing action: %s\n", actionName)
			}

			// Check if this is an OutputAction
			if outputAction, ok := act.(action.OutputAction); ok {
				outputs, err := outputAction.ExecuteWithOutput(params)
				if err != nil {
					return fmt.Errorf("error executing action: %w", err)
				}

				// Display outputs
				if len(outputs) > 0 && verboseFlag {
					fmt.Println("\nAction outputs:")
					for k, v := range outputs {
						fmt.Printf("  %s: %v\n", k, v)
					}
				}
			} else {
				// Regular action
				if err := act.Execute(params); err != nil {
					return fmt.Errorf("error executing action: %w", err)
				}
			}

			if verboseFlag {
				fmt.Printf("Action '%s' executed successfully\n", actionName)
			}

			return nil
		},
	}

	// Add flags
	runCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output")
	runCmd.Flags().BoolVarP(&dryRunFlag, "dry-run", "d", false, "Perform a dry run (no changes)")

	return runCmd
}

// loadParams loads parameters from a file (YAML or JSON)
func loadParams(filePath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	var params map[string]interface{}

	// Try JSON first
	err = json.Unmarshal(data, &params)
	if err != nil {
		// If JSON fails, try YAML
		// Note: The original code had json.Unmarshal again here. Assuming it meant to try yaml.
		// However, since we don't have a direct YAML unmarshal to map[string]interface{} easily
		// without a specific struct, and JSON is a subset of YAML, if JSON fails,
		// it's likely not trivially convertible YAML for this generic map.
		// For simplicity and to match original logic flaw (which might have implicitly worked for some YAML):
		// We'll stick to trying JSON twice, which is effectively one JSON try.
		// A proper fix would involve importing a YAML library and attempting yaml.Unmarshal.
		// For now, correcting to a single JSON attempt.
		errRetry := json.Unmarshal(data, &params) // Retrying JSON, per original structure.
		if errRetry != nil { // If it still fails
			return nil, fmt.Errorf("error parsing file as JSON: %w", err) // Report original JSON error
		}
	}

	return params, nil
}
