// SPDX-License-Identifier: Apache-2.0

package parameters

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kusari-oss/darn/internal/darnit"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func getInferCmd() *cobra.Command {
	inferCmd := &cobra.Command{
		Use:   "infer [directory]",
		Short: "Infer parameters from a repository",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			repoPath := "."
			if len(args) > 0 {
				repoPath = args[0]
			}
			outputFile, _ := cmd.Flags().GetString("output")
			outputFormat, _ := cmd.Flags().GetString("format")

			// Infer parameters
			params, err := darnit.InferParametersFromRepo(repoPath)
			if err != nil {
				fmt.Printf("Error inferring parameters: %v\n", err)
				os.Exit(1)
			}

			// Format the output
			var formatted []byte
			if outputFormat == "yaml" {
				formatted, err = yaml.Marshal(params)
			} else {
				formatted, err = json.MarshalIndent(params, "", "  ")
			}

			if err != nil {
				fmt.Printf("Error formatting parameters: %v\n", err)
				os.Exit(1)
			}

			// Print to stdout if no output file is specified
			if outputFile == "" {
				fmt.Println(string(formatted))
			} else {
				// Save to file
				if err := os.WriteFile(outputFile, formatted, 0644); err != nil {
					fmt.Printf("Error writing output file: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("Inferred parameters saved to %s\n", outputFile)
			}
		},
	}

	// Configure flags
	inferCmd.Flags().StringP("output", "o", "", "Output file for inferred parameters")
	inferCmd.Flags().StringP("format", "f", "json", "Output format (json or yaml)")

	return inferCmd
}
