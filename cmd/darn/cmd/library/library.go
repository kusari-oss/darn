// SPDX-License-Identifier: Apache-2.0

package library

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kusari-oss/darn/internal/core/config"
	"github.com/kusari-oss/darn/internal/core/library"
	"github.com/kusari-oss/darn/internal/defaults"
	"github.com/kusari-oss/darn/internal/version"
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

	return libraryCmd
}

// newInitCommand creates the init subcommand
func newInitCommand() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize a darn library",
		Long:  `Initialize a darn library with default templates, actions, and configs.`,
		Args:  cobra.MaximumNArgs(1),
		Run:   runInitCommand,
	}

	// Configure flags
	initCmd.Flags().StringP("templates-dir", "t", "templates", "Directory for template files")
	initCmd.Flags().StringP("actions-dir", "a", "actions", "Directory for action files")
	initCmd.Flags().StringP("configs-dir", "c", "configs", "Directory for configuration files")
	initCmd.Flags().StringP("mappings-dir", "m", "mappings", "Directory for mapping files")
	initCmd.Flags().BoolP("local-only", "l", false, "Use only embedded defaults, don't attempt to fetch latest from remote")
	initCmd.Flags().StringP("remote-url", "r", "https://raw.githubusercontent.com/kusari-oss/darn-defaults/main", "URL for remote defaults repository")
	initCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	return initCmd
}

// runInitCommand is the function executed for the init command
func runInitCommand(cmd *cobra.Command, args []string) {
	// Determine the directory to initialize
	projectDir := "."
	if len(args) > 0 {
		projectDir = args[0]
	}

	// Get flag values
	templatesDir, _ := cmd.Flags().GetString("templates-dir")
	actionsDir, _ := cmd.Flags().GetString("actions-dir")
	configsDir, _ := cmd.Flags().GetString("configs-dir")
	mappingsDir, _ := cmd.Flags().GetString("mappings-dir")
	localOnly, _ := cmd.Flags().GetBool("local-only")
	remoteURL, _ := cmd.Flags().GetString("remote-url")
	_, _ = cmd.Flags().GetBool("verbose")

	// Create the project directory if it doesn't exist
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		fmt.Printf("Error creating project directory: %v\n", err)
		os.Exit(1)
	}

	// Create directory paths
	templatesDirPath := filepath.Join(projectDir, templatesDir)
	actionsDirPath := filepath.Join(projectDir, actionsDir)
	configsDirPath := filepath.Join(projectDir, configsDir)
	mappingsDirPath := filepath.Join(projectDir, mappingsDir)

	// Create configuration
	cfg := &config.Config{
		TemplatesDir: templatesDir,
		ActionsDir:   actionsDir,
		ConfigsDir:   configsDir,
		MappingsDir:  mappingsDir,
		UseLocal:     true,
		UseGlobal:    !localOnly,
		GlobalFirst:  false,
		LibraryPath:  config.ExpandPath(config.DefaultGlobalLibrary),
	}

	// Save configuration
	if err := config.SaveConfig(cfg, projectDir); err != nil {
		fmt.Printf("Error saving configuration: %v\n", err)
		os.Exit(1)
	}

	// Create defaults manager
	defaultsConfig := defaults.DefaultsConfig{
		DefaultsURL: remoteURL,
		UseRemote:   !localOnly,
		Timeout:     10,
	}
	manager := defaults.NewManager(defaultsConfig)

	// Copy defaults
	usedRemote, err := manager.CopyDefaults(templatesDirPath, actionsDirPath, configsDirPath, mappingsDirPath, !localOnly)
	if err != nil {
		fmt.Printf("Error copying defaults: %v\n", err)
		os.Exit(1)
	}

	// Create and save state
	state := &config.State{
		ProjectDir:    projectDir,
		LibraryInUse:  cfg.LibraryPath,
		LastUpdated:   time.Now().Format(time.RFC3339),
		InitializedAt: time.Now().Format(time.RFC3339),
		Version:       version.Version,
	}

	if err := config.SaveState(state, projectDir); err != nil {
		fmt.Printf("Error saving state: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nInitialization complete!")
	fmt.Printf("Templates directory: %s\n", templatesDirPath)
	fmt.Printf("Actions directory: %s\n", actionsDirPath)
	fmt.Printf("Configs directory: %s\n", configsDirPath)
	fmt.Printf("Mappings directory: %s\n", mappingsDirPath)
	fmt.Printf("Used remote defaults: %v\n", usedRemote)
}

