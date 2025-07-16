// SPDX-License-Identifier: Apache-2.0

package library

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kusari-oss/darn/internal/core/config"
	"github.com/spf13/cobra"
)

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Diagnose library configuration issues",
	Long: `Diagnose library configuration and path resolution issues.

This command provides detailed information about:
- Library path resolution order and results
- Configuration sources and their values
- Directory existence and accessibility
- Common configuration problems

Use this command when you're experiencing issues with library paths,
missing actions, or environment-specific problems.`,
	RunE: runDiagnoseCommand,
}

var (
	jsonOutput bool
	verbose    bool
)

func init() {
	diagnoseCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	diagnoseCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
}

func runDiagnoseCommand(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig("", "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Enable verbose logging if requested
	if verbose {
		cfg.SetVerboseLibraryLogging(true)
	}

	// Get diagnostic information
	diagnostics := cfg.GetLibraryDiagnostics()

	// Try to resolve library path
	libraryInfo, libraryErr := cfg.GetLibraryInfo()
	if libraryInfo != nil {
		diagnostics["library_resolution"] = map[string]interface{}{
			"path":   libraryInfo.Path,
			"source": libraryInfo.Source,
			"valid":  libraryInfo.Valid,
			"exists": libraryInfo.Exists,
			"errors": libraryInfo.Errors,
			"subdirectories": map[string]string{
				"actions":   libraryInfo.ActionsDir,
				"templates": libraryInfo.TemplatesDir,
				"configs":   libraryInfo.ConfigsDir,
				"mappings":  libraryInfo.MappingsDir,
			},
		}
	}
	if libraryErr != nil {
		diagnostics["library_error"] = libraryErr.Error()
	}

	// Test command availability for common commands
	commonCommands := []string{"sh", "bash", "cmd", "echo", "mkdir", "cp", "mv"}
	commandTests := make(map[string]interface{})
	for _, cmd := range commonCommands {
		if err := cfg.ValidateCommand(cmd); err != nil {
			commandTests[cmd] = map[string]interface{}{
				"available": false,
				"error":     err.Error(),
			}
		} else {
			commandTests[cmd] = map[string]interface{}{
				"available": true,
			}
		}
	}
	diagnostics["commands"] = commandTests

	// Add current working directory
	if cwd, err := os.Getwd(); err == nil {
		diagnostics["current_directory"] = cwd
	}

	// Add environment variables
	diagnostics["environment"] = map[string]string{
		"HOME":      os.Getenv("HOME"),
		"DARN_HOME": os.Getenv("DARN_HOME"),
		"PATH":      os.Getenv("PATH"),
	}

	// Output results
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(diagnostics)
	}

	// Human-readable output
	fmt.Println("=== Darn Library Diagnostics ===")
	fmt.Println()

	// Library resolution status
	fmt.Println("Library Resolution:")
	if libraryInfo != nil {
		if libraryInfo.Valid {
			fmt.Printf("  ✓ Library found and valid: %s\n", libraryInfo.Path)
			fmt.Printf("    Source: %s\n", libraryInfo.Source)
		} else {
			fmt.Printf("  ✗ Library invalid: %s\n", libraryInfo.Path)
			fmt.Printf("    Source: %s\n", libraryInfo.Source)
			for _, err := range libraryInfo.Errors {
				fmt.Printf("    Error: %s\n", err)
			}
		}
	} else {
		fmt.Printf("  ✗ No library path resolved\n")
		if libraryErr != nil {
			fmt.Printf("    Error: %s\n", libraryErr.Error())
		}
	}
	fmt.Println()

	// Configuration sources
	fmt.Println("Configuration Sources:")
	if cmdLineLib := diagnostics["cmdline_library_path"]; cmdLineLib != nil && cmdLineLib != "" {
		fmt.Printf("  Command line: %s\n", cmdLineLib)
	} else {
		fmt.Printf("  Command line: (not set)\n")
	}
	
	if darnHome := diagnostics["darn_home"]; darnHome != nil && darnHome != "" {
		fmt.Printf("  DARN_HOME: %s\n", darnHome)
	} else {
		fmt.Printf("  DARN_HOME: (not set)\n")
	}
	
	if globalLib := diagnostics["global_library_path"]; globalLib != nil && globalLib != "" {
		fmt.Printf("  Global config: %s\n", globalLib)
	} else {
		fmt.Printf("  Global config: (not set)\n")
	}
	
	if userHome := diagnostics["user_home"]; userHome != nil {
		fmt.Printf("  User home: %s\n", userHome)
	}
	fmt.Println()

	// Command availability
	fmt.Println("Command Availability:")
	for cmd, result := range commandTests {
		resultMap := result.(map[string]interface{})
		if resultMap["available"].(bool) {
			fmt.Printf("  ✓ %s\n", cmd)
		} else {
			fmt.Printf("  ✗ %s - %s\n", cmd, resultMap["error"])
		}
	}
	fmt.Println()

	// Recommendations
	fmt.Println("Recommendations:")
	if libraryInfo == nil || !libraryInfo.Valid {
		fmt.Println("  1. Initialize a library with: darn library init")
		fmt.Println("  2. Or set a custom library path with: darn library set-global <path>")
	}
	
	// Check for missing common shell commands
	missingCommands := []string{}
	for cmd, result := range commandTests {
		resultMap := result.(map[string]interface{})
		if !resultMap["available"].(bool) {
			missingCommands = append(missingCommands, cmd)
		}
	}
	if len(missingCommands) > 0 {
		fmt.Printf("  3. Install missing commands: %v\n", missingCommands)
	}

	return nil
}