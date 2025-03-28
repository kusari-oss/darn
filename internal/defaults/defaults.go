// SPDX-License-Identifier: Apache-2.0

// TODO: Defaults probably shouldn't be embedded and we should have a separate library folder or repository for them.

package defaults

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed actions/* templates/* configs/* mappings/*
var embeddedFiles embed.FS

// DefaultsConfig stores configuration for where to fetch defaults
type DefaultsConfig struct {
	// Base URL for remote defaults
	DefaultsURL string `json:"defaults_url"`

	// Whether to attempt to fetch remote defaults
	UseRemote bool `json:"use_remote"`

	// Timeout for remote fetch operations in seconds
	Timeout int `json:"timeout"`
}

// NewDefaultsConfig creates a default configuration
func NewDefaultsConfig() DefaultsConfig {
	return DefaultsConfig{
		DefaultsURL: "https://raw.githubusercontent.com/kusari-oss/darn-defaults/main",
		UseRemote:   true,
		Timeout:     5,
	}
}

// Manager manages access to default files
type Manager struct {
	config DefaultsConfig
}

// NewManager creates a new defaults manager
func NewManager(config DefaultsConfig) *Manager {
	return &Manager{
		config: config,
	}
}

// CopyDefaults copies default files to the specified directories
func (m *Manager) CopyDefaults(templatesDir, actionsDir, configsDir, mappingsDir string, useRemote bool) (bool, error) {
	usedRemote := false

	// Try remote first if enabled
	if useRemote && m.config.UseRemote {
		fmt.Println("Attempting to fetch latest defaults from remote...")

		remoteSuccess, err := m.copyRemoteDefaults(templatesDir, actionsDir, configsDir, mappingsDir)
		if err != nil {
			fmt.Printf("Warning: Failed to fetch remote defaults: %v\n", err)
			fmt.Println("Falling back to embedded defaults...")
		} else if remoteSuccess {
			fmt.Println("Successfully fetched remote defaults.")
			usedRemote = true
		}
	}

	// If remote didn't succeed, use embedded
	if !usedRemote {
		fmt.Println("Using embedded defaults...")
		if err := m.copyEmbeddedDefaults(templatesDir, actionsDir, configsDir, mappingsDir); err != nil {
			return false, fmt.Errorf("error copying embedded defaults: %w", err)
		}
	}

	return usedRemote, nil
}

// copyEmbeddedDefaults copies files from embedded filesystem
func (m *Manager) copyEmbeddedDefaults(templatesDir, actionsDir, configsDir, mappingsDir string) error {
	// Copy templates
	if err := m.copyEmbeddedDir("templates", templatesDir); err != nil {
		return fmt.Errorf("error copying templates: %w", err)
	}

	// Copy actions
	if err := m.copyEmbeddedDir("actions", actionsDir); err != nil {
		return fmt.Errorf("error copying actions: %w", err)
	}

	// Copy configs
	if err := m.copyEmbeddedDir("configs", configsDir); err != nil {
		return fmt.Errorf("error copying configs: %w", err)
	}

	// Copy mappings
	if err := m.copyEmbeddedDir("mappings", mappingsDir); err != nil {
		return fmt.Errorf("error copying mappings: %w", err)
	}

	return nil
}

// copyEmbeddedDir recursively copies files from the embedded filesystem to the target directory
func (m *Manager) copyEmbeddedDir(srcDir, dstDir string) error {
	// List files in the source directory
	entries, err := embeddedFiles.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %w", srcDir, err)
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dstDir, err)
	}

	// Copy each file
	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectories
			if err := m.copyEmbeddedDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := m.copyEmbeddedFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyEmbeddedFile copies a single file from the embedded filesystem to the target path
func (m *Manager) copyEmbeddedFile(srcPath, dstPath string) error {
	// Open source file
	src, err := embeddedFiles.Open(srcPath)
	if err != nil {
		return fmt.Errorf("error opening source file %s: %w", srcPath, err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("error creating destination file %s: %w", dstPath, err)
	}
	defer dst.Close()

	// Copy content
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("error copying file content: %w", err)
	}

	return nil
}

// copyRemoteDefaults attempts to fetch and copy defaults from a remote source
func (m *Manager) copyRemoteDefaults(templatesDir, actionsDir, configsDir, mappingsDir string) (bool, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(m.config.Timeout) * time.Second,
	}

	// First, fetch the manifest to get the list of files
	manifestURL := fmt.Sprintf("%s/manifest.json", m.config.DefaultsURL)
	resp, err := client.Get(manifestURL)
	if err != nil {
		return false, fmt.Errorf("error fetching manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("manifest not found, status: %d", resp.StatusCode)
	}

	var manifest struct {
		Files []string `json:"files"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return false, fmt.Errorf("error decoding manifest: %w", err)
	}

	// Process each file in the manifest
	for _, file := range manifest.Files {
		// Determine destination directory based on file path
		var dstDir string
		if strings.HasPrefix(file, "templates/") {
			dstDir = templatesDir
		} else if strings.HasPrefix(file, "actions/") {
			dstDir = actionsDir
		} else if strings.HasPrefix(file, "configs/") {
			dstDir = configsDir
		} else if strings.HasPrefix(file, "mappings/") {
			dstDir = mappingsDir
		} else {
			return false, fmt.Errorf("unknown file category: %s", file)
		}

		// Create the destination path
		relPath := strings.SplitN(file, "/", 2)[1] // Remove category prefix
		dstPath := filepath.Join(dstDir, relPath)

		// Create parent directory if needed
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return false, fmt.Errorf("error creating directory for %s: %w", dstPath, err)
		}

		// Fetch the file
		fileURL := fmt.Sprintf("%s/%s", m.config.DefaultsURL, file)
		if err := m.downloadFile(fileURL, dstPath); err != nil {
			return false, fmt.Errorf("error downloading %s: %w", file, err)
		}
	}

	return true, nil
}

// downloadFile downloads a file from a URL to a local path
func (m *Manager) downloadFile(url, dstPath string) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(m.config.Timeout) * time.Second,
	}

	// Get the file
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("file not found, status: %d", resp.StatusCode)
	}

	// Create the destination file
	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy content
	_, err = io.Copy(dst, resp.Body)
	return err
}

// ListEmbeddedFiles returns a list of all embedded default files
func (m *Manager) ListEmbeddedFiles() ([]string, error) {
	var files []string

	err := fs.WalkDir(embeddedFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking embedded files: %w", err)
	}

	return files, nil
}
