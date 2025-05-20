// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/kusari-oss/darn/cmd/darnit/cmd/mapping"
	"github.com/kusari-oss/darn/cmd/darnit/cmd/parameters"
	"github.com/kusari-oss/darn/cmd/darnit/cmd/plan"
	"github.com/kusari-oss/darn/internal/core/config"
	"github.com/kusari-oss/darn/internal/version"
	"github.com/spf13/cobra"
)

// libraryPathFlag holds the value from the --library-path flag.
var libraryPathFlag string

// Create the root command
var rootCmd = &cobra.Command{
	Use:   "darnit",
	Short: "Darnit - Security Findings Remediation Orchestration Tool",
	Long: `Darnit is a tool that analyzes security reports and generates remediation plans
using darn actions to implement security best practices in software projects.`,
	Version: fmt.Sprintf("%s (commit: %s)", version.Version, version.Commit),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration here to make it available to subcommands
		// Note: projectDir is typically determined dynamically, e.g., from current working directory.
		// For now, let's assume "." (current directory) for projectDir.
		// This might need adjustment based on how project context is handled in darnit.
		// projectDir := "." // Placeholder, adjust as needed - projectDir is no longer a parameter for LoadConfig
		// Pass an empty string for globalConfigPathOverride as darnit doesn't have this flag (yet)
		_, err := config.LoadConfig(libraryPathFlag, "")
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		// The loaded config can be stored in a context or a global variable if needed by command execution.
		// For this task, we are primarily concerned with passing libraryPathFlag to LoadConfig.
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add the --library-path persistent flag
	rootCmd.PersistentFlags().StringVar(&libraryPathFlag, "library-path", "", "Override the library path (e.g., ~/my-darn-library or /opt/darn/library)")

	rootCmd.AddCommand(plan.GetPlanCmd())
	rootCmd.AddCommand(parameters.GetParametersCmd())
	rootCmd.AddCommand(mapping.GetMappingCmd())
}
