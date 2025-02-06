package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrite2File(t *testing.T) {
	// Temporary directory for test files
	tempDir, err := os.MkdirTemp("", "kubeedge-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test cases
	testCases := []struct {
		name     string
		data     interface{}
		fileName string
		wantErr  bool
	}{
		{
			name: "Write struct to file",
			data: struct {
				Name string `yaml:"name"`
				Age  int    `yaml:"age"`
			}{
				Name: "John Doe",
				Age:  30,
			},
			fileName: "test_struct.yaml",
			wantErr:  false,
		},
		{
			name:     "Write map to file",
			data:     map[string]string{"key": "value"},
			fileName: "test_map.yaml",
			wantErr:  false,
		},
		{
			name:     "Write nil data",
			data:     nil,
			fileName: "test_nil.yaml",
			wantErr:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Construct full file path
			filePath := filepath.Join(tempDir, tc.fileName)

			// Call the function
			err := Write2File(filePath, tc.data)

			// Check for expected error
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Verify file was created
			_, err = os.Stat(filePath)
			assert.NoError(t, err)

			// Read and verify file contents
			content, err := os.ReadFile(filePath)
			assert.NoError(t, err)
			assert.NotEmpty(t, content)
		})
	}
}
