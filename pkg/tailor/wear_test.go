package tailor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplySuit_DryRun(t *testing.T) {
	tempDir := t.TempDir()

	costumeYaml := `name: test-costume
description: Test costume for dry-run
packages:
  - curl
  - wget
packages_no_recommends:
  - htop
packages_interactive:
  - debconf
sequence:
  repositories:
    sources_list:
      - contrib
      - non-free
    update: true
`
	yamlPath := filepath.Join(tempDir, "index.yaml")
	if err := os.WriteFile(yamlPath, []byte(costumeYaml), 0644); err != nil {
		t.Fatalf("failed to write fixture yaml: %v", err)
	}

	suit, err := loadSuit(yamlPath)
	if err != nil {
		t.Fatalf("loadSuit failed: %v", err)
	}

	installed, failed, err := applySuit(tempDir, suit, true)
	if err != nil {
		t.Fatalf("applySuit in dry-run mode failed: %v", err)
	}

	if len(failed) != 0 {
		t.Errorf("expected 0 failed packages in dry-run, got %d: %v", len(failed), failed)
	}

	// In dry-run, all packages (packages + packages_no_recommends + packages_interactive) are simulated as installed
	expectedTotal := len(suit.Packages) + len(suit.PackagesNoRecommends) + len(suit.PackagesInteractive)
	if len(installed) != expectedTotal {
		t.Errorf("expected %d installed packages in dry-run, got %d: %v", expectedTotal, len(installed), installed)
	}
}
