// SPDX-License-Identifier: Apache-2.0

package library

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kusari-oss/darn/internal/core/config"
	"github.com/kusari-oss/darn/internal/core/library"
	"github.com/kusari-oss/darn/internal/defaults"
	"github.com/spf13/cobra"
)

// NewLibraryCommand creates the library command
func NewLibraryCommand() *cobra.Command {
	libraryCmd := &cobra.Command{
		Use:   "library",
		Short: "Manage the darn library of templates and actions",
		Long:  `Manage the darn library of templates and actions. Initialize the library or update it with new files.`,
	}

	// Add init subcommand
	initCmd := newInitCommand()
	libraryCmd.AddCommand(initCmd)

	// Add update subcommand
	updateCmd := newUpdateCommand()
	libraryCmd.AddCommand(updateCmd)

	// Add set-global subcommand
	setGlobalCmd := newSetGlobalCommand()
	libraryCmd.AddCommand(setGlobalCmd)

	return libraryCmd
}

// newSetGlobalCommand creates the set-global subcommand
func newSetGlobalCommand() *cobra.Command {
	setGlobalCmd := &cobra.Command{
		Use:   "set-global <path>",
		Short: "Sets the global darn library path in the user's global configuration file.",
		Long: `Sets the global darn library path in the user's global configuration file.

This command updates the global configuration file (typically located at '~/.darn/config.yaml')
to use the specified <path> as the darn library for operations outside of a project.
The <path> argument will be expanded; for example, '~' will be resolved to your home directory.

If the global configuration file or the '~/.darn' directory does not exist, they will be created.

Specifically, this command:
  - Sets the 'library_path' in '~/.darn/config.yaml' to the provided <path>.
  - Sets 'use_global: true' and 'use_local: false' in the global config to ensure this library is used by default.

This global library path is used when 'darn' is run outside a darn project (which would have its own
'.darn/config.yaml'). If a project-specific configuration exists, it will take precedence over this
global setting for operations within that project.

Examples:
  darn library set-global ~/my-darn-templates
  darn library set-global /usr/local/share/darn-library
  darn library set-global C:\Users\YourUser\Documents\DarnLibrary

After running, darn will use this path to look for templates and actions when no project-specific
library is configured or when the '--library' flag is not used.`,
		Args: cobra.ExactArgs(1),
		Run:  runSetGlobalCommand,
	}
	return setGlobalCmd
}

// runSetGlobalCommand is the function executed for the set-global command
func runSetGlobalCommand(cmd *cobra.Command, args []string) {
	newLibraryPath := args[0]

	// Expand the provided path
	// config.ExpandPathWithTilde does not return an error.
	expandedLibraryPath := config.ExpandPathWithTilde(newLibraryPath)

	// Define the global config file path
	globalConfigPath, err := config.GlobalConfigFilePath()
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error determining global config path: %v\n", err)
		os.Exit(1)
	}

	// Attempt to load the existing global config file
	cfg, err := config.LoadConfigFile(globalConfigPath)
	if err != nil {
		// If not found or other error, create a new default config
		if os.IsNotExist(err) {
			fmt.Fprintf(cmd.OutOrStdout(), "Global config file not found at %s. Creating a new one.\n", globalConfigPath)
			cfg = config.NewDefaultConfig()
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error loading global config file '%s': %v\n", globalConfigPath, err)
			// Still try to create a new one if loading failed for other reasons,
			// as the user intends to set a new global library.
			cfg = config.NewDefaultConfig()
		}
	}

	// Update the LibraryPath field
	cfg.LibraryPath = expandedLibraryPath
	cfg.UseGlobal = true
	cfg.UseLocal = false // When setting a global library, local usage should typically be disabled in the global config itself.

	// Save the modified config back to the global config file path
	if err := config.SaveGlobalConfig(cfg); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error saving global configuration to '%s': %v\n", globalConfigPath, err)
		os.Exit(1)
	}

	// Check if the new library path exists and inform the user
	pathExists := "exists"
	if _, err := os.Stat(expandedLibraryPath); os.IsNotExist(err) {
		pathExists = "does not exist (it may need to be initialized or created)"
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Global darn library path successfully set to: %s\n", expandedLibraryPath)
	fmt.Fprintf(cmd.OutOrStdout(), "The specified path %s.\n", pathExists)
	fmt.Fprintf(cmd.OutOrStdout(), "Configuration saved to: %s\n", globalConfigPath)
}

