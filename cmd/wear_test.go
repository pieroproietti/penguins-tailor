package cmd

import (
	"testing"
)

func TestWearCmdFlags(t *testing.T) {
	cmd := wearCmd()

	branchFlag := cmd.Flags().Lookup("branch")
	if branchFlag == nil {
		t.Fatal("expected --branch flag to exist")
	}
	if branchFlag.Shorthand != "b" {
		t.Errorf("expected shorthand 'b' for --branch, got %q", branchFlag.Shorthand)
	}

	noAccFlag := cmd.Flags().Lookup("no-acc")
	if noAccFlag == nil {
		t.Fatal("expected --no-acc flag to exist")
	}

	noFirmFlag := cmd.Flags().Lookup("no-firm")
	if noFirmFlag == nil {
		t.Fatal("expected --no-firm flag to exist")
	}

	linearFlag := cmd.Flags().Lookup("linear")
	if linearFlag == nil {
		t.Fatal("expected --linear flag to exist")
	}

	dryRunFlag := cmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Fatal("expected --dry-run flag to exist")
	}
	if dryRunFlag.Shorthand != "n" {
		t.Errorf("expected shorthand 'n' for --dry-run, got %q", dryRunFlag.Shorthand)
	}

	simulateFlag := cmd.Flags().Lookup("simulate")
	if simulateFlag == nil {
		t.Fatal("expected --simulate flag to exist")
	}
}
