// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kusari-oss/darn/cmd/darn/cmd/action"
	"github.com/kusari-oss/darn/cmd/darn/cmd/library"
	"github.com/kusari-oss/darn/internal/core/config"
	"github.com/kusari-oss/darn/internal/version"

	"github.com/spf13/cobra"
)

var (
	// Configuration path
	configFile string

	// Project directory
	projectDir string

	// Loaded configuration
	cfg *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "darn",
	Short: "Security Findings Remediation Tool",
	Long: `Darn is a command-line tool designed to manage and enforce security 
best practices in software projects. It provides a flexible framework 
for applying security remediation actions through templated files and CLI commands.`,
	Version: fmt.Sprintf("%s (commit: %s)", version.Version, version.Commit),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Determine project directory
		var err error
		if projectDir == "" {
			projectDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("error getting current directory: %w", err)
			}
		} else {
			projectDir, err = filepath.Abs(projectDir)
			if err != nil {
				return fmt.Errorf("error resolving project directory: %w", err)
			}
		}

		// Try to load project config, but don't fail if it doesn't exist
		configDir := filepath.Join(projectDir, config.DefaultConfigDir)
		configPath := filepath.Join(configDir, config.DefaultConfigFileName)

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// Project not initialized, use default config
			fmt.Println("No project configuration found. Using defaults.")
			cfg = config.NewDefaultConfig()
		} else {
			// Load configuration
			cfg, err = config.LoadConfig(projectDir)
			if err != nil {
				fmt.Printf("Warning: Error loading configuration: %v\n", err)
				fmt.Println("Using default configuration instead.")
				cfg = config.NewDefaultConfig()
			}
		}

		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(library.NewLibraryCommand())
	rootCmd.AddCommand(action.NewActionCmd())

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is .darn/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&projectDir, "project-dir", "", "project directory (default is current directory)")
}
