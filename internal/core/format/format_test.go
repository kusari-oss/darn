// SPDX-License-Identifier: Apache-2.0

package format

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	Name  string `json:"name" yaml:"name"`
	Value int    `json:"value" yaml:"value"`
	Items []string `json:"items" yaml:"items"`
}

func TestParseData(t *testing.T) {
	testData := TestStruct{
		Name:  "test",
		Value: 42,
		Items: []string{"a", "b", "c"},
	}

	t.Run("ParseValidYAML", func(t *testing.T) {
		yamlData := `name: test
value: 42
items:
  - a
  - b
  - c`

		var result TestStruct
		err := ParseData([]byte(yamlData), &result)
		require.NoError(t, err)
		assert.Equal(t, testData, result)
	})

	t.Run("ParseValidJSON", func(t *testing.T) {
		jsonData := `{
  "name": "test",
  "value": 42,
  "items": ["a", "b", "c"]
}`

		var result TestStruct
		err := ParseData([]byte(jsonData), &result)
		require.NoError(t, err)
		assert.Equal(t, testData, result)
	})

	t.Run("ParseInvalidData", func(t *testing.T) {
		invalidData := `this is not valid yaml or json`

		var result TestStruct
		err := ParseData([]byte(invalidData), &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse as YAML")
		assert.Contains(t, err.Error(), "JSON")
	})
}

func TestParseFile(t *testing.T) {
	tempDir := t.TempDir()
	testData := TestStruct{
		Name:  "file-test",
		Value: 100,
		Items: []string{"x", "y"},
	}

	t.Run("ParseYAMLFile", func(t *testing.T) {
		yamlFile := filepath.Join(tempDir, "test.yaml")
		yamlContent := `name: file-test
value: 100
items:
  - x
  - y`
		err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
		require.NoError(t, err)

		var result TestStruct
		err = ParseFile(yamlFile, &result)
		require.NoError(t, err)
		assert.Equal(t, testData, result)
	})

	t.Run("ParseJSONFile", func(t *testing.T) {
		jsonFile := filepath.Join(tempDir, "test.json")
		jsonContent := `{
  "name": "file-test",
  "value": 100,
  "items": ["x", "y"]
}`
		err := os.WriteFile(jsonFile, []byte(jsonContent), 0644)
		require.NoError(t, err)

		var result TestStruct
		err = ParseFile(jsonFile, &result)
		require.NoError(t, err)
		assert.Equal(t, testData, result)
	})

	t.Run("ParseNonexistentFile", func(t *testing.T) {
		var result TestStruct
		err := ParseFile(filepath.Join(tempDir, "nonexistent.yaml"), &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading file")
	})
}

func TestWriteFile(t *testing.T) {
	tempDir := t.TempDir()
	testData := TestStruct{
		Name:  "write-test",
		Value: 200,
		Items: []string{"p", "q"},
	}

	t.Run("WriteYAMLFile", func(t *testing.T) {
		yamlFile := filepath.Join(tempDir, "output.yaml")
		err := WriteFile(yamlFile, testData)
		require.NoError(t, err)

		// Verify file was created and can be parsed back
		var result TestStruct
		err = ParseFile(yamlFile, &result)
		require.NoError(t, err)
		assert.Equal(t, testData, result)

		// Check content is YAML format
		content, err := os.ReadFile(yamlFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "name: write-test")
		assert.Contains(t, string(content), "value: 200")
	})

	t.Run("WriteJSONFile", func(t *testing.T) {
		jsonFile := filepath.Join(tempDir, "output.json")
		err := WriteFile(jsonFile, testData)
		require.NoError(t, err)

		// Verify file was created and can be parsed back
		var result TestStruct
		err = ParseFile(jsonFile, &result)
		require.NoError(t, err)
		assert.Equal(t, testData, result)

		// Check content is JSON format
		content, err := os.ReadFile(jsonFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), `"name": "write-test"`)
		assert.Contains(t, string(content), `"value": 200`)
	})

	t.Run("WriteNoExtension", func(t *testing.T) {
		// Should default to YAML
		noExtFile := filepath.Join(tempDir, "output")
		err := WriteFile(noExtFile, testData)
		require.NoError(t, err)

		// Verify it's YAML format
		content, err := os.ReadFile(noExtFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "name: write-test")
	})
}

func TestWriteYAML(t *testing.T) {
	tempDir := t.TempDir()
	testData := TestStruct{
		Name:  "yaml-test",
		Value: 300,
		Items: []string{"m", "n"},
	}

	yamlFile := filepath.Join(tempDir, "explicit.yaml")
	err := WriteYAML(yamlFile, testData)
	require.NoError(t, err)

	// Verify file content
	content, err := os.ReadFile(yamlFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "name: yaml-test")
	assert.Contains(t, string(content), "value: 300")

	// Verify it can be parsed back
	var result TestStruct
	err = ParseFile(yamlFile, &result)
	require.NoError(t, err)
	assert.Equal(t, testData, result)
}

func TestWriteJSON(t *testing.T) {
	tempDir := t.TempDir()
	testData := TestStruct{
		Name:  "json-test",
		Value: 400,
		Items: []string{"u", "v"},
	}

	jsonFile := filepath.Join(tempDir, "explicit.json")
	err := WriteJSON(jsonFile, testData)
	require.NoError(t, err)

	// Verify file content
	content, err := os.ReadFile(jsonFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), `"name": "json-test"`)
	assert.Contains(t, string(content), `"value": 400`)

	// Verify it can be parsed back
	var result TestStruct
	err = ParseFile(jsonFile, &result)
	require.NoError(t, err)
	assert.Equal(t, testData, result)
}

func TestFormatData(t *testing.T) {
	testData := TestStruct{
		Name:  "format-test",
		Value: 500,
		Items: []string{"w", "x"},
	}

	t.Run("FormatAsYAML", func(t *testing.T) {
		result, err := FormatData(testData, true)
		require.NoError(t, err)
		assert.Contains(t, result, "name: format-test")
		assert.Contains(t, result, "value: 500")
	})

	t.Run("FormatAsJSON", func(t *testing.T) {
		result, err := FormatData(testData, false)
		require.NoError(t, err)
		assert.Contains(t, result, `"name": "format-test"`)
		assert.Contains(t, result, `"value": 500`)
	})
}

func TestFileTypeDetection(t *testing.T) {
	tests := []struct {
		filename   string
		expectYAML bool
		expectJSON bool
	}{
		{"test.yaml", true, false},
		{"test.yml", true, false},
		{"test.YAML", true, false},
		{"test.json", false, true},
		{"test.JSON", false, true},
		{"test.txt", false, false},
		{"test", false, false},
		{"config.yaml", true, false},
		{"data.json", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			assert.Equal(t, tt.expectYAML, IsYAMLFile(tt.filename))
			assert.Equal(t, tt.expectJSON, IsJSONFile(tt.filename))
		})
	}
}