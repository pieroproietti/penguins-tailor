//go:build linux

package tailor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestSysrootRsyncCmdForcesRootOwnership(t *testing.T) {
	cmd := sysrootRsyncCmd("/home/user/.wardrobe/v2/costumes/x/sysroot", "/")

	for _, want := range []string{
		"--no-o",
		"--no-g",
		"--chown=root:root",
		"--chmod=D755",
		" /home/user/.wardrobe/v2/costumes/x/sysroot/ /",
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("command %q is missing %q", cmd, want)
		}
	}
}

// TestSysrootRsyncAppliesRootOwnership applies an overlay that looks like the
// checkout of a regular (non-root) user, with a foreign owner and odd modes,
// and verifies that neither that owner nor those modes leak into the target.
// Changing ownership requires root, so the test is skipped otherwise.
func TestSysrootRsyncAppliesRootOwnership(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("requires root to exercise ownership handling")
	}
	if _, err := exec.LookPath("rsync"); err != nil {
		t.Skip("rsync not installed")
	}

	const foreign = 4242
	src := t.TempDir()
	dst := t.TempDir()

	bin := filepath.Join(src, "usr", "local", "bin")
	skel := filepath.Join(src, "etc", "skel")
	tool := filepath.Join(bin, "tool")
	conf := filepath.Join(skel, "conf")

	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(skel, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tool, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(conf, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Make the overlay look like a checkout owned by another user, with
	// group-writable files and directories without access for "others".
	odd := map[string]os.FileMode{
		src:                                0o2770,
		filepath.Join(src, "usr"):          0o770,
		filepath.Join(src, "usr", "local"): 0o770,
		bin:                                0o770,
		filepath.Join(src, "etc"):          0o770,
		skel:                               0o770,
		tool:                               0o770, // executable
		conf:                               0o660, // not executable
	}
	for p, mode := range odd {
		if err := os.Chmod(p, mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Lchown(p, foreign, foreign); err != nil {
			t.Fatal(err)
		}
	}

	if out, err := exec.Command("sh", "-c", sysrootRsyncCmd(src, dst)).CombinedOutput(); err != nil {
		t.Fatalf("rsync failed: %v\n%s", err, out)
	}

	want := map[string]os.FileMode{
		dst:                                0o755,
		filepath.Join(dst, "usr"):          0o755,
		filepath.Join(dst, "usr", "local"): 0o755,
		filepath.Join(dst, "usr", "local", "bin"):         0o755,
		filepath.Join(dst, "etc", "skel"):                 0o755,
		filepath.Join(dst, "usr", "local", "bin", "tool"): 0o755, // executable bit kept
		filepath.Join(dst, "etc", "skel", "conf"):         0o644,
	}
	for path, mode := range want {
		info, err := os.Lstat(path)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		st := info.Sys().(*syscall.Stat_t)
		if st.Uid != 0 || st.Gid != 0 {
			t.Errorf("%s: owner %d:%d, want 0:0", path, st.Uid, st.Gid)
		}
		if info.Mode().Perm() != mode {
			t.Errorf("%s: mode %o, want %o", path, info.Mode().Perm(), mode)
		}
	}
}
