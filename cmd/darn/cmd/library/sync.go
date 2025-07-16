// SPDX-License-Identifier: Apache-2.0

package library

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kusari-oss/darn/internal/core/config"
	"github.com/kusari-oss/darn/internal/defaults"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:     "sync",
	Aliases: []string{"refresh"},
	Short:   "Sync the global library with latest embedded defaults",
	Long: `Sync the global library with the latest embedded defaults from the darn binary.

This command updates your global library (typically ~/.darn/library) with the latest 
action definitions, templates, configs, and mappings that are embedded in the darn binary.

This is useful when:
- You've updated darn and want the latest default actions/templates
- Your library is missing some default components
- You want to restore default functionality

Examples:
  darn library sync                    # Sync the currently configured global library
  darn library sync --library-path /custom/path  # Sync a specific library
  darn library sync --dry-run         # Show what would be synced without making changes
  darn library sync --force           # Overwrite existing files even if they're newer`,
	RunE: runSyncCommand,
}

var (
	syncLibraryPath string
	syncDryRun      bool
	syncForce       bool
	syncVerbose     bool
	syncLocalOnly   bool
)

func init() {
	syncCmd.Flags().StringVar(&syncLibraryPath, "library-path", "", "Path to library to sync (defaults to global library)")
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Show what would be synced without making changes")
	syncCmd.Flags().BoolVar(&syncForce, "force", false, "Overwrite existing files even if they're newer")
	syncCmd.Flags().BoolVarP(&syncVerbose, "verbose", "v", false, "Verbose output")
	syncCmd.Flags().BoolVar(&syncLocalOnly, "local-only", false, "Use only embedded defaults, don't attempt remote fetch")
}

func runSyncCommand(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig("", "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Determine target library path
	var targetLibraryPath string
	if syncLibraryPath != "" {
		targetLibraryPath = config.ExpandPathWithTilde(syncLibraryPath)
		if syncVerbose {
			fmt.Printf("Using library path from --library-path flag: %s\n", targetLibraryPath)
		}
	} else {
		// Use global library
		if cfg.LibraryManager == nil {
			cfg.LibraryManager = cfg.LibraryManager
		}
		
		libraryInfo, err := cfg.GetLibraryInfo()
		if err != nil {
			return fmt.Errorf("failed to resolve global library path: %w", err)
		}
		
		targetLibraryPath = libraryInfo.Path
		if syncVerbose {
			fmt.Printf("Using global library path: %s (source: %s)\n", targetLibraryPath, libraryInfo.Source)
		}
	}

	// Ensure target library exists
	absTargetPath, err := filepath.Abs(targetLibraryPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for %s: %w", targetLibraryPath, err)
	}

	if syncVerbose {
		fmt.Printf("Target library path: %s\n", absTargetPath)
	}

	// Create defaults manager
	defaultsConfig := defaults.DefaultsConfig{
		DefaultsURL: "https://raw.githubusercontent.com/kusari-oss/darn-defaults/main",
		UseRemote:   !syncLocalOnly,
		Timeout:     10,
	}
	manager := defaults.NewManager(defaultsConfig)

	// Define subdirectory paths
	subdirs := map[string]string{
		"actions":   filepath.Join(absTargetPath, "actions"),
		"templates": filepath.Join(absTargetPath, "templates"),
		"configs":   filepath.Join(absTargetPath, "configs"),
		"mappings":  filepath.Join(absTargetPath, "mappings"),
	}

	if syncDryRun {
		fmt.Printf("🔍 DRY RUN: Would sync to library at: %s\n", absTargetPath)
		
		// Check what exists
		for name, path := range subdirs {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				fmt.Printf("  Would create: %s/\n", name)
			} else {
				fmt.Printf("  Would update: %s/\n", name)
			}
		}
		
		fmt.Printf("\nTo perform the actual sync, run without --dry-run\n")
		return nil
	}

	fmt.Printf("🔄 Syncing library at: %s\n", absTargetPath)

	// Create target directories
	for name, path := range subdirs {
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", name, err)
		}
	}

	// Perform the sync using the defaults manager
	usedRemote, err := manager.CopyDefaults(
		subdirs["templates"],
		subdirs["actions"],
		subdirs["configs"],
		subdirs["mappings"],
		!syncLocalOnly,
	)
	if err != nil {
		return fmt.Errorf("failed to sync defaults: %w", err)
	}

	// Success message
	fmt.Printf("✅ Library sync complete!\n")
	fmt.Printf("  📁 Actions: %s\n", subdirs["actions"])
	fmt.Printf("  📄 Templates: %s\n", subdirs["templates"])
	fmt.Printf("  ⚙️  Configs: %s\n", subdirs["configs"])
	fmt.Printf("  🗺️  Mappings: %s\n", subdirs["mappings"])
	
	if syncLocalOnly {
		fmt.Printf("  📦 Used embedded defaults (local-only mode)\n")
	} else {
		fmt.Printf("  📦 Used %s defaults\n", map[bool]string{true: "remote", false: "embedded"}[usedRemote])
	}

	// Show some available actions
	fmt.Printf("\n📋 Available actions:\n")
	if actions, err := cfg.LibraryManager.ListAvailableActions(absTargetPath); err == nil {
		count := 0
		for _, action := range actions {
			if count >= 5 {
				break
			}
			fmt.Printf("  - %s\n", action)
			count++
		}
		if len(actions) > 5 {
			fmt.Printf("  ... and %d more\n", len(actions)-5)
		}
	}

	return nil
}