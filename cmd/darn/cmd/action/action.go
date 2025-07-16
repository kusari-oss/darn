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
	"github.com/kusari-oss/darn/internal/core/format"
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
	actionCmd.AddCommand(newActionAddCmd())
	actionCmd.AddCommand(newActionValidateCmd())
	actionCmd.AddCommand(newActionSchemaCmd())
	actionCmd.AddCommand(newActionExampleCmd())

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

			// Display parameter information in a user-friendly way
			if actionConfig.Schema != nil {
				fmt.Println("\nParameters:")
				schemaMap := actionConfig.Schema
				if props, exists := schemaMap["properties"]; exists {
					if properties, ok := props.(map[string]interface{}); ok {
						// Get required fields
						requiredFields := make(map[string]bool)
						if required, exists := schemaMap["required"]; exists {
							if reqArray, ok := required.([]interface{}); ok {
								for _, req := range reqArray {
									if reqStr, ok := req.(string); ok {
										requiredFields[reqStr] = true
									}
								}
							}
						}

						// Display each parameter
						for paramName, paramDef := range properties {
							if def, ok := paramDef.(map[string]interface{}); ok {
								fmt.Printf("  %s", paramName)
								
								// Show if required
								if requiredFields[paramName] {
									fmt.Printf(" (required)")
								} else {
									fmt.Printf(" (optional)")
								}
								
								// Show type
								if pType, exists := def["type"]; exists {
									fmt.Printf(" [%v]", pType)
								}
								
								fmt.Println()
								
								// Show description
								if desc, exists := def["description"]; exists {
									fmt.Printf("    %v\n", desc)
								}
								
								// Show default value if any
								if defaultVal, hasDefault := actionConfig.Defaults[paramName]; hasDefault {
									fmt.Printf("    Default: %v\n", defaultVal)
								}
								
								// Show example value
								example := generateExampleValue(def, paramName)
								fmt.Printf("    Example: %v\n", example)
								
								fmt.Println()
							}
						}
					}
				}
				
				fmt.Printf("\n💡 Use 'darn action schema %s' to see full JSON schema\n", actionConfig.Name)
				fmt.Printf("💡 Use 'darn action example %s' to see working examples\n", actionConfig.Name)
				fmt.Printf("💡 Use 'darn action validate %s --parameters file.json' to validate parameters\n", actionConfig.Name)
			} else {
				fmt.Println("\nParameters: No parameter schema defined")
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

// loadParams loads parameters from a file (supports both YAML and JSON)
func loadParams(filePath string) (map[string]interface{}, error) {
	var params map[string]interface{}
	if err := format.ParseFile(filePath, &params); err != nil {
		return nil, fmt.Errorf("error parsing parameter file: %w", err)
	}
	return params, nil
}

// newActionAddCmd creates an 'add' subcommand for creating new actions
func newActionAddCmd() *cobra.Command {
	var actionType string
	var description string
	var command string
	var args []string
	var templatePath string
	var targetPath string
	var createDirs bool
	var interactiveFlag bool

	addCmd := &cobra.Command{
		Use:   "add [action-name]",
		Short: "Create a new action",
		Long: `Create a new action in the current library.
This command helps you create custom CLI or file actions with proper validation.

Examples:
  # Create a CLI action interactively
  darn action add my-security-check --interactive

  # Create a CLI action with flags
  darn action add git-check --type cli --command git --args status,--porcelain

  # Create a file action
  darn action add add-readme --type file --template readme.md --target README.md`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			actionName := args[0]

			// Load configuration to get library path
			cfg, err := config.LoadConfig("", "")
			if err != nil {
				return fmt.Errorf("error loading configuration: %w", err)
			}

			// Use interactive mode if requested
			if interactiveFlag {
				return createActionInteractive(actionName, cfg)
			}

			// Validate required flags
			if actionType == "" {
				return fmt.Errorf("action type is required (use --type cli or --type file)")
			}

			if actionType != "cli" && actionType != "file" {
				return fmt.Errorf("action type must be 'cli' or 'file'")
			}

			if actionType == "cli" && command == "" {
				return fmt.Errorf("command is required for CLI actions (use --command)")
			}

			if actionType == "file" && templatePath == "" {
				return fmt.Errorf("template path is required for file actions (use --template)")
			}

			if actionType == "file" && targetPath == "" {
				return fmt.Errorf("target path is required for file actions (use --target)")
			}

			// Create the action
			return createAction(actionName, actionType, description, command, args, templatePath, targetPath, createDirs, cfg)
		},
	}

	// Add flags
	addCmd.Flags().StringVarP(&actionType, "type", "t", "", "Action type (cli or file)")
	addCmd.Flags().StringVarP(&description, "description", "d", "", "Action description")
	addCmd.Flags().StringVarP(&command, "command", "c", "", "Command to execute (for CLI actions)")
	addCmd.Flags().StringSliceVarP(&args, "args", "a", []string{}, "Command arguments (for CLI actions)")
	addCmd.Flags().StringVar(&templatePath, "template", "", "Template file path (for file actions)")
	addCmd.Flags().StringVar(&targetPath, "target", "", "Target file path (for file actions)")
	addCmd.Flags().BoolVar(&createDirs, "create-dirs", true, "Create parent directories for target file")
	addCmd.Flags().BoolVarP(&interactiveFlag, "interactive", "i", false, "Use interactive mode")

	return addCmd
}

