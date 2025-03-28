// SPDX-License-Identifier: Apache-2.0

package template_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kusari-oss/darn/internal/core/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessString(t *testing.T) {
	tests := []struct {
		name     string
		template string
		params   map[string]interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "simple substitution",
			template: "Hello, {{.name}}!",
			params:   map[string]interface{}{"name": "World"},
			expected: "Hello, World!",
			wantErr:  false,
		},
		{
			name:     "multiple substitutions",
			template: "Project: {{.name}}, Repo: {{.repo}}",
			params:   map[string]interface{}{"name": "Darn", "repo": "github.com/kusari-oss/darn"},
			expected: "Project: Darn, Repo: github.com/kusari-oss/darn",
			wantErr:  false,
		},
		{
			name:     "missing parameter",
			template: "Hello, {{.missing}}!",
			params:   map[string]interface{}{"name": "World"},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "array parameter",
			template: "Emails: {{range .emails}}{{.}}, {{end}}",
			params: map[string]interface{}{
				"emails": []string{"user1@example.com", "user2@example.com"},
			},
			expected: "Emails: user1@example.com, user2@example.com, ",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := template.ProcessString(tt.template, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, string(result))
			}
		})
	}
}

func TestProcessFile(t *testing.T) {
	// Create a temporary template file
	tempDir := t.TempDir()
	templatePath := filepath.Join(tempDir, "test.tmpl")
	templateContent := "# Security Policy for {{.name}}\n\nEmails:\n{{- range .emails}}\n- {{.}}\n{{- end}}"

	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err, "Failed to create test template file")

	// Test parameters
	params := map[string]interface{}{
		"name":   "Test Project",
		"emails": []string{"security@example.com", "admin@example.com"},
	}

	// Expected result
	expected := "# Security Policy for Test Project\n\nEmails:\n- security@example.com\n- admin@example.com"

	// Process the file
	result, err := template.ProcessFile(templatePath, params)
	require.NoError(t, err)

	// Verify the result
	assert.Equal(t, expected, string(result))

	// Test with non-existent file
	_, err = template.ProcessFile(filepath.Join(tempDir, "nonexistent.tmpl"), params)
	assert.Error(t, err)
}