// newInitCommand creates the init subcommand
func newInitCommand() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init [target-library-path]",
		Short: "Initializes a darn library with standard structure and content.",
		Long: `Initializes a darn library at the specified [target-library-path]
(or at '~/.darn/library' if no path is provided) with the standard
darn library structure (actions, templates, configs, mappings) and populates
it with default content.

This command creates the necessary directories and copies in the default
set of actions, templates, etc. It does NOT set the initialized path as the
active library for darn operations; for that, use 'darn library set-global'.

Examples:
  darn library init                               # Initializes at ~/.darn/library
  darn library init ~/my-custom-darn-lib          # Initializes at ~/my-custom-darn-lib
  darn library init /opt/shared/darn-library      # Initializes at /opt/shared/darn-library
  darn library init ./my-local-lib                # Initializes at ./my-local-lib (relative to current dir)`,
		Args: cobra.MaximumNArgs(1), // Allows zero or one argument
		Run:  runInitCommand,
	}

	// Configure flags
	initCmd.Flags().StringP("templates-dir", "t", "templates", "Name for the templates directory within the library (default: \"templates\")")
	initCmd.Flags().StringP("actions-dir", "a", "actions", "Name for the actions directory within the library (default: \"actions\")")
	initCmd.Flags().StringP("configs-dir", "c", "configs", "Name for the configs directory within the library (default: \"configs\")")
	initCmd.Flags().StringP("mappings-dir", "m", "mappings", "Name for the mappings directory within the library (default: \"mappings\")")
	// Removed --global-config-path flag
	initCmd.Flags().BoolP("local-only", "l", false, "Use only embedded defaults when populating the library; do not attempt to fetch latest from remote source.")
	initCmd.Flags().StringP("remote-url", "r", "https://raw.githubusercontent.com/kusari-oss/darn-defaults/main", "URL for remote defaults repository")
	initCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	return initCmd
}

// runInitCommand is the function executed for the init command
func runInitCommand(cmd *cobra.Command, args []string) {
	var targetLibraryPath string
	if len(args) > 0 {
		targetLibraryPath = config.ExpandPathWithTilde(args[0])
	} else {
		targetLibraryPath = config.ExpandPathWithTilde(config.DefaultGlobalLibrary)
	}

	// Ensure targetLibraryPath is absolute
	absTargetLibraryPath, err := filepath.Abs(targetLibraryPath)
	if err != nil {
		fmt.Printf("Error resolving absolute path for target library '%s': %v\n", targetLibraryPath, err)
		os.Exit(1)
	}

	// Get flag values for subdirectory names
	templatesDirName, _ := cmd.Flags().GetString("templates-dir")
	actionsDirName, _ := cmd.Flags().GetString("actions-dir")
	configsDirName, _ := cmd.Flags().GetString("configs-dir")
	mappingsDirName, _ := cmd.Flags().GetString("mappings-dir")
	localOnly, _ := cmd.Flags().GetBool("local-only")
	remoteURL, _ := cmd.Flags().GetString("remote-url")
	verbose, _ := cmd.Flags().GetBool("verbose") // Retain verbose for defaults manager

	if verbose {
		fmt.Printf("Initializing library structure at: %s\n", absTargetLibraryPath)
	}

	// Create directory paths for library subdirectories (these are absolute paths)
	templatesDirPath := filepath.Join(absTargetLibraryPath, templatesDirName)
	actionsDirPath := filepath.Join(absTargetLibraryPath, actionsDirName)
	configsDirPath := filepath.Join(absTargetLibraryPath, configsDirName)
	mappingsDirPath := filepath.Join(absTargetLibraryPath, mappingsDirName)

	// Create the library directory and its subdirectories
	// MkdirAll also creates parent directories if they don't exist (e.g. absTargetLibraryPath itself)
	for _, dirPath := range []string{absTargetLibraryPath, templatesDirPath, actionsDirPath, configsDirPath, mappingsDirPath} {
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", dirPath, err)
			os.Exit(1)
		}
	}

	// Create defaults manager
	defaultsConfig := defaults.DefaultsConfig{
		DefaultsURL: remoteURL,
		UseRemote:   !localOnly,
		Timeout:     10, // TODO: Make timeout configurable?
		//Verbose:     verbose,
	}
	manager := defaults.NewManager(defaultsConfig)

	// Copy defaults into the new library subdirectories
	usedRemote, err := manager.CopyDefaults(templatesDirPath, actionsDirPath, configsDirPath, mappingsDirPath, !localOnly)
	if err != nil {
		fmt.Printf("Error copying defaults to library at %s: %v\n", absTargetLibraryPath, err)
		os.Exit(1)
	}

	fmt.Printf("\nLibrary content initialized successfully at: %s\n", absTargetLibraryPath)
	fmt.Printf("  Templates directory: %s (subdirectory name: %s)\n", templatesDirPath, templatesDirName)
	fmt.Printf("  Actions directory: %s (subdirectory name: %s)\n", actionsDirPath, actionsDirName)
	fmt.Printf("  Configs directory: %s (subdirectory name: %s)\n", configsDirPath, configsDirName)
	fmt.Printf("  Mappings directory: %s (subdirectory name: %s)\n", mappingsDirPath, mappingsDirName)
	if defaultsConfig.UseRemote {
		fmt.Printf("Used remote defaults: %v\n", usedRemote)
	} else {
		fmt.Println("Used embedded defaults (local-only mode).")
	}
	fmt.Println("\nNote: This command does not set the initialized path as the active library.")
	fmt.Println("To use this library globally, run: darn library set-global", absTargetLibraryPath)
}

