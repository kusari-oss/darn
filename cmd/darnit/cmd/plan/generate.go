// SPDX-License-Identifier: Apache-2.0

package plan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kusari-oss/darn/internal/core/config"
	"github.com/kusari-oss/darn/internal/core/format"
	"github.com/kusari-oss/darn/internal/darnit"
	. "github.com/kusari-oss/darn/internal/darnit/plan"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func getGenerateCmd() *cobra.Command {
	generateCmd := &cobra.Command{
		Use:   "generate [report-file]",
		Short: "Generate a remediation plan from a report",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			reportFile := args[0]
			outputFile, _ := cmd.Flags().GetString("output")
			mappingFile, _ := cmd.Flags().GetString("mappings")
			mappingsDir, _ := cmd.Flags().GetString("mappings-dir") // New flag
			paramsFile, _ := cmd.Flags().GetString("params")
			paramsJSON, _ := cmd.Flags().GetString("params-json")
			repoPath, _ := cmd.Flags().GetString("repo")
			nonInteractive, _ := cmd.Flags().GetBool("non-interactive")
			verbose, _ := cmd.Flags().GetBool("verbose")

			// Get working directory
			_, err := os.Getwd()
			if err != nil {
				fmt.Printf("Error getting working directory: %v\n", err)
				os.Exit(1)
			}

			// Load configuration to get library path
			// For darnit plan generate, cmdLineLibraryPath is implicitly handled by LoadConfig if rootCmd's PersistentPreRunE
			// has already loaded a config influenced by the --library-path flag.
			// globalConfigPathOverride is also not directly applicable from this subcommand's flags.
			// The projectDir argument has been removed from LoadConfig.
			cfg, err := config.LoadConfig("", "")
			if err != nil {
				fmt.Printf("Error loading configuration: %v\n", err)
				os.Exit(1)
			}

			// Set the default mappings directory to the global library mappings path
			defaultMappingsDir := filepath.Join(cfg.LibraryPath, "mappings")

			// Override with flag value if specified
			actualMappingsDir := defaultMappingsDir
			if mappingsDir != "" {
				actualMappingsDir = mappingsDir
			}

			// Parse the report
			if verbose {
				fmt.Printf("Parsing report file: %s\n", reportFile)
			}
			report, err := darnit.ParseReportFile(reportFile)
			if err != nil {
				fmt.Printf("Error parsing report: %v\n", err)
				os.Exit(1)
			}

			// Load additional parameters
			extraParams, err := loadParameters(paramsFile, paramsJSON)
			if err != nil {
				fmt.Printf("Error loading parameters: %v\n", err)
				os.Exit(1)
			}

			// Set up options
			options := darnit.GenerateOptions{
				RepoPath:       repoPath,
				MappingsDir:    actualMappingsDir, // Use the determined mappings directory
				ExtraParams:    extraParams,
				NonInteractive: nonInteractive,
				VerboseLogging: verbose,
			}

			// Generate remediation plan
			if verbose {
				fmt.Printf("Generating remediation plan using mapping file: %s\n", mappingFile)
			}
			plan, err := GenerateRemediationPlan(report, mappingFile, options)
			if err != nil {
				fmt.Printf("Error generating remediation plan: %v\n", err)
				os.Exit(1)
			}

			// Output the plan
			if outputFile == "" {
				// Print to stdout - default to YAML for better readability
				planOutput, err := format.FormatData(plan, true) // true = YAML
				if err != nil {
					fmt.Printf("Error formatting plan: %v\n", err)
					os.Exit(1)
				}
				fmt.Print(planOutput)
			} else {
				// Save to file using format based on extension
				if verbose {
					fmt.Printf("Saving remediation plan to: %s\n", outputFile)
				}
				err = format.WriteFile(outputFile, plan)
				if err != nil {
					fmt.Printf("Error writing output file: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("Remediation plan saved to %s\n", outputFile)
			}
		},
	}

	// Configure flags
	generateCmd.Flags().StringP("output", "o", "", "Output file for remediation plan")
	generateCmd.Flags().StringP("mappings", "m", "mappings.yaml", "Path to mappings configuration file")
	generateCmd.Flags().StringP("mappings-dir", "d", "", "Directory to search for mapping references (defaults to global library mappings)")
	generateCmd.Flags().StringP("params", "p", "", "JSON or YAML file with additional parameters")
	generateCmd.Flags().String("params-json", "", "JSON string with additional parameters")
	generateCmd.Flags().StringP("repo", "r", "", "Path to repository (for parameter inference)")
	generateCmd.Flags().BoolP("non-interactive", "n", false, "Do not prompt for missing parameters")
	generateCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	return generateCmd
}

// loadParameters loads parameters from a file or JSON string
func loadParameters(paramsFile, paramsJSON string) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	// Load from file if specified
	if paramsFile != "" {
		data, err := os.ReadFile(paramsFile)
		if err != nil {
			return nil, fmt.Errorf("error reading params file: %w", err)
		}

		// Determine format based on extension
		if strings.HasSuffix(paramsFile, ".yaml") || strings.HasSuffix(paramsFile, ".yml") {
			if err := yaml.Unmarshal(data, &params); err != nil {
				return nil, fmt.Errorf("error parsing YAML params: %w", err)
			}
		} else {
			// Default to JSON
			if err := json.Unmarshal(data, &params); err != nil {
				return nil, fmt.Errorf("error parsing JSON params: %w", err)
			}
		}
	}

	// Parse JSON string if provided
	if paramsJSON != "" {
		var jsonParams map[string]interface{}
		if err := json.Unmarshal([]byte(paramsJSON), &jsonParams); err != nil {
			return nil, fmt.Errorf("error parsing JSON param string: %w", err)
		}

		// Merge with file params (JSON string overrides file)
		for k, v := range jsonParams {
			params[k] = v
		}
	}

	return params, nil
}
