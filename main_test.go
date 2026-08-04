package main

import "testing"

func TestNormalizeRepo(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"github.com/owner/repo", "owner/repo"},
		{"https://github.com/owner/repo", "owner/repo"},
		{"http://github.com/owner/repo", "owner/repo"},
		{"https://github.com/owner/repo.git", "owner/repo"},
		{"  owner/repo  ", "owner/repo"},
		{"/owner/repo/", "owner/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeRepo(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeRepo(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestPackageIDFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{
			path:     "manifests/n/Notepad++/Notepad++/8.9/Notepad++.Notepad++.installer.yaml",
			expected: "Notepad++.Notepad++",
		},
		{
			path:     "manifests/g/Google/Chrome/100.0.0/Google.Chrome.yaml",
			expected: "Google.Chrome",
		},
		{
			path:     "invalid/path/to/file.txt",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := packageIDFromPath(tt.path)
			if got != tt.expected {
				t.Errorf("packageIDFromPath(%q) = %q; want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestVersionFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{
			path:     "manifests/n/Notepad++/Notepad++/8.9/Notepad++.Notepad++.installer.yaml",
			expected: "8.9",
		},
		{
			path:     "manifests/g/Google/Chrome/100.0.0/Google.Chrome.yaml",
			expected: "100.0.0",
		},
		{
			path:     "file.txt",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := versionFromPath(tt.path)
			if got != tt.expected {
				t.Errorf("versionFromPath(%q) = %q; want %q", tt.path, got, tt.expected)
			}
		})
	}
}
