// SPDX-License-Identifier: Apache-2.0

// TODO: Add more inference methods

package darnit

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// InferParametersFromRepo extracts information from a Git repository
func InferParametersFromRepo(repoPath string) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	// Change to repo directory if provided
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if repoPath != "" {
		if err := os.Chdir(repoPath); err != nil {
			return nil, fmt.Errorf("error changing to repo directory: %w", err)
		}
		defer os.Chdir(currentDir)
	}

	// Get remote URL
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err == nil {
		remoteURL := strings.TrimSpace(string(output))
		params["project_repo"] = remoteURL

		// Extract org/repo from URL
		// This is a simplified example - you'll need more robust parsing
		parts := strings.Split(remoteURL, "/")
		if len(parts) >= 2 {
			params["organization"] = parts[len(parts)-2]
			repoName := parts[len(parts)-1]
			repoName = strings.TrimSuffix(repoName, ".git")
			params["repo_name"] = repoName
		}
	}

	// Get project name from package.json, go.mod, etc.
	// This is just an example for one file type
	if _, err := os.Stat("package.json"); err == nil {
		data, err := os.ReadFile("package.json")
		if err == nil {
			var pkg struct {
				Name string `json:"name"`
			}
			if json.Unmarshal(data, &pkg) == nil && pkg.Name != "" {
				params["project_name"] = pkg.Name
			}
		}
	}

	return params, nil
}