// createActionInteractive creates an action using interactive prompts
func createActionInteractive(actionName string, cfg *config.Config) error {
	fmt.Printf("Creating action '%s' interactively...\n\n", actionName)

	// Prompt for action type
	fmt.Print("Action type (cli/file): ")
	var actionType string
	fmt.Scanln(&actionType)

	if actionType != "cli" && actionType != "file" {
		return fmt.Errorf("action type must be 'cli' or 'file'")
	}

	// Prompt for description
	fmt.Print("Description: ")
	var description string
	fmt.Scanln(&description)

	var command string
	var args []string
	var templatePath string
	var targetPath string
	var createDirs bool = true

	if actionType == "cli" {
		// CLI-specific prompts
		fmt.Print("Command: ")
		fmt.Scanln(&command)

		fmt.Print("Arguments (space-separated, or press enter for none): ")
		var argsStr string
		fmt.Scanln(&argsStr)
		if argsStr != "" {
			args = strings.Fields(argsStr)
		}
	} else {
		// File-specific prompts
		fmt.Print("Template path: ")
		fmt.Scanln(&templatePath)

		fmt.Print("Target path: ")
		fmt.Scanln(&targetPath)

		fmt.Print("Create parent directories? (y/n) [y]: ")
		var createDirsStr string
		fmt.Scanln(&createDirsStr)
		createDirs = createDirsStr != "n" && createDirsStr != "no"
	}

	return createAction(actionName, actionType, description, command, args, templatePath, targetPath, createDirs, cfg)
}

// createAction creates the action file
func createAction(actionName, actionType, description, command string, args []string, templatePath, targetPath string, createDirs bool, cfg *config.Config) error {
	// Resolve library path
	libraryPath := cfg.LibraryPath
	if libraryPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("error getting user home directory: %w", err)
		}
		libraryPath = filepath.Join(homeDir, ".darn", "library")
	}

	actionsDir := filepath.Join(libraryPath, cfg.ActionsDir)
	if cfg.ActionsDir == "" {
		actionsDir = filepath.Join(libraryPath, "actions")
	}

	// Ensure actions directory exists
	if err := os.MkdirAll(actionsDir, 0755); err != nil {
		return fmt.Errorf("error creating actions directory: %w", err)
	}

	actionFilePath := filepath.Join(actionsDir, fmt.Sprintf("%s.yaml", actionName))

	// Check if action already exists
	if _, err := os.Stat(actionFilePath); err == nil {
		return fmt.Errorf("action '%s' already exists at %s", actionName, actionFilePath)
	}

	// Create action content
	var actionContent strings.Builder
	actionContent.WriteString(fmt.Sprintf("name: \"%s\"\n", actionName))
	if description != "" {
		actionContent.WriteString(fmt.Sprintf("description: \"%s\"\n", description))
	} else {
		actionContent.WriteString(fmt.Sprintf("description: \"Custom %s action\"\n", actionType))
	}
	actionContent.WriteString(fmt.Sprintf("type: \"%s\"\n", actionType))

	if actionType == "cli" {
		actionContent.WriteString(fmt.Sprintf("command: \"%s\"\n", command))
		if len(args) > 0 {
			actionContent.WriteString("args:\n")
			for _, arg := range args {
				actionContent.WriteString(fmt.Sprintf("  - \"%s\"\n", arg))
			}
		} else {
			// Add comment if no args provided
			actionContent.WriteString("# args: []\n")
		}
	} else if actionType == "file" {
		actionContent.WriteString(fmt.Sprintf("template_path: \"%s\"\n", templatePath))
		actionContent.WriteString(fmt.Sprintf("target_path: \"%s\"\n", targetPath))
		actionContent.WriteString(fmt.Sprintf("create_dirs: %t\n", createDirs))
	}

	// Add basic parameter schema
	actionContent.WriteString("\nparameters:\n")
	actionContent.WriteString("  # Add your parameters here\n")
	actionContent.WriteString("  # Example:\n")
	actionContent.WriteString("  # - name: \"example_param\"\n")
	actionContent.WriteString("  #   type: \"string\"\n")
	actionContent.WriteString("  #   required: true\n")
	actionContent.WriteString("  #   description: \"Example parameter\"\n")

	// Write action file
	if err := os.WriteFile(actionFilePath, []byte(actionContent.String()), 0644); err != nil {
		return fmt.Errorf("error writing action file: %w", err)
	}

	fmt.Printf("✅ Action '%s' created successfully at %s\n", actionName, actionFilePath)

	// If it's a file action, also suggest creating the template
	if actionType == "file" {
		templatesDir := filepath.Join(libraryPath, cfg.TemplatesDir)
		if cfg.TemplatesDir == "" {
			templatesDir = filepath.Join(libraryPath, "templates")
		}
		templateFilePath := filepath.Join(templatesDir, templatePath)

		if _, err := os.Stat(templateFilePath); os.IsNotExist(err) {
			fmt.Printf("\n💡 Don't forget to create the template file at: %s\n", templateFilePath)
			fmt.Printf("   You can create a basic template with:\n")
			fmt.Printf("   mkdir -p %s\n", filepath.Dir(templateFilePath))
			fmt.Printf("   echo '# Template content here' > %s\n", templateFilePath)
		}
	}

	fmt.Printf("\n📝 Edit the action file to add parameters and customize behavior\n")
	fmt.Printf("🔍 Use 'darn action info %s' to view the action details\n", actionName)

	return nil
}

