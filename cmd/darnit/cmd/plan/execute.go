// SPDX-License-Identifier: Apache-2.0

package plan

import (
	"fmt"
	"os"

	"github.com/kusari-oss/darn/internal/core/models"
	"github.com/kusari-oss/darn/internal/darnit"
	"github.com/spf13/cobra"
)

func getExecuteCmd() *cobra.Command {
	executeCmd := &cobra.Command{
		Use:   "execute [plan-file]",
		Short: "Execute a remediation plan",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			planFile := args[0]
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			verbose, _ := cmd.Flags().GetBool("verbose")

			// Load the plan
			if verbose {
				fmt.Printf("Loading remediation plan from: %s\n", planFile)
			}
			plan, err := darnit.LoadPlanFile(planFile)
			if err != nil {
				fmt.Printf("Error loading plan: %v\n", err)
				os.Exit(1)
			}

			// Execute the plan
			executionOpts := models.ExecutionOptions{
				DryRun:         dryRun,
				VerboseLogging: verbose,
			}

			if verbose {
				fmt.Printf("Executing remediation plan with %d steps\n", len(plan.Steps))
			}

			if dryRun {
				fmt.Println("Running in dry-run mode - no actions will be executed")
			}

			err = darnit.ExecutePlan(plan, executionOpts)
			if err != nil {
				fmt.Printf("Error executing plan: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("Remediation plan executed successfully")
		},
	}

	// Configure flags
	executeCmd.Flags().BoolP("dry-run", "d", false, "Show what would be done without executing actions")
	executeCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	return executeCmd
}
