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

	for _, removed := range []string{"no-acc", "no-firm"} {
		if cmd.Flags().Lookup(removed) != nil {
			t.Errorf("expected removed --%s flag not to exist", removed)
		}
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