// newActionValidateCmd creates a 'validate' subcommand for validating parameters
func newActionValidateCmd() *cobra.Command {
	var paramsFile string

	validateCmd := &cobra.Command{
		Use:   "validate [action-name]",
		Short: "Validate parameters for an action",
		Long: `Validate parameters against an action's schema before execution.
This helps catch parameter errors early without running the action.

Examples:
  # Validate parameters from file
  darn action validate add-security-md --parameters params.json
  
  # Validate with inline JSON
  darn action validate add-security-md --parameters '{"project_name": "test"}'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			actionName := args[0]

			if paramsFile == "" {
				return fmt.Errorf("parameters are required (use --parameters)")
			}

			// Load configuration
			cfg, err := config.LoadConfig("", "")
			if err != nil {
				return fmt.Errorf("error loading configuration: %w", err)
			}

			// Get working directory
			workingDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("error getting working directory: %w", err)
			}

			// Create resolver
			context := action.ActionContext{
				TemplatesDir: filepath.Join(workingDir, cfg.TemplatesDir),
				WorkingDir:   workingDir,
				VerboseMode:  false,
			}
			factory := action.NewFactory(context)
			factory.RegisterDefaultTypes()
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
				return fmt.Errorf("error loading action: %w", err)
			}

			// Load parameters
			var params map[string]interface{}
			if strings.HasPrefix(paramsFile, "{") {
				// Inline JSON
				err = json.Unmarshal([]byte(paramsFile), &params)
				if err != nil {
					return fmt.Errorf("error parsing JSON parameters: %w", err)
				}
			} else {
				// File path
				params, err = loadParams(paramsFile)
				if err != nil {
					return fmt.Errorf("error loading parameters from file: %w", err)
				}
			}

			// Validate parameters against schema
			if actionConfig.Schema != nil {
				// Merge with defaults if any
				if actionConfig.Defaults != nil {
					params = schema.MergeWithDefaults(params, actionConfig.Defaults)
				}

				if err := schema.ValidateParams(actionConfig.Schema, params); err != nil {
					fmt.Printf("❌ Parameter validation failed for action '%s':\n", actionName)
					fmt.Printf("   %v\n", err)
					return err
				}

				fmt.Printf("✅ Parameters are valid for action '%s'\n", actionName)
				fmt.Printf("📊 Validated %d parameter(s)\n", len(params))

				// Show final parameters that would be used
				fmt.Println("\nFinal parameters (including defaults):")
				paramsJSON, _ := json.MarshalIndent(params, "", "  ")
				fmt.Println(string(paramsJSON))
			} else {
				fmt.Printf("⚠️  Action '%s' has no parameter schema - validation skipped\n", actionName)
			}

			return nil
		},
	}

	validateCmd.Flags().StringVarP(&paramsFile, "parameters", "p", "", "Parameters file (JSON/YAML) or inline JSON string")

	return validateCmd
}

// newActionSchemaCmd creates a 'schema' subcommand for outputting JSON schema
func newActionSchemaCmd() *cobra.Command {
	var outputFormat string

	schemaCmd := &cobra.Command{
		Use:   "schema [action-name]",
		Short: "Output JSON schema for action parameters",
		Long: `Output the JSON schema that defines valid parameters for an action.
This is useful for understanding required parameters, types, and constraints.

Examples:
  # Output JSON schema
  darn action schema add-security-md
  
  # Output as YAML
  darn action schema add-security-md --format yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			actionName := args[0]

			// Load configuration
			cfg, err := config.LoadConfig("", "")
			if err != nil {
				return fmt.Errorf("error loading configuration: %w", err)
			}

			// Get working directory
			workingDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("error getting working directory: %w", err)
			}

			// Create resolver
			context := action.ActionContext{
				TemplatesDir: filepath.Join(workingDir, cfg.TemplatesDir),
				WorkingDir:   workingDir,
				VerboseMode:  false,
			}
			factory := action.NewFactory(context)
			factory.RegisterDefaultTypes()
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
				return fmt.Errorf("error loading action: %w", err)
			}

			if actionConfig.Schema == nil {
				fmt.Printf("Action '%s' has no parameter schema defined\n", actionName)
				return nil
			}

			// Output schema in requested format
			switch outputFormat {
			case "yaml", "yml":
				// Convert to YAML format (simplified representation)
				fmt.Printf("# Parameter schema for action: %s\n", actionName)
				fmt.Printf("# Description: %s\n\n", actionConfig.Description)
				
				schemaMap := actionConfig.Schema
				if props, exists := schemaMap["properties"]; exists {
					if properties, ok := props.(map[string]interface{}); ok {
						fmt.Println("parameters:")
						for paramName, paramDef := range properties {
							if def, ok := paramDef.(map[string]interface{}); ok {
								fmt.Printf("  %s:\n", paramName)
								if pType, exists := def["type"]; exists {
									fmt.Printf("    type: %v\n", pType)
								}
								if desc, exists := def["description"]; exists {
									fmt.Printf("    description: \"%v\"\n", desc)
								}
								if required, exists := schemaMap["required"]; exists {
									if reqArray, ok := required.([]interface{}); ok {
										isRequired := false
										for _, req := range reqArray {
											if req == paramName {
												isRequired = true
												break
											}
										}
										fmt.Printf("    required: %t\n", isRequired)
									}
								}
							}
						}
					}
				}
			default:
				// JSON format
				schemaJSON, err := json.MarshalIndent(actionConfig.Schema, "", "  ")
				if err != nil {
					return fmt.Errorf("error marshaling schema: %w", err)
				}
				fmt.Println(string(schemaJSON))
			}

			return nil
		},
	}

	schemaCmd.Flags().StringVarP(&outputFormat, "format", "f", "json", "Output format (json or yaml)")

	return schemaCmd
}

