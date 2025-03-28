// SPDX-License-Identifier: Apache-2.0

package darnit_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/kusari-oss/darn/internal/darnit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupGitRepo creates a temporary Git repository for testing
func setupGitRepo(t *testing.T) string {
	tempDir := t.TempDir()

	// Initialize Git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	err := cmd.Run()
	require.NoError(t, err, "Failed to initialize git repository")

	// Set up git config for test user
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err, "Failed to set git user.name")

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err, "Failed to set git user.email")

	// Create test package.json
	packageJSON := `{
		"name": "test-project",
		"version": "1.0.0",
		"description": "Test project for inference tests"
	}`
	err = os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err, "Failed to create package.json")

	// Add and commit the file
	cmd = exec.Command("git", "add", "package.json")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err, "Failed to git add package.json")

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err, "Failed to git commit")

	// Set up a remote
	cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/test-org/test-repo.git")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err, "Failed to add git remote")

	return tempDir
}

func TestInferParametersFromRepo(t *testing.T) {
	// Skip if git is not available
	_, err := exec.LookPath("git")
	if err != nil {
		t.Skip("Git not available, skipping test")
	}

	// Set up a test repository
	repoPath := setupGitRepo(t)

	// Test inference
	params, err := darnit.InferParametersFromRepo(repoPath)
	require.NoError(t, err, "Error inferring parameters from repo")

	// Verify inferred parameters
	assert.NotNil(t, params, "Parameters should not be nil")
	assert.Equal(t, "https://github.com/test-org/test-repo.git", params["project_repo"], "Project repo URL incorrect")
	assert.Equal(t, "test-org", params["organization"], "Organization incorrect")
	assert.Equal(t, "test-repo", params["repo_name"], "Repository name incorrect")
	assert.Equal(t, "test-project", params["project_name"], "Project name incorrect")
}

func TestInferParametersFromRepoWithoutRemote(t *testing.T) {
	// Skip if git is not available
	_, err := exec.LookPath("git")
	if err != nil {
		t.Skip("Git not available, skipping test")
	}

	// Create a temporary directory
	tempDir := t.TempDir()

	// Initialize Git repo without remote
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err, "Failed to initialize git repository")

	// Set up git config
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err, "Failed to set git user.name")

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err, "Failed to set git user.email")

	// Create package.json
	packageJSON := `{
		"name": "local-project",
		"version": "1.0.0"
	}`
	err = os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err, "Failed to create package.json")

	// Add and commit the file
	cmd = exec.Command("git", "add", "package.json")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err, "Failed to git add package.json")

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err, "Failed to git commit")

	// Test inference with repo without remote
	params, err := darnit.InferParametersFromRepo(tempDir)
	require.NoError(t, err, "Error inferring parameters from repo")

	// Verify inferred parameters
	assert.NotNil(t, params, "Parameters should not be nil")
	assert.Equal(t, "local-project", params["project_name"], "Project name incorrect")
	assert.NotContains(t, params, "project_repo", "Project repo should not be inferred")
	assert.NotContains(t, params, "organization", "Organization should not be inferred")
	assert.NotContains(t, params, "repo_name", "Repository name should not be inferred")
}

func TestInferParametersFromInvalidDirectory(t *testing.T) {
	// Create a non-git directory
	tempDir := t.TempDir()

	// Test inference with non-git directory
	params, err := darnit.InferParametersFromRepo(tempDir)
	require.NoError(t, err, "Error should not be returned for non-git directory")

	// Verify minimal parameters are returned
	assert.NotNil(t, params, "Parameters should not be nil")
	assert.Empty(t, params, "No parameters should be inferred from non-git directory")
}

func TestInferParametersFromNonExistentDirectory(t *testing.T) {
	// Test inference with non-existent directory
	params, err := darnit.InferParametersFromRepo("/path/that/does/not/exist")
	assert.Error(t, err, "Error should be returned for non-existent directory")
	assert.Nil(t, params, "Parameters should be nil for non-existent directory")
	assert.Contains(t, err.Error(), "error changing to repo directory", "Error message should indicate directory issue")
}
