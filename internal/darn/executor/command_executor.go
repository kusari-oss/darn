// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/kusari-oss/darn/internal/core/template"
)

// CommandExecutor handles running cli command functionality
type CommandExecutor struct {
	// Configuration fields
	command     string
	args        []string
	workingDir  string
	environment []string
	verbose     bool
}

// CommandResult holds the result of command execution
type CommandResult struct {
	Output     []byte
	Error      error
	ExitStatus int
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(command string, args []string) *CommandExecutor {
	return &CommandExecutor{
		command: command,
		args:    args,
	}
}

// WithWorkingDir sets the working directory
func (e *CommandExecutor) WithWorkingDir(dir string) *CommandExecutor {
	e.workingDir = dir
	return e
}

// WithEnvironment sets environment variables
func (e *CommandExecutor) WithEnvironment(env []string) *CommandExecutor {
	e.environment = env
	return e
}

// WithVerbose enables verbose output
func (e *CommandExecutor) WithVerbose(verbose bool) *CommandExecutor {
	e.verbose = verbose
	return e
}

// ProcessParameters processes command and arguments with template parameters
func (e *CommandExecutor) ProcessParameters(params map[string]interface{}) error {
	// Process command with templating
	processedCommand, err := template.ProcessString(e.command, params)
	if err != nil {
		return fmt.Errorf("error processing command: %w", err)
	}
	e.command = string(processedCommand)

	// Process args with templating
	processedArgs := make([]string, 0, len(e.args))
	for _, arg := range e.args {
		processedArg, err := template.ProcessString(arg, params)
		if err != nil {
			return fmt.Errorf("error processing argument: %w", err)
		}
		processedArgs = append(processedArgs, string(processedArg))
	}
	e.args = processedArgs

	// Set working directory if specified in params
	if workingDir, ok := params["working_dir"].(string); ok && workingDir != "" {
		e.workingDir = workingDir
	}

	// Set environment variables if specified in params
	if env, ok := params["environment"].([]interface{}); ok {
		envVars := os.Environ()
		for _, e := range env {
			if strEnv, ok := e.(string); ok {
				processedEnv, err := template.ProcessString(strEnv, params)
				if err != nil {
					return fmt.Errorf("error processing environment variable: %w", err)
				}
				envVars = append(envVars, string(processedEnv))
			}
		}
		e.environment = envVars
	}

	return nil
}

// Execute runs the command and returns its output
func (e *CommandExecutor) Execute() (*CommandResult, error) {
	// Create and configure the command
	cmd := exec.Command(e.command, e.args...)

	var stdout, stderr bytes.Buffer

	// Configure stdout/stderr based on verbosity
	if e.verbose {
		cmd.Stdout = io.MultiWriter(&stdout, os.Stdout)
		cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)
	} else {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	// Set working directory if specified
	if e.workingDir != "" {
		cmd.Dir = e.workingDir
	}

	// Set environment variables if specified
	if len(e.environment) > 0 {
		cmd.Env = e.environment
	}

	// Print the command being executed
	fmt.Printf("Executing: %s %s\n", e.command, strings.Join(e.args, " "))

	// Run the command
	err := cmd.Run()

	// Create result
	result := &CommandResult{
		Output: stdout.Bytes(),
		Error:  err,
	}

	if exitError, ok := err.(*exec.ExitError); ok {
		result.ExitStatus = exitError.ExitCode()
	}

	return result, err
}