// newActionExampleCmd creates an 'example' subcommand for showing working examples
func newActionExampleCmd() *cobra.Command {
	var outputFormat string

	exampleCmd := &cobra.Command{
		Use:   "example [action-name]",
		Short: "Show working parameter examples for an action",
		Long: `Show working parameter examples that demonstrate how to use an action.
Examples include both minimal required parameters and full examples with optional parameters.

Examples:
  # Show parameter examples
  darn action example add-security-md
  
  # Output as YAML
  darn action example add-security-md --format yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			actionName := args[0]

			// Load configuration
			cfg, err := config.LoadConfig("", "")
			if err != nil {
				return fmt.Errorf("error loading configuration: %w", err)
			}

			// Get working directory
			workingDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("error getting working directory: %w", err)
			}

			// Create resolver
			context := action.ActionContext{
				TemplatesDir: filepath.Join(workingDir, cfg.TemplatesDir),
				WorkingDir:   workingDir,
				VerboseMode:  false,
			}
			factory := action.NewFactory(context)
			factory.RegisterDefaultTypes()
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
				return fmt.Errorf("error loading action: %w", err)
			}

			fmt.Printf("# Parameter Examples for Action: %s\n", actionName)
			fmt.Printf("# Description: %s\n", actionConfig.Description)
			fmt.Printf("# Type: %s\n\n", actionConfig.Type)

			// Generate examples from schema
			examples := generateParameterExamples(*actionConfig)
			
			if len(examples) == 0 {
				fmt.Println("No parameter examples available for this action.")
				if actionConfig.Schema == nil {
					fmt.Println("Reason: Action has no parameter schema defined.")
				}
				return nil
			}

			// Output examples
			for i, example := range examples {
				fmt.Printf("## Example %d: %s\n\n", i+1, example.Description)
				
				switch outputFormat {
				case "yaml", "yml":
					fmt.Println("```yaml")
					yamlData, _ := json.Marshal(example.Parameters)
					fmt.Println(string(yamlData)) // Simple JSON representation
					fmt.Println("```")
				default:
					fmt.Println("```json")
					exampleJSON, _ := json.MarshalIndent(example.Parameters, "", "  ")
					fmt.Println(string(exampleJSON))
					fmt.Println("```")
				}
				
				fmt.Printf("\n**Usage:**\n")
				fmt.Printf("```bash\n")
				if outputFormat == "yaml" {
					fmt.Printf("darn action run %s params.yaml\n", actionName)
				} else {
					fmt.Printf("darn action run %s params.json\n", actionName)
				}
				fmt.Printf("```\n\n")
			}

			return nil
		},
	}

	exampleCmd.Flags().StringVarP(&outputFormat, "format", "f", "json", "Output format (json or yaml)")

	return exampleCmd
}