// newUpdateCommand creates the update subcommand
func newUpdateCommand() *cobra.Command {
	updateCmd := &cobra.Command{
		Use:   "update [source-directory]",
		Short: "Updates a darn library with content from a source directory.",
		Long: `Updates a darn library with new or modified templates, actions, and configs
from the specified [source-directory] (defaults to the current directory).

Behavior:
- If --library-path is provided, the library at that specific path is updated.
- If --library-path is NOT provided, the command attempts to update the
  currently active global library (as configured in '~/.darn/config.yaml').
- If no global library is configured in '~/.darn/config.yaml', it updates the
  default global library path ('~/.darn/library').

Examples:
  darn library update
    (Updates active global library, or ~/.darn/library if none set, using current directory as source)

  darn library update ./my-library-source-files
    (Updates active global library, or ~/.darn/library, using ./my-library-source-files as source)

  darn library update --library-path /specific/darn-lib ./my-library-source-files
    (Updates the library at /specific/darn-lib using ./my-library-source-files as source)`,
		Args: cobra.MaximumNArgs(1),
		Run:  runUpdateCommand,
	}

	// Configure flags
	// The default value for library-path is effectively handled by the logic in runUpdateCommand.
	// Setting it to "" here makes it clear that its absence triggers the global config lookup.
	updateCmd.Flags().StringP("library-path", "l", "", "Path to the library to update. If omitted, updates the active global library or default (~/.darn/library).")
	updateCmd.Flags().BoolP("force", "f", false, "Force update even if files are identical.")
	updateCmd.Flags().BoolP("dry-run", "d", false, "Show what would be updated without making changes.")
	updateCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output.")

	return updateCmd
}

