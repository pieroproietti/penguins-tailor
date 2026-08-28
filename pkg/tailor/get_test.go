package tailor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeGitURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "https://github.com/pieroproietti/penguins-wardrobe",
			expected: "github.com/pieroproietti/penguins-wardrobe",
		},
		{
			input:    "https://github.com/pieroproietti/penguins-wardrobe.git",
			expected: "github.com/pieroproietti/penguins-wardrobe",
		},
		{
			input:    "https://github.com/pieroproietti/penguins-wardrobe/",
			expected: "github.com/pieroproietti/penguins-wardrobe",
		},
		{
			input:    "git@github.com:pieroproietti/penguins-wardrobe.git",
			expected: "github.com/pieroproietti/penguins-wardrobe",
		},
		{
			input:    "http://github.com/pieroproietti/penguins-wardrobe",
			expected: "github.com/pieroproietti/penguins-wardrobe",
		},
		{
			input:    "ssh://git@github.com/pieroproietti/penguins-wardrobe.git",
			expected: "github.com/pieroproietti/penguins-wardrobe",
		},
		{
			input:    "https://github.com/charliemartinez/penguins-wardrobe",
			expected: "github.com/charliemartinez/penguins-wardrobe",
		},
		{
			input:    "https://github.com/pieroproietti/penguins-wardrobe#main",
			expected: "github.com/pieroproietti/penguins-wardrobe",
		},
		{
			input:    "https://github.com/pieroproietti/penguins-wardrobe.git#dev",
			expected: "github.com/pieroproietti/penguins-wardrobe",
		},
	}

	for _, tt := range tests {
		got := normalizeGitURL(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeGitURL(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}

	// Verify equality of equivalents
	u1 := "https://github.com/pieroproietti/penguins-wardrobe"
	u2 := "git@github.com:pieroproietti/penguins-wardrobe.git"
	u3 := "https://github.com/charliemartinez/penguins-wardrobe"

	if normalizeGitURL(u1) != normalizeGitURL(u2) {
		t.Errorf("expected %q and %q to normalize equally", u1, u2)
	}

	if normalizeGitURL(u1) == normalizeGitURL(u3) {
		t.Errorf("expected %q and %q to normalize differently", u1, u3)
	}
}

func TestGetGitOrigin(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Non-existent dir or non-git dir
	if origin := getGitOrigin(tempDir); origin != "" {
		t.Errorf("expected empty origin for non-git dir, got %q", origin)
	}

	// 2. Git dir with .git/config
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
[remote "origin"]
	url = https://github.com/pieroproietti/penguins-wardrobe.git
	fetch = +refs/heads/*:refs/remotes/origin/*
[branch "main"]
	remote = origin
	merge = refs/heads/main
`
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write .git/config: %v", err)
	}

	origin := getGitOrigin(tempDir)
	expected := "https://github.com/pieroproietti/penguins-wardrobe.git"
	if origin != expected {
		t.Errorf("getGitOrigin() = %q, expected %q", origin, expected)
	}
}

func TestGetGitBranch(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Non-existent dir or non-git dir
	if branch := getGitBranch(tempDir); branch != "" {
		t.Errorf("expected empty branch for non-git dir, got %q", branch)
	}

	// 2. Git dir with .git/HEAD pointing to develop
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/develop\n"), 0644); err != nil {
		t.Fatalf("failed to write .git/HEAD: %v", err)
	}

	branch := getGitBranch(tempDir)
	if branch != "develop" {
		t.Errorf("getGitBranch() = %q, expected 'develop'", branch)
	}
}