// ParameterExample represents a working example of parameters
type ParameterExample struct {
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// generateParameterExamples creates working examples from action schema
func generateParameterExamples(actionConfig action.Config) []ParameterExample {
	var examples []ParameterExample

	if actionConfig.Schema == nil {
		return examples
	}

	schemaMap := actionConfig.Schema
	if schemaMap == nil {
		return examples
	}

	props, exists := schemaMap["properties"]
	if !exists {
		return examples
	}

	properties, ok := props.(map[string]interface{})
	if !ok {
		return examples
	}

	// Get required fields
	requiredFields := make(map[string]bool)
	if required, exists := schemaMap["required"]; exists {
		if reqArray, ok := required.([]interface{}); ok {
			for _, req := range reqArray {
				if reqStr, ok := req.(string); ok {
					requiredFields[reqStr] = true
				}
			}
		}
	}

	// Generate minimal example (only required fields)
	minimalParams := make(map[string]interface{})
	for paramName, paramDef := range properties {
		if requiredFields[paramName] {
			if def, ok := paramDef.(map[string]interface{}); ok {
				example := generateExampleValue(def, paramName)
				minimalParams[paramName] = example
			}
		}
	}

	if len(minimalParams) > 0 {
		examples = append(examples, ParameterExample{
			Description: "Minimal (required parameters only)",
			Parameters:  minimalParams,
		})
	}

	// Generate full example (all parameters)
	fullParams := make(map[string]interface{})
	for paramName, paramDef := range properties {
		if def, ok := paramDef.(map[string]interface{}); ok {
			// Use default if available, otherwise generate example
			if defaultVal, hasDefault := actionConfig.Defaults[paramName]; hasDefault {
				fullParams[paramName] = defaultVal
			} else {
				example := generateExampleValue(def, paramName)
				fullParams[paramName] = example
			}
		}
	}

	if len(fullParams) > len(minimalParams) {
		examples = append(examples, ParameterExample{
			Description: "Full (all parameters with examples)",
			Parameters:  fullParams,
		})
	}

	return examples
}

// generateExampleValue creates example values based on parameter schema
func generateExampleValue(paramDef map[string]interface{}, paramName string) interface{} {
	paramType, hasType := paramDef["type"]
	if !hasType {
		return "example_value"
	}

	switch paramType {
	case "string":
		// Generate context-aware examples
		lowerName := strings.ToLower(paramName)
		if strings.Contains(lowerName, "email") {
			return "security@example.com"
		} else if strings.Contains(lowerName, "name") {
			return "my-project"
		} else if strings.Contains(lowerName, "url") {
			return "https://example.com"
		} else if strings.Contains(lowerName, "path") {
			return "/path/to/file"
		} else if strings.Contains(lowerName, "dir") {
			return "./directory"
		}
		return "example_string"
	case "number", "integer":
		return 42
	case "boolean":
		return true
	case "array":
		return []string{"item1", "item2"}
	case "object":
		return map[string]interface{}{"key": "value"}
	default:
		return "example_value"
	}
}
