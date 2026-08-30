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

	installed, failed, err := applySuit(tempDir, suit, true, false)
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

func TestApplySuit_AccessoryMode(t *testing.T) {
	tempDir := t.TempDir()

	accYaml := `name: eggs-dev
description: Accessory with 6 packages
packages:
  - code
  - nodejs
  - build-essential
  - dpkg-dev
  - git
  - golang
`
	yamlPath := filepath.Join(tempDir, "index.yaml")
	if err := os.WriteFile(yamlPath, []byte(accYaml), 0644); err != nil {
		t.Fatalf("failed to write accessory fixture: %v", err)
	}

	suit, err := loadSuit(yamlPath)
	if err != nil {
		t.Fatalf("loadSuit failed: %v", err)
	}

	installed, failed, err := applySuit(tempDir, suit, true, true)
	if err != nil {
		t.Fatalf("applySuit in accessory dry-run mode failed: %v", err)
	}

	if len(failed) != 0 {
		t.Errorf("expected 0 failed packages, got %d: %v", len(failed), failed)
	}
	if len(installed) != 6 {
		t.Errorf("expected 6 installed packages for accessory, got %d: %v", len(installed), installed)
	}
}

func TestSequenceAndFinalizeSeparation(t *testing.T) {
	tempDir := t.TempDir()

	costumeYaml := `name: test-order
sequence:
  packages:
    - pkg1
  accessories:
    - acc1
  cmds:
    - echo "sequence command 1"
    - echo "sequence command 2"
finalize:
  customize: true
  cmds:
    - echo "finalize command 1"
    - update-initramfs -u
`
	yamlPath := filepath.Join(tempDir, "index.yaml")
	if err := os.WriteFile(yamlPath, []byte(costumeYaml), 0644); err != nil {
		t.Fatalf("failed to write fixture yaml: %v", err)
	}

	suit, err := loadSuit(yamlPath)
	if err != nil {
		t.Fatalf("loadSuit failed: %v", err)
	}

	if len(suit.SequenceCmds) != 2 {
		t.Errorf("expected 2 sequence commands, got %d: %v", len(suit.SequenceCmds), suit.SequenceCmds)
	}
	if len(suit.FinalizeCmds) != 2 {
		t.Errorf("expected 2 finalize commands, got %d: %v", len(suit.FinalizeCmds), suit.FinalizeCmds)
	}
	if suit.SequenceCmds[0] != "echo \"sequence command 1\"" {
		t.Errorf("unexpected first sequence command: %s", suit.SequenceCmds[0])
	}
	if suit.FinalizeCmds[1] != "update-initramfs -u" {
		t.Errorf("unexpected second finalize command: %s", suit.FinalizeCmds[1])
	}
}

func TestLegacyCmdsFallbackToFinalize(t *testing.T) {
	tempDir := t.TempDir()

	legacyYaml := `name: legacy-costume
packages:
  - pkg1
cmds:
  - echo "legacy cmd 1"
  - echo "legacy cmd 2"
`
	yamlPath := filepath.Join(tempDir, "index.yaml")
	if err := os.WriteFile(yamlPath, []byte(legacyYaml), 0644); err != nil {
		t.Fatalf("failed to write fixture yaml: %v", err)
	}

	suit, err := loadSuit(yamlPath)
	if err != nil {
		t.Fatalf("loadSuit failed: %v", err)
	}

	if len(suit.SequenceCmds) != 0 {
		t.Errorf("expected 0 sequence commands in legacy format, got %d", len(suit.SequenceCmds))
	}
	if len(suit.FinalizeCmds) != 2 {
		t.Errorf("expected 2 finalize commands in legacy format, got %d", len(suit.FinalizeCmds))
	}
}
