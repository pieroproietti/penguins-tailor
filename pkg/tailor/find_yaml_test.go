package tailor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindYamlAndLoadSuit(t *testing.T) {
	v2Dir, err := getWardrobeV2Dir()
	if err != nil {
		t.Fatalf("getWardrobeV2Dir error: %v", err)
	}

	if _, err := os.Stat(v2Dir); os.IsNotExist(err) {
		// Fallback to local repo v2 for test suite execution
		if localV2, err := filepath.Abs("../../v2"); err == nil {
			if _, err := os.Stat(localV2); err == nil {
				v2Dir = localV2
			}
		}
	}

	if _, err := os.Stat(v2Dir); os.IsNotExist(err) {
		t.Skipf("costumes directory %s not present; skipping live wardrobe test", v2Dir)
	}

	// Test costume colibri
	colibriDir := filepath.Join(v2Dir, "costumes", "colibri")
	colibriYaml := findYaml(colibriDir)
	if colibriYaml == "" {
		t.Skipf("costume colibri not found in %s; skipping", colibriDir)
	}
	colibriSuit, err := loadSuit(colibriYaml)
	if err != nil {
		t.Fatalf("loadSuit failed for colibri: %v", err)
	}
	if len(colibriSuit.Accessories) != 2 {
		t.Errorf("expected 2 accessories, got %d: %v", len(colibriSuit.Accessories), colibriSuit.Accessories)
	}

	// Test accessory eggs-dev
	eggsDevDir := filepath.Join(v2Dir, "accessories", "eggs-dev")
	eggsDevYaml := findYaml(eggsDevDir)
	if eggsDevYaml != "" {
		eggsDevSuit, err := loadSuit(eggsDevYaml)
		if err != nil {
			t.Fatalf("loadSuit failed for eggs-dev: %v", err)
		}
		if eggsDevSuit.Name != "eggs-dev" {
			t.Errorf("expected name 'eggs-dev', got '%s'", eggsDevSuit.Name)
		}
		if len(eggsDevSuit.Packages) == 0 {
			t.Errorf("expected packages in eggs-dev, got none")
		}
	}

	// Test accessory base
	baseDir := filepath.Join(v2Dir, "accessories", "base")
	baseYaml := findYaml(baseDir)
	if baseYaml != "" {
		baseSuit, err := loadSuit(baseYaml)
		if err != nil {
			t.Fatalf("loadSuit failed for base: %v", err)
		}
		if baseSuit.Name != "base" {
			t.Errorf("expected name 'base', got '%s'", baseSuit.Name)
		}
	}
}

func TestFindYamlAndLoadSuit_Fixture(t *testing.T) {
	tempDir := t.TempDir()
	costumeDir := filepath.Join(tempDir, "test-costume")
	if err := os.MkdirAll(costumeDir, 0755); err != nil {
		t.Fatalf("failed to create costume dir: %v", err)
	}

	yamlContent := `name: test-costume
description: A test costume
author: Piero Proietti
release: "1.0"
distributions:
  - debian
packages:
  - curl
  - git
accessories:
  - test-acc
`
	yamlPath := filepath.Join(costumeDir, "index.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write index.yaml: %v", err)
	}

	found := findYaml(costumeDir)
	if found != yamlPath {
		t.Fatalf("findYaml returned %q, expected %q", found, yamlPath)
	}

	suit, err := loadSuit(found)
	if err != nil {
		t.Fatalf("loadSuit failed: %v", err)
	}

	if suit.Name != "test-costume" {
		t.Errorf("expected name 'test-costume', got %q", suit.Name)
	}
	if len(suit.Packages) != 2 {
		t.Errorf("expected 2 packages, got %d", len(suit.Packages))
	}
	if len(suit.Accessories) != 1 || suit.Accessories[0] != "test-acc" {
		t.Errorf("unexpected accessories: %v", suit.Accessories)
	}
}
