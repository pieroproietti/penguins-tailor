package tailor

import (
	"os"
	"path/filepath"
	"testing"
)

func withBrandingRoot(t *testing.T) string {
	t.Helper()
	original := brandingRoot
	brandingRoot = filepath.Join(t.TempDir(), "penguins-eggs.d", "branding")
	t.Cleanup(func() { brandingRoot = original })
	return brandingRoot
}

func TestApplyBrandingReplacesPreviousSelection(t *testing.T) {
	target := withBrandingRoot(t)
	v2Dir := t.TempDir()
	source := filepath.Join(v2Dir, "branding", "quirinux", "calamares")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "logo.png"), []byte("quirinux"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "old-brand"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := applyBranding(v2Dir, "quirinux", false); err != nil {
		t.Fatalf("applyBranding() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "old-brand")); !os.IsNotExist(err) {
		t.Fatalf("previous branding still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "quirinux")); !os.IsNotExist(err) {
		t.Fatalf("branding name directory must not be replicated: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 {
		t.Fatalf("branding root permissions = %04o, want 0755", info.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(target, "theme")); !os.IsNotExist(err) {
		t.Fatalf("legacy theme directory must not be replicated: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(target, "calamares", "logo.png"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "quirinux" {
		t.Fatalf("copied logo = %q, want quirinux", data)
	}
}

func TestApplyBrandingEmptySelectionRemovesPrevious(t *testing.T) {
	target := withBrandingRoot(t)
	if err := os.MkdirAll(filepath.Join(target, "old-brand"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := applyBranding(t.TempDir(), "", false); err != nil {
		t.Fatalf("applyBranding() error = %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("branding root still exists: %v", err)
	}
}

func TestApplyBrandingDryRunDoesNotChangeTarget(t *testing.T) {
	target := withBrandingRoot(t)
	v2Dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(v2Dir, "branding", "quirinux"), 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(target, "old-brand", "marker")
	if err := os.MkdirAll(filepath.Dir(marker), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := applyBranding(v2Dir, "quirinux", true); err != nil {
		t.Fatalf("applyBranding() error = %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("dry-run changed existing branding: %v", err)
	}
}

func TestApplyBrandingRejectsInvalidOrMissingSelection(t *testing.T) {
	withBrandingRoot(t)
	v2Dir := t.TempDir()
	for _, name := range []string{"../quirinux", "missing"} {
		if err := applyBranding(v2Dir, name, false); err == nil {
			t.Fatalf("applyBranding(%q) succeeded, want error", name)
		}
	}
}

func TestLoadSuitReadsBranding(t *testing.T) {
	yamlPath := filepath.Join(t.TempDir(), "index.yaml")
	if err := os.WriteFile(yamlPath, []byte("name: colibri\nbranding: quirinux\n"), 0644); err != nil {
		t.Fatal(err)
	}

	suit, err := loadSuit(yamlPath)
	if err != nil {
		t.Fatalf("loadSuit() error = %v", err)
	}
	if suit.Branding != "quirinux" {
		t.Fatalf("Branding = %q, want quirinux", suit.Branding)
	}
}
