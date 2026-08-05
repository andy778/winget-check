package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectWorkflowContent(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"winget-releaser.yml", true},
		{"microsoft-submission.yml", true},
		{"wingetcreate-cli.yml", true},
		{"standard-ci.yml", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			path := filepath.Join("..", "..", "testdata", "workflows", tt.filename)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read test file %s: %v", path, err)
			}

			got, detail := InspectWorkflowContent(string(data))
			if got != tt.want {
				t.Errorf("InspectWorkflowContent(%s) = %v (detail: %q); want %v", tt.filename, got, detail, tt.want)
			}
		})
	}
}
