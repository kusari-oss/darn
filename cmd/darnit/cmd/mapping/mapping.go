// SPDX-License-Identifier: Apache-2.0

package mapping

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kusari-oss/darn/internal/core/config"
	"github.com/kusari-oss/darn/internal/darnit/plan"
	"github.com/spf13/cobra"
)

// GetMappingCmd returns the mapping command
func GetMappingCmd() *cobra.Command {
	mappingCmd := &cobra.Command{
		Use:   "mapping",
		Short: "Manage remediation mappings",
		Long:  `Commands for working with remediation mappings.`,
	}

	mappingCmd.AddCommand(getListCmd())
	mappingCmd.AddCommand(getValidateCmd())

	return mappingCmd
}

func getListCmd() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available mappings",
		Run: func(cmd *cobra.Command, args []string) {
			// Get working directory
			workingDir, err := os.Getwd()
			if err != nil {
				fmt.Printf("Error getting working directory")
				os.Exit(1)
			}

			// Load configuration
			cfg, err := config.LoadConfig(workingDir)
			if err != nil {
				fmt.Printf("Error loading configuration: %v\n", err)
				os.Exit(1)
			}

			// List mappings in the mappings directory
			mappingsDir := filepath.Join(workingDir, cfg.MappingsDir)
			if _, err := os.Stat(mappingsDir); os.IsNotExist(err) {
				fmt.Printf("Mappings directory %s does not exist\n", mappingsDir)
				os.Exit(1)
			}

			files, err := os.ReadDir(mappingsDir)
			if err != nil {
				fmt.Printf("Error reading mappings directory: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("Available mapping files:")
			for _, file := range files {
				if !file.IsDir() && (filepath.Ext(file.Name()) == ".yaml" || filepath.Ext(file.Name()) == ".yml") {
					// Load mapping to show description
					mappingPath := filepath.Join(mappingsDir, file.Name())
					mapping, err := plan.LoadMappingConfig(mappingPath)
					if err != nil {
						fmt.Printf("  %s (Error: %v)\n", file.Name(), err)
						continue
					}

					fmt.Printf("  %s - %d rule(s)\n", file.Name(), len(mapping.Mappings))
					for _, rule := range mapping.Mappings {
						fmt.Printf("    - %s: %s\n", rule.ID, rule.Reason)
					}
				}
			}
		},
	}

	return listCmd
}

func getValidateCmd() *cobra.Command {
	validateCmd := &cobra.Command{
		Use:   "validate [mapping-file]",
		Short: "Validate a mapping file",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			mappingPath := args[0]

			// Load the mapping file
			mapping, err := plan.LoadMappingConfig(mappingPath)
			if err != nil {
				fmt.Printf("Error loading mapping file: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Mapping file %s is valid\n", mappingPath)
			fmt.Printf("Contains %d rule(s)\n", len(mapping.Mappings))

			// TODO: Add more validation checks
		},
	}

	return validateCmd
}
