package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTechnicalLoggerWritesStructuredEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tailor.log")
	logger := NewTechnicalLogger(path)

	if err := logger.Info("wear started", LogField{Key: "costume", Value: "colibri"}); err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if err := logger.Warn("package failed", LogField{Key: "package", Value: "policykit 1"}); err != nil {
		t.Fatalf("Warn() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(data)
	if !strings.Contains(got, " INFO  wear started costume=colibri") {
		t.Errorf("missing structured info event in %q", got)
	}
	if !strings.Contains(got, ` WARN  package failed package="policykit 1"`) {
		t.Errorf("missing structured warning event in %q", got)
	}
}

func TestExecLogOnlyCapturesCommandOutputWithoutTerminalWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tailor.log")
	command := "printf 'standard output\\n'; printf 'standard error\\n' >&2"

	if err := ExecLogOnly(command, path); err != nil {
		t.Fatalf("ExecLogOnly() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(data)
	for _, want := range []string{
		`INFO  command started command="printf 'standard output\\n'; printf 'standard error\\n' >&2"`,
		"COMMAND stdout standard output",
		"COMMAND stderr standard error",
		"INFO  command finished exit_code=0",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("log missing %q in %q", want, got)
		}
	}
}