// runUpdateCommand is the function executed for the update command
func runUpdateCommand(cmd *cobra.Command, args []string) {
	sourceDir := "."
	if len(args) > 0 {
		sourceDir = args[0]
	}

	libraryPathFlag, _ := cmd.Flags().GetString("library-path")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")

	var finalLibraryPathToUpdate string

	if cmd.Flags().Changed("library-path") {
		if verbose {
			fmt.Fprintf(cmd.OutOrStdout(), "Updating library specified by --library-path flag: %s\n", libraryPathFlag)
		}
		finalLibraryPathToUpdate = config.ExpandPathWithTilde(libraryPathFlag)
	} else {
		// --library-path not set, try to load global config
		if verbose {
			fmt.Fprintln(cmd.OutOrStdout(), "Attempting to update active global library (from ~/.darn/config.yaml or default ~/.darn/library)...")
		}
		// LoadConfig takes (cmdLineLibraryPath, globalConfigPathOverride)
		// For this specific purpose of finding the *active* global library, these should be empty.
		globalCfg, err := config.LoadConfig("", "")
		if err != nil {
			// This error might occur if the global config is malformed.
			// LoadConfig itself prints warnings for non-existent files.
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: Error loading global configuration: %v. Will attempt to update default global library path.\n", err)
			finalLibraryPathToUpdate = config.ExpandPathWithTilde(config.DefaultGlobalLibrary)
			if verbose || !cmd.Flags().Changed("library-path") { // Print message if not overridden by flag
				fmt.Fprintf(cmd.OutOrStdout(), "Updating default global library at: %s (no active global library configured or error loading config).\n", finalLibraryPathToUpdate)
			}
		} else {
			if globalCfg != nil && globalCfg.LibraryPath != "" {
				finalLibraryPathToUpdate = globalCfg.LibraryPath // This path should already be expanded by LoadConfig
				if verbose || !cmd.Flags().Changed("library-path") {
					fmt.Fprintf(cmd.OutOrStdout(), "Updating active global library configured at: %s\n", finalLibraryPathToUpdate)
				}
			} else {
				finalLibraryPathToUpdate = config.ExpandPathWithTilde(config.DefaultGlobalLibrary)
				if verbose || !cmd.Flags().Changed("library-path") {
					fmt.Fprintf(cmd.OutOrStdout(), "Updating default global library at: %s (no library path in global config or config not found).\n", finalLibraryPathToUpdate)
				}
			}
		}
	}

	// Ensure finalLibraryPathToUpdate is absolute
	absFinalLibraryPathToUpdate, err := filepath.Abs(finalLibraryPathToUpdate)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error resolving absolute path for library to update '%s': %v\n", finalLibraryPathToUpdate, err)
		os.Exit(1)
	}
	finalLibraryPathToUpdate = absFinalLibraryPathToUpdate

	// Create updater
	updater := library.NewUpdater(finalLibraryPathToUpdate, sourceDir, force, dryRun, verbose)

	// Update library
	if err := updater.UpdateLibrary(); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error updating library at %s: %v\n", finalLibraryPathToUpdate, err)
		os.Exit(1)
	}

	if !dryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "\nLibrary update complete!")
		fmt.Fprintf(cmd.OutOrStdout(), "Updated library at: %s\n", finalLibraryPathToUpdate)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "\nDry run completed. No files were actually modified.")
		fmt.Fprintf(cmd.OutOrStdout(), "Target library for update (if not dry run) would have been: %s\n", finalLibraryPathToUpdate)
	}
}

// NewDeprecatedInitCommand creates a backward compatible 'init' command
func NewDeprecatedInitCommand() *cobra.Command {
	deprecatedInitCmd := &cobra.Command{
		Use:   "init [target-library-path]", // Matches the new signature for clarity
		Short: "DEPRECATED: Initializes a darn library. Use 'darn library init' instead.",
		Long: `DEPRECATED: This command now initializes a darn library at the specified path (or default location)
but NO LONGER creates project-specific configuration (.darn/config.yaml, .darn/state.yaml).
It only populates the library directory structure.

For the updated behavior, please use 'darn library init [target-library-path]'.
To set the initialized library as your global default, use 'darn library set-global <path>'.
This command will be removed in a future version.`,
		Args:       cobra.MaximumNArgs(1),
		Deprecated: "Use 'darn library init [target-library-path]' for initializing library content and 'darn library set-global <path>' to set the global library. This command no longer creates project-specific configurations.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("WARNING: The 'darn init' command is deprecated and its behavior has changed.")
			fmt.Println("It now only initializes library content and does NOT create project-specific configuration.")
			fmt.Println("Please use 'darn library init' for this functionality.")
			fmt.Println("--------------------------------------------------")

			// Directly call the new runInitCommand logic.
			// We need to simulate the new initCmd for flag parsing.
			// The flags attached to *this* deprecatedInitCmd are the ones that will be parsed by Cobra.
			runInitCommand(cmd, args)
		},
	}

	// Keep flags that are still relevant to populating a library directory.
	// These flags are also present on the new `darn library init` command.
	deprecatedInitCmd.Flags().StringP("templates-dir", "t", "templates", "Name for the templates directory within the library (default: \"templates\")")
	deprecatedInitCmd.Flags().StringP("actions-dir", "a", "actions", "Name for the actions directory within the library (default: \"actions\")")
	deprecatedInitCmd.Flags().StringP("configs-dir", "c", "configs", "Name for the configs directory within the library (default: \"configs\")")
	deprecatedInitCmd.Flags().StringP("mappings-dir", "m", "mappings", "Name for the mappings directory within the library (default: \"mappings\")")
	deprecatedInitCmd.Flags().BoolP("local-only", "l", false, "Use only embedded defaults when populating the library; do not attempt to fetch latest from remote source.")
	deprecatedInitCmd.Flags().StringP("remote-url", "r", "https://raw.githubusercontent.com/kusari-oss/darn-defaults/main", "URL for remote defaults repository")
	deprecatedInitCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	// The --global-config-path flag is intentionally removed as it's no longer used by the core init logic.

	return deprecatedInitCmd
}
