// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/kusari-oss/darn/cmd/darnit/cmd/mapping"
	"github.com/kusari-oss/darn/cmd/darnit/cmd/parameters"
	"github.com/kusari-oss/darn/cmd/darnit/cmd/plan"
	"github.com/kusari-oss/darn/internal/version"
	"github.com/spf13/cobra"
)

// Create the root command
var rootCmd = &cobra.Command{
	Use:   "darnit",
	Short: "Darnit - Security Findings Remediation Orchestration Tool",
	Long: `Darnit is a tool that analyzes security reports and generates remediation plans
using darn actions to implement security best practices in software projects.`,
	Version: fmt.Sprintf("%s (commit: %s)", version.Version, version.Commit),
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(plan.GetPlanCmd())
	rootCmd.AddCommand(parameters.GetParametersCmd())
	rootCmd.AddCommand(mapping.GetMappingCmd())
}
