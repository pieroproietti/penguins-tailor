package tailor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteWearReportSeparatesUnavailableFromFailed(t *testing.T) {
	path, err := writeWearReport(wearReport{
		CostumeName:   "test-costume",
		System:        "debian-bookworm",
		Installed:     []string{"installed-package"},
		Unavailable:   []string{"missing-package"},
		FailedInstall: []string{"failed-package"},
	})
	if err != nil {
		t.Fatalf("writeWearReport() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(data)
	for _, want := range []string{
		"=== Installed (1) ===\ninstalled-package",
		"=== Unavailable (1) ===\nmissing-package",
		"=== Could NOT be installed (1) ===\nfailed-package",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("report missing %q in %q", want, got)
		}
	}
	if want := "tailor-report-debian-bookworm-"; !strings.HasPrefix(filepath.Base(path), want) {
		t.Errorf("report filename %q does not start with %q", filepath.Base(path), want)
	}
}

func TestWearReportFilenameWithoutIdentityKeepsLegacyFormat(t *testing.T) {
	stamp := time.Date(2026, 9, 3, 12, 34, 56, 0, time.UTC)
	if got, want := wearReportFilename("", stamp), "tailor-report-20260903-123456.txt"; got != want {
		t.Fatalf("wearReportFilename() = %q, want %q", got, want)
	}
}

func TestDistroLogPathIncludesIdentity(t *testing.T) {
	if got, want := distroLogPath("debian-bookworm"), "/var/log/tailor/tailor-debian-bookworm.log"; got != want {
		t.Fatalf("distroLogPath() = %q, want %q", got, want)
	}
}
