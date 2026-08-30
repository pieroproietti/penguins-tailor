package utils

import (
	"strings"
	"testing"
)

func TestFormatHeaderLines(t *testing.T) {
	// Test 1: Config without icon (standard wear style)
	cfg1 := SplitScreenConfig{
		Atelier: "https://github.com/pieroproietti/penguins-wardrobe",
		Costume: "Costume: standard (v2.0)",
		Branch:  "develop",
		Notes:   "Costume standard per pinguini - Preseed applied",
	}
	lines1 := FormatHeaderLines(cfg1)
	if len(lines1) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines1), lines1)
	}
	if !strings.Contains(lines1[0], "Atelier:") || !strings.Contains(lines1[0], "https://github.com/pieroproietti/penguins-wardrobe (develop)") {
		t.Errorf("line 0 unexpected: %q", lines1[0])
	}
	if strings.Contains(lines1[0], "👗") {
		t.Errorf("expected no icon in line 0, got %q", lines1[0])
	}
	if !strings.Contains(lines1[1], "Costume:") || !strings.Contains(lines1[1], "standard (v2.0)") {
		t.Errorf("line 1 unexpected: %q", lines1[1])
	}
	if !strings.Contains(lines1[2], "Costume standard per pinguini - Preseed applied") {
		t.Errorf("line 2 unexpected: %q", lines1[2])
	}
	if strings.Contains(lines1[2], "Note:") || strings.Contains(lines1[2], "Nome:") {
		t.Errorf("expected no Note:/Nome: label in line 2, got %q", lines1[2])
	}

	// Test 2: Default branch without icon
	cfg2 := SplitScreenConfig{
		Atelier: "https://github.com/pieroproietti/penguins-wardrobe",
		Costume: "Costume: standard",
		Branch:  "main",
		Notes:   "Note di test",
	}
	lines2 := FormatHeaderLines(cfg2)
	if len(lines2) != 3 {
		t.Fatalf("expected 3 lines for main branch with atelier, got %d: %v", len(lines2), lines2)
	}
	if strings.Contains(lines2[0], "(main)") {
		t.Errorf("expected no (main) in line 0, got %q", lines2[0])
	}
	if !strings.Contains(lines2[1], "Costume:") || !strings.Contains(lines2[1], "standard") {
		t.Errorf("expected Costume in line 1, got %q", lines2[1])
	}
	if !strings.Contains(lines2[2], "Note di test") || strings.Contains(lines2[2], "Note:") {
		t.Errorf("expected description without Note: label, got %q", lines2[2])
	}

	// Test 3: Local mode without Atelier and without icon
	cfg3 := SplitScreenConfig{
		Costume: "Accessory: firmwares",
		Notes:   "Hardware firmware packages",
	}
	lines3 := FormatHeaderLines(cfg3)
	if len(lines3) != 2 {
		t.Fatalf("expected 2 lines without Atelier, got %d: %v", len(lines3), lines3)
	}
	if !strings.Contains(lines3[0], "Accessory:") || !strings.Contains(lines3[0], "firmwares") {
		t.Errorf("expected Accessory title in line 0, got %q", lines3[0])
	}
	if !strings.Contains(lines3[1], "Hardware firmware packages") || strings.Contains(lines3[1], "Note:") {
		t.Errorf("expected description in line 1 without Note: label, got %q", lines3[1])
	}

	// Test 4: Backwards compatibility with explicit icon
	cfg4 := SplitScreenConfig{
		Icon:    "👗",
		Atelier: "https://github.com/pieroproietti/penguins-wardrobe",
		Costume: "Costume: standard",
		Notes:   "Note di test",
	}
	lines4 := FormatHeaderLines(cfg4)
	if !strings.Contains(lines4[0], "👗") {
		t.Errorf("expected icon in line 0 when explicitly provided, got %q", lines4[0])
	}
}
