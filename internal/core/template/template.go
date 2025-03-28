// SPDX-License-Identifier: Apache-2.0

package template

import (
	"bytes"
	"fmt"
	"os"
	"text/template"
)

// ProcessFile processes a template file with the given parameters
func ProcessFile(filePath string, params map[string]interface{}) ([]byte, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("template file does not exist: %s", filePath)
	}

	// Read template file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading template file: %w", err)
	}

	return ProcessString(string(content), params)
}

// ProcessString processes a template string with the given parameters
func ProcessString(text string, params map[string]interface{}) ([]byte, error) {
	// Create a new template
	tmpl, err := template.New("template").Option("missingkey=error").Parse(text)
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %w", err)
	}

	// Execute the template with the parameters
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return nil, fmt.Errorf("error executing template: %w", err)
	}

	return buf.Bytes(), nil
}