// newUpdateCommand creates the update subcommand
func newUpdateCommand() *cobra.Command {
	updateCmd := &cobra.Command{
		Use:   "update [source-directory]",
		Short: "Update the darn library",
		Long:  `Update the darn library with new or modified templates, actions, and configs.`,
		Args:  cobra.MaximumNArgs(1),
		Run:   runUpdateCommand,
	}

	// Configure flags
	updateCmd.Flags().StringP("library-path", "l", config.DefaultGlobalLibrary, "Path to the library to update")
	updateCmd.Flags().BoolP("force", "f", false, "Force update even if files are identical")
	updateCmd.Flags().BoolP("dry-run", "d", false, "Show what would be updated without making changes")
	updateCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	return updateCmd
}

// runUpdateCommand is the function executed for the update command
func runUpdateCommand(cmd *cobra.Command, args []string) {
	// Determine source directory
	sourceDir := "."
	if len(args) > 0 {
		sourceDir = args[0]
	}

	// Get flag values
	libraryPath, _ := cmd.Flags().GetString("library-path")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Expand library path if needed
	libraryPath = config.ExpandPath(libraryPath)

	// Create updater
	updater := library.NewUpdater(libraryPath, sourceDir, force, dryRun, verbose)

	// Update library
	if err := updater.UpdateLibrary(); err != nil {
		fmt.Printf("Error updating library: %v\n", err)
		os.Exit(1)
	}

	if !dryRun {
		fmt.Println("\nLibrary update complete!")
		fmt.Printf("Updated library: %s\n", libraryPath)
	} else {
		fmt.Println("\nDry run completed. No files were actually modified.")
	}
}

// NewDeprecatedInitCommand creates a backward compatible 'init' command
func NewDeprecatedInitCommand() *cobra.Command {
	initCmd := &cobra.Command{
		Use:        "init [directory]",
		Short:      "Initialize a darn project (DEPRECATED: use 'library init' instead)",
		Long:       `DEPRECATED: Please use 'darn library init' instead. This command will be removed in a future version.`,
		Args:       cobra.MaximumNArgs(1),
		Deprecated: "use 'library init' instead",
		Run: func(cmd *cobra.Command, args []string) {
			// Get all flags
			templatesDir, _ := cmd.Flags().GetString("templates-dir")
			actionsDir, _ := cmd.Flags().GetString("actions-dir")
			configsDir, _ := cmd.Flags().GetString("configs-dir")
			localOnly, _ := cmd.Flags().GetBool("local-only")
			remoteURL, _ := cmd.Flags().GetString("remote-url")
			verbose, _ := cmd.Flags().GetBool("verbose")

			// Create new args slice
			newArgs := []string{"library", "init"}
			if len(args) > 0 {
				newArgs = append(newArgs, args[0])
			}

			// Add all flags to the new command
			if cmd.Flags().Changed("templates-dir") {
				newArgs = append(newArgs, "--templates-dir="+templatesDir)
			}
			if cmd.Flags().Changed("actions-dir") {
				newArgs = append(newArgs, "--actions-dir="+actionsDir)
			}
			if cmd.Flags().Changed("configs-dir") {
				newArgs = append(newArgs, "--configs-dir="+configsDir)
			}
			if cmd.Flags().Changed("local-only") && localOnly {
				newArgs = append(newArgs, "--local-only")
			}
			if cmd.Flags().Changed("remote-url") {
				newArgs = append(newArgs, "--remote-url="+remoteURL)
			}
			if cmd.Flags().Changed("verbose") && verbose {
				newArgs = append(newArgs, "--verbose")
			}

			// Get the root command
			root := cmd.Root()

			// Execute the new command
			root.SetArgs(newArgs)
			root.Execute()
		},
	}

	// Add the same flags as the library init command
	initCmd.Flags().StringP("templates-dir", "t", "templates", "Directory for template files")
	initCmd.Flags().StringP("actions-dir", "a", "actions", "Directory for action files")
	initCmd.Flags().StringP("configs-dir", "c", "configs", "Directory for configuration files")
	initCmd.Flags().BoolP("local-only", "l", false, "Use only embedded defaults, don't attempt to fetch latest from remote")
	initCmd.Flags().StringP("remote-url", "r", "https://raw.githubusercontent.com/kusari-oss/darn-defaults/main", "URL for remote defaults repository")
	initCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	return initCmd
}
