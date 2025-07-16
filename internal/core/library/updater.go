// SPDX-License-Identifier: Apache-2.0

package library

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Updater handles the updating of library files
type Updater struct {
	// Library path where files will be updated
	libraryPath string

	// Source directory containing the files to update from
	sourceDir string

	// Force update even if files exist
	force bool

	// Dry run mode only shows what would be updated
	dryRun bool

	// Enable verbose output
	verbose bool

	// Stats for tracking updates
	stats struct {
		Created  int
		Updated  int
		Skipped  int
		Examined int
	}
}

// NewUpdater creates a new library updater
func NewUpdater(libraryPath, sourceDir string, force, dryRun, verbose bool) *Updater {
	return &Updater{
		libraryPath: libraryPath,
		sourceDir:   sourceDir,
		force:       force,
		dryRun:      dryRun,
		verbose:     verbose,
	}
}

// UpdateLibrary updates the library with files from the source directory
func (u *Updater) UpdateLibrary() error {
	// Ensure the library path exists
	if err := os.MkdirAll(u.libraryPath, 0755); err != nil {
		return fmt.Errorf("error creating library directory: %w", err)
	}

	// Update templates
	if err := u.updateDirectory("templates", "templates"); err != nil {
		return err
	}

	// Update actions
	if err := u.updateDirectory("actions", "actions"); err != nil {
		return err
	}

	// Update configs
	if err := u.updateDirectory("configs", "configs"); err != nil {
		return err
	}

	// Update mappings
	if err := u.updateDirectory("mappings", "mappings"); err != nil {
		return err
	}

	// Print summary
	if u.dryRun {
		fmt.Println("\nDRY RUN SUMMARY:")
		fmt.Printf("Would examine: %d files\n", u.stats.Examined)
		fmt.Printf("Would create: %d files\n", u.stats.Created)
		fmt.Printf("Would update: %d files\n", u.stats.Updated)
		fmt.Printf("Would skip: %d files\n", u.stats.Skipped)
	} else {
		fmt.Println("\nUPDATE SUMMARY:")
		fmt.Printf("Examined: %d files\n", u.stats.Examined)
		fmt.Printf("Created: %d files\n", u.stats.Created)
		fmt.Printf("Updated: %d files\n", u.stats.Updated)
		fmt.Printf("Skipped: %d files\n", u.stats.Skipped)
	}

	// Update the state file to record the last update time
	if !u.dryRun {
		if err := u.updateStateFile(); err != nil {
			return fmt.Errorf("error updating state file: %w", err)
		}
	}

	return nil
}

// updateDirectory updates files in a specific category (templates, actions, configs)
func (u *Updater) updateDirectory(sourceSubdir, targetSubdir string) error {
	sourceDir := filepath.Join(u.sourceDir, sourceSubdir)
	targetDir := filepath.Join(u.libraryPath, targetSubdir)

	// Check if source directory exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		if u.verbose {
			fmt.Printf("Source directory does not exist: %s\n", sourceDir)
		}
		return nil // Skip if source doesn't exist
	}

	// Ensure target directory exists
	if !u.dryRun {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("error creating target directory %s: %w", targetDir, err)
		}
	}

	// Walk the source directory
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		u.stats.Examined++

		// Compute relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("error computing relative path: %w", err)
		}

		// Compute target path
		targetPath := filepath.Join(targetDir, relPath)

		// Check if update is needed
		needsUpdate, reason := u.needsUpdate(path, targetPath, info)
		if !needsUpdate {
			u.stats.Skipped++
			if u.verbose {
				fmt.Printf("Skipping %s: %s\n", relPath, reason)
			}
			return nil
		}

		// Ensure target directory exists
		targetDirPath := filepath.Dir(targetPath)
		if !u.dryRun {
			if err := os.MkdirAll(targetDirPath, 0755); err != nil {
				return fmt.Errorf("error creating directory %s: %w", targetDirPath, err)
			}
		}

		// Copy the file
		if u.dryRun {
			fmt.Printf("Would %s: %s -> %s\n", reason, relPath, targetPath)
			if reason == "create" {
				u.stats.Created++
			} else {
				u.stats.Updated++
			}
		} else {
			if err := u.copyFile(path, targetPath); err != nil {
				return fmt.Errorf("error copying %s: %w", relPath, err)
			}

			if u.verbose {
				fmt.Printf("%s: %s\n", strings.Title(reason), relPath)
			}

			if reason == "create" {
				u.stats.Created++
			} else {
				u.stats.Updated++
			}
		}

		return nil
	})
}

// needsUpdate checks if a file needs to be updated
func (u *Updater) needsUpdate(sourcePath, targetPath string, sourceInfo os.FileInfo) (bool, string) {
	// Check if target file exists
	targetInfo, err := os.Stat(targetPath)
	if os.IsNotExist(err) {
		return true, "create" // File doesn't exist in target
	}
	if err != nil {
		// Some other error occurred, better update
		return true, "update (error checking target)"
	}

	// If force is true, always update
	if u.force {
		return true, "update (forced)"
	}

	// Compare modification times
	if sourceInfo.ModTime().After(targetInfo.ModTime()) {
		return true, "update (newer)"
	}

	// Compare file sizes
	if sourceInfo.Size() != targetInfo.Size() {
		return true, "update (size different)"
	}

	// Files appear identical
	return false, "files identical"
}

// copyFile copies a file from source to target
func (u *Updater) copyFile(sourcePath, targetPath string) error {
	// Open source file
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	// Create target file
	target, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer target.Close()

	// Copy content
	_, err = io.Copy(target, source)
	if err != nil {
		return err
	}

	// Copy file mode
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}
	return os.Chmod(targetPath, sourceInfo.Mode())
}

// updateStateFile updates a simple timestamp file to track updates
func (u *Updater) updateStateFile() error {
	stateFile := filepath.Join(u.libraryPath, ".last_updated")
	timestamp := time.Now().Format(time.RFC3339)
	
	return os.WriteFile(stateFile, []byte(timestamp), 0644)
}
