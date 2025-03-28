// SPDX-License-Identifier: Apache-2.0

package plan

import (
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Manage remediation plans",
	Long:  `Commands for generating and executing remediation plans.`,
}

func GetPlanCmd() *cobra.Command {
	return planCmd
}

func init() {
	planCmd.AddCommand(getGenerateCmd())
	planCmd.AddCommand(getExecuteCmd())
}
