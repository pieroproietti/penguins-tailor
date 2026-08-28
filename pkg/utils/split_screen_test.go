package utils

import (
	"strings"
	"testing"
)

func TestFormatHeaderLines(t *testing.T) {
	// Test 1: Full config with custom branch
	cfg1 := SplitScreenConfig{
		Icon:    "👗",
		Atelier: "https://github.com/pieroproietti/penguins-wardrobe",
		Costume: "COSTUME: standard (v2.0)",
		Branch:  "develop",
		Notes:   "Costume standard per pinguini - Preseed applied",
	}
	lines1 := FormatHeaderLines(cfg1)
	if len(lines1) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines1), lines1)
	}
	if !strings.Contains(lines1[0], "Atelier:") || !strings.Contains(lines1[0], "https://github.com/pieroproietti/penguins-wardrobe") || !strings.Contains(lines1[0], "COSTUME: standard (v2.0)") {
		t.Errorf("line 0 unexpected: %q", lines1[0])
	}
	if !strings.Contains(lines1[1], "Branch:") || !strings.Contains(lines1[1], "develop") {
		t.Errorf("line 1 unexpected: %q", lines1[1])
	}
	if !strings.Contains(lines1[2], "Note:") || !strings.Contains(lines1[2], "Costume standard per pinguini - Preseed applied") {
		t.Errorf("line 2 unexpected: %q", lines1[2])
	}

	// Test 2: Default branch (main/master) should NOT output line 2
	cfg2 := SplitScreenConfig{
		Icon:    "👗",
		Atelier: "https://github.com/pieroproietti/penguins-wardrobe",
		Costume: "COSTUME: standard",
		Branch:  "main",
		Notes:   "Note di test",
	}
	lines2 := FormatHeaderLines(cfg2)
	if len(lines2) != 2 {
		t.Fatalf("expected 2 lines for main branch, got %d: %v", len(lines2), lines2)
	}
	if strings.Contains(strings.Join(lines2, "\n"), "Branch:") {
		t.Errorf("expected no Branch line for main, got %v", lines2)
	}
	if !strings.Contains(lines2[1], "Note: Note di test") {
		t.Errorf("expected Note in line 2, got %q", lines2[1])
	}

	// Test 3: Local mode without Atelier
	cfg3 := SplitScreenConfig{
		Icon:    "👝",
		Costume: "ACCESSORY: firmwares",
		Notes:   "Hardware firmware packages",
	}
	lines3 := FormatHeaderLines(cfg3)
	if len(lines3) != 2 {
		t.Fatalf("expected 2 lines without Atelier, got %d: %v", len(lines3), lines3)
	}
	if !strings.Contains(lines3[0], "ACCESSORY: firmwares") {
		t.Errorf("expected Costume title in line 0, got %q", lines3[0])
	}
	if !strings.Contains(lines3[1], "Note: Hardware firmware packages") {
		t.Errorf("expected Note in line 1, got %q", lines3[1])
	}
}
