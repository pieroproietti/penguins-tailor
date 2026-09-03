package utils

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFormatHeaderLines(t *testing.T) {
	// Test 1: Config without icon (standard wear style)
	cfg1 := SplitScreenConfig{
		Atelier: "https://github.com/pieroproietti/penguins-wardrobe",
		System:  "debian-bookworm",
		Costume: "Costume: standard (v2.0)",
		Branch:  "develop",
		Notes:   "Standard penguin costume - Preseed applied",
	}
	lines1 := FormatHeaderLines(cfg1)
	if len(lines1) != 4 {
		t.Fatalf("expected 4 lines, got %d: %v", len(lines1), lines1)
	}
	if !strings.Contains(lines1[0], "Atelier:") || !strings.Contains(lines1[0], "https://github.com/pieroproietti/penguins-wardrobe (develop)") {
		t.Errorf("line 0 unexpected: %q", lines1[0])
	}
	if strings.Contains(lines1[0], "👗") {
		t.Errorf("expected no icon in line 0, got %q", lines1[0])
	}
	if !strings.Contains(lines1[1], "Distribution:") || !strings.Contains(lines1[1], "debian-bookworm") {
		t.Errorf("line 1 unexpected: %q", lines1[1])
	}
	if !strings.Contains(lines1[2], "Costume:") || !strings.Contains(lines1[2], "standard (v2.0)") {
		t.Errorf("line 2 unexpected: %q", lines1[2])
	}
	if !strings.Contains(lines1[3], "Standard penguin costume - Preseed applied") {
		t.Errorf("line 3 unexpected: %q", lines1[3])
	}
	if strings.Contains(lines1[3], "Note:") || strings.Contains(lines1[3], "Nome:") {
		t.Errorf("expected no Note:/Nome: label in line 3, got %q", lines1[3])
	}

	// Test 2: Default branch without icon
	cfg2 := SplitScreenConfig{
		Atelier: "https://github.com/pieroproietti/penguins-wardrobe",
		Costume: "Costume: standard",
		Branch:  "main",
		Notes:   "Test note",
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
	if !strings.Contains(lines2[2], "Test note") || strings.Contains(lines2[2], "Note:") {
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
		Notes:   "Test note",
	}
	lines4 := FormatHeaderLines(cfg4)
	if !strings.Contains(lines4[0], "👗") {
		t.Errorf("expected icon in line 0 when explicitly provided, got %q", lines4[0])
	}
}

func TestDrawHeader_NoTopDivider(t *testing.T) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	ss := &SplitScreen{
		totalCols:   80,
		headerLines: []string{"  Atelier: origin", "  Costume: standard"},
	}
	ss.drawHeader()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (2 header lines + 1 bottom divider), got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[0], "Atelier: origin") {
		t.Errorf("expected line 0 to contain Atelier: origin, got %q", lines[0])
	}
	if strings.Contains(lines[0], "═") {
		t.Errorf("expected line 0 NOT to be a divider, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "Costume: standard") {
		t.Errorf("expected line 1 to contain Costume: standard, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "═") {
		t.Errorf("expected line 2 to be bottom divider, got %q", lines[2])
	}
}

func TestProgressiveRendererOnlyRewritesCurrentLine(t *testing.T) {
	ss := &SplitScreen{
		active:    true,
		totalCols: 80,
		stopChan:  make(chan struct{}),
	}

	output := captureSplitScreenOutput(t, func() {
		ss.SetAction("First action")
		firstSince := ss.currentSince
		time.Sleep(time.Millisecond)
		ss.SetAction("Second action")
		if !ss.currentSince.After(firstSince) {
			t.Error("new action did not reset its timer")
		}
		ss.mu.Lock()
		ss.renderCurrentLocked(asciiSpinnerFrames[1])
		ss.mu.Unlock()
		ss.AddStep("[OK] Second action")
		ss.AddPackageStep("Accessory: base", 2, 1, 0)
	})

	for _, forbidden := range []string{"\033[2J", ";1H"} {
		if strings.Contains(output, forbidden) {
			t.Errorf("progressive output contains forbidden terminal control %q: %q", forbidden, output)
		}
	}
	if strings.Count(output, "\r\033[2K") < 4 {
		t.Errorf("expected current-line rewrites, got %q", output)
	}
	for _, want := range []string{"[INFO] First action", "[OK] Second action", "Accessory: base — 2 installed · 1 unavailable · 0 failed"} {
		if !strings.Contains(output, want) {
			t.Errorf("progressive output missing %q in %q", want, output)
		}
	}
	if strings.Index(output, "[INFO] First action") > strings.Index(output, "[OK] Second action") {
		t.Errorf("completed lines are not kept in order: %q", output)
	}
}

func TestCloseFinalizesCurrentLineWithoutTerminalPadding(t *testing.T) {
	ss := &SplitScreen{
		active:      true,
		installed:   4,
		unavailable: 1,
		failed:      2,
		stopChan:    make(chan struct{}),
	}

	output := captureSplitScreenOutput(t, func() {
		ss.SetAction("Final operation")
		ss.Close()
	})
	for _, forbidden := range []string{"\033[2J", ";1H", "\n\n"} {
		if strings.Contains(output, forbidden) {
			t.Errorf("Close() contains forbidden terminal output %q: %q", forbidden, output)
		}
	}
	for _, want := range []string{
		"[INFO] Final operation\n",
		"Packages: 4 installed · 1 unavailable · 2 failed\n",
		"\033[?25h",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("Close() missing %q in %q", want, output)
		}
	}
}

func TestExecInteractiveResumesWithoutGlobalRedraw(t *testing.T) {
	ss := &SplitScreen{
		active:    true,
		totalCols: 80,
		stopChan:  make(chan struct{}),
	}
	logPath := filepath.Join(t.TempDir(), "interactive.log")
	output := captureSplitScreenOutput(t, func() {
		ss.SetAction("Awaiting interactive input")
		if err := ss.ExecInteractive("printf 'interactive output\\n'", logPath); err != nil {
			t.Fatalf("ExecInteractive() error = %v", err)
		}
		ss.SetAction("Continuing after interaction")
	})

	for _, forbidden := range []string{"\033[2J", ";1H", "Costume:"} {
		if strings.Contains(output, forbidden) {
			t.Errorf("ExecInteractive() redrew terminal state with %q: %q", forbidden, output)
		}
	}
	for _, want := range []string{"[INFO] Awaiting interactive input", "INTERACTIVE CONSOLE", "interactive output", "Continuing after interaction"} {
		if !strings.Contains(output, want) {
			t.Errorf("ExecInteractive() missing %q in %q", want, output)
		}
	}
	if strings.Index(output, "[INFO] Awaiting interactive input") > strings.Index(output, "INTERACTIVE CONSOLE") || strings.Index(output, "INTERACTIVE CONSOLE") > strings.Index(output, "interactive output") {
		t.Errorf("interactive output is not progressive: %q", output)
	}
	if ss.interactive {
		t.Error("interactive state remained enabled after command completion")
	}
}

func captureSplitScreenOutput(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}
