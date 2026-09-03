package tailor

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestWearRefreshesBeforeSuit(t *testing.T) {
	tempDir := t.TempDir()
	costumeDir := filepath.Join(tempDir, "v2", "costumes", "ordering")
	if err := os.MkdirAll(costumeDir, 0755); err != nil {
		t.Fatalf("failed to create costume fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(costumeDir, "index.yaml"), []byte("name: ordering\n"), 0644); err != nil {
		t.Fatalf("failed to write costume fixture: %v", err)
	}

	var events []string
	pm := &orderedPackageManager{events: &events}
	originalNewPackageManager := newWearPackageManager
	originalApplySuit := applyWearSuit
	originalGetWardrobeRoot := getWearWardrobeRoot
	originalGetWardrobeV2Dir := getWearWardrobeV2Dir
	newWearPackageManager = func(string) (PackageManager, error) { return pm, nil }
	applyWearSuit = func(string, *Suit, bool, bool, PackageManager) (PackageInstallResult, error) {
		events = append(events, "apply")
		return PackageInstallResult{}, nil
	}
	getWearWardrobeRoot = func() (string, error) { return tempDir, nil }
	getWearWardrobeV2Dir = func() (string, error) { return filepath.Join(tempDir, "v2"), nil }
	t.Cleanup(func() {
		newWearPackageManager = originalNewPackageManager
		applyWearSuit = originalApplySuit
		getWearWardrobeRoot = originalGetWardrobeRoot
		getWearWardrobeV2Dir = originalGetWardrobeV2Dir
	})

	if err := Wear("ordering", true, "", true); err != nil {
		t.Fatalf("Wear failed: %v", err)
	}

	want := []string{"refresh", "apply"}
	if !slices.Equal(events, want) {
		t.Fatalf("Wear lifecycle events = %v, want %v", events, want)
	}
}

func TestApplySuitChecksHeadersAfterRepositoryUpdate(t *testing.T) {
	var events []string
	pm := &orderedPackageManager{events: &events}
	originalEnsureHeaders := ensureWearHeaders
	ensureWearHeaders = func(bool) error {
		events = append(events, "headers")
		return nil
	}
	t.Cleanup(func() { ensureWearHeaders = originalEnsureHeaders })

	suit := &Suit{
		Name:     "ordering",
		Sequence: &Sequence{Repositories: &Repositories{Update: true}},
		Packages: []string{"dkms-module"},
	}
	if _, err := applySuit(t.TempDir(), suit, false, false, pm); err != nil {
		t.Fatalf("applySuit failed: %v", err)
	}

	want := []string{"refresh", "headers", "install"}
	if !slices.Equal(events, want) {
		t.Fatalf("applySuit lifecycle events = %v, want %v", events, want)
	}
}

func TestWearStopsWhenInitialRefreshFails(t *testing.T) {
	originalNewPackageManager := newWearPackageManager
	originalEnsureHeaders := ensureWearHeaders
	originalApplySuit := applyWearSuit
	refreshErr := errors.New("refresh failed")
	newWearPackageManager = func(string) (PackageManager, error) {
		return &orderedPackageManager{refreshErr: refreshErr}, nil
	}
	ensureWearHeaders = func(bool) error {
		t.Fatal("kernel headers checked after failed initial refresh")
		return nil
	}
	applyWearSuit = func(string, *Suit, bool, bool, PackageManager) (PackageInstallResult, error) {
		t.Fatal("suit applied after failed initial refresh")
		return PackageInstallResult{}, nil
	}
	t.Cleanup(func() {
		newWearPackageManager = originalNewPackageManager
		ensureWearHeaders = originalEnsureHeaders
		applyWearSuit = originalApplySuit
	})

	err := Wear("ordering", true, "", true)
	if !errors.Is(err, refreshErr) {
		t.Fatalf("Wear error = %v, want refresh error", err)
	}
}

type orderedPackageManager struct {
	events     *[]string
	refreshErr error
}

func (pm *orderedPackageManager) Refresh() error {
	if pm.events != nil {
		*pm.events = append(*pm.events, "refresh")
	}
	return pm.refreshErr
}

func (*orderedPackageManager) Upgrade(bool) error { return nil }

func (pm *orderedPackageManager) Install([]string, InstallMode) PackageInstallResult {
	if pm.events != nil {
		*pm.events = append(*pm.events, "install")
	}
	return PackageInstallResult{}
}

func (*orderedPackageManager) IsInstalled(string) bool { return false }

func (*orderedPackageManager) Heal() error { return nil }

func TestNewPackageManager(t *testing.T) {
	pm, err := newPackageManager("debian")
	if err != nil {
		t.Fatalf("newPackageManager(debian) failed: %v", err)
	}
	if _, ok := pm.(*aptPackageManager); !ok {
		t.Fatalf("newPackageManager(debian) returned %T, want *aptPackageManager", pm)
	}

	pm, err = newPackageManager("archlinux")
	if err == nil {
		t.Fatal("newPackageManager(archlinux) succeeded, want unsupported error")
	}
	if pm != nil {
		t.Fatalf("newPackageManager(archlinux) returned %T, want nil", pm)
	}
}

type fakePackageManager struct {
	installCalls []InstallMode
	refreshCalls int
	upgradeCalls []bool
	healCalls    int
	result       PackageInstallResult
	installed    map[string]bool
}

func (pm *fakePackageManager) Refresh() error {
	pm.refreshCalls++
	return nil
}

func (pm *fakePackageManager) Upgrade(refresh bool) error {
	pm.upgradeCalls = append(pm.upgradeCalls, refresh)
	return nil
}

func (pm *fakePackageManager) Install(_ []string, mode InstallMode) PackageInstallResult {
	pm.installCalls = append(pm.installCalls, mode)
	return pm.result
}

func (pm *fakePackageManager) IsInstalled(pkg string) bool {
	return pm.installed[pkg]
}

func (pm *fakePackageManager) Heal() error {
	pm.healCalls++
	return nil
}

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

	pm := &fakePackageManager{}
	result, err := applySuit(tempDir, suit, true, false, pm)
	if err != nil {
		t.Fatalf("applySuit in dry-run mode failed: %v", err)
	}

	if len(result.Failed) != 0 {
		t.Errorf("expected 0 failed packages in dry-run, got %d: %v", len(result.Failed), result.Failed)
	}

	// In dry-run, all packages (packages + packages_no_recommends + packages_interactive) are simulated as installed
	expectedTotal := len(suit.Packages) + len(suit.PackagesNoRecommends) + len(suit.PackagesInteractive)
	if len(result.Installed) != expectedTotal {
		t.Errorf("expected %d installed packages in dry-run, got %d: %v", expectedTotal, len(result.Installed), result.Installed)
	}
	if len(pm.installCalls) != 0 || pm.refreshCalls != 0 {
		t.Error("expected dry-run not to invoke the package manager")
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

	result, err := applySuit(tempDir, suit, true, true, &fakePackageManager{})
	if err != nil {
		t.Fatalf("applySuit in accessory dry-run mode failed: %v", err)
	}

	if len(result.Failed) != 0 {
		t.Errorf("expected 0 failed packages, got %d: %v", len(result.Failed), result.Failed)
	}
	if len(result.Installed) != 6 {
		t.Errorf("expected 6 installed packages for accessory, got %d: %v", len(result.Installed), result.Installed)
	}
}

func TestApplySuitUsesPackageManagerModes(t *testing.T) {
	tempDir := t.TempDir()
	costumeYaml := `name: package-modes
packages:
  - pkg-normal
sequence:
  packages_no_install_recommends:
    - pkg-no-recommends
  packages_interactive:
    - pkg-interactive
`
	yamlPath := filepath.Join(tempDir, "index.yaml")
	if err := os.WriteFile(yamlPath, []byte(costumeYaml), 0644); err != nil {
		t.Fatalf("failed to write fixture yaml: %v", err)
	}

	suit, err := loadSuit(yamlPath)
	if err != nil {
		t.Fatalf("loadSuit failed: %v", err)
	}

	pm := &fakePackageManager{}
	if _, err := applySuit(tempDir, suit, false, false, pm); err != nil {
		t.Fatalf("applySuit failed: %v", err)
	}

	if len(pm.installCalls) != 3 {
		t.Fatalf("expected 3 package manager install calls, got %d", len(pm.installCalls))
	}
	if pm.installCalls[0] != (InstallMode{Retries: 3}) {
		t.Errorf("unexpected normal install mode: %+v", pm.installCalls[0])
	}
	if pm.installCalls[1] != (InstallMode{NoRecommends: true, Retries: 3}) {
		t.Errorf("unexpected no-recommends install mode: %+v", pm.installCalls[1])
	}
	if pm.installCalls[2] != (InstallMode{Interactive: true}) {
		t.Errorf("unexpected interactive install mode: %+v", pm.installCalls[2])
	}
}

func TestApplySuitPreservesPackageInstallOutcomes(t *testing.T) {
	suit := &Suit{Packages: []string{"installed-package", "missing-package", "failed-package"}}
	pm := &fakePackageManager{result: PackageInstallResult{
		Installed:   []string{"installed-package"},
		Unavailable: []string{"missing-package"},
		Failed:      []string{"failed-package"},
	}}

	result, err := applySuit(t.TempDir(), suit, false, false, pm)
	if err != nil {
		t.Fatalf("applySuit() error = %v", err)
	}
	if got, want := result.Installed, []string{"installed-package"}; !slices.Equal(got, want) {
		t.Errorf("Installed = %v, want %v", got, want)
	}
	if got, want := result.Unavailable, []string{"missing-package"}; !slices.Equal(got, want) {
		t.Errorf("Unavailable = %v, want %v", got, want)
	}
	if got, want := result.Failed, []string{"failed-package"}; !slices.Equal(got, want) {
		t.Errorf("Failed = %v, want %v", got, want)
	}
}

func TestApplySuitUsesPackageManagerRepositoryOperations(t *testing.T) {
	tests := []struct {
		name         string
		repositories Repositories
		refreshCalls int
		upgradeCalls []bool
	}{
		{
			name:         "update only",
			repositories: Repositories{Update: true},
			refreshCalls: 1,
		},
		{
			name:         "upgrade only",
			repositories: Repositories{Upgrade: true},
			upgradeCalls: []bool{false},
		},
		{
			name:         "update and upgrade",
			repositories: Repositories{Update: true, Upgrade: true},
			upgradeCalls: []bool{true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suit := &Suit{Sequence: &Sequence{Repositories: &tt.repositories}}
			pm := &fakePackageManager{}

			if _, err := applySuit(t.TempDir(), suit, false, false, pm); err != nil {
				t.Fatalf("applySuit failed: %v", err)
			}
			if pm.refreshCalls != tt.refreshCalls {
				t.Errorf("expected %d refresh calls, got %d", tt.refreshCalls, pm.refreshCalls)
			}
			if len(pm.upgradeCalls) != len(tt.upgradeCalls) {
				t.Fatalf("expected upgrade calls %v, got %v", tt.upgradeCalls, pm.upgradeCalls)
			}
			for i, refresh := range tt.upgradeCalls {
				if pm.upgradeCalls[i] != refresh {
					t.Errorf("upgrade call %d refresh=%t, want %t", i, pm.upgradeCalls[i], refresh)
				}
			}
		})
	}
}

func TestHealAndRetryFailedUsesPackageManager(t *testing.T) {
	pm := &fakePackageManager{installed: map[string]bool{"failed-package": true}}
	remaining := healAndRetryFailed([]string{"failed-package"}, pm)

	if pm.healCalls != 1 {
		t.Errorf("expected one heal call, got %d", pm.healCalls)
	}
	if len(pm.installCalls) != 1 || pm.installCalls[0] != (InstallMode{Retries: 1}) {
		t.Errorf("unexpected retry install calls: %+v", pm.installCalls)
	}
	if len(remaining) != 0 {
		t.Errorf("expected no remaining failed packages, got %v", remaining)
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
