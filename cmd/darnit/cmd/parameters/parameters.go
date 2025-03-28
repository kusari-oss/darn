// SPDX-License-Identifier: Apache-2.0

package parameters

import (
	"github.com/spf13/cobra"
)

var parametersCmd = &cobra.Command{
	Use:   "parameters",
	Short: "Manage parameters",
	Long:  `Commands for working with parameters.`,
}

// GetParametersCmd returns the parameters command
func GetParametersCmd() *cobra.Command {
	return parametersCmd
}

func init() {
	// Add subcommands
	parametersCmd.AddCommand(getInferCmd())
}
