package tailor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pieroproietti/penguins-tailor/pkg/utils"
)

var brandingRoot = "/etc/penguins-eggs.d/branding"

func validateBranding(v2Dir, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if filepath.Base(name) != name || name == "." || name == ".." {
		return fmt.Errorf("invalid branding name %q", name)
	}
	source := filepath.Join(v2Dir, "branding", name)
	info, err := os.Stat(source)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("branding %q not found in %s", name, source)
		}
		return fmt.Errorf("unable to inspect branding %q: %w", name, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("branding %q is not a directory: %s", name, source)
	}
	return nil
}

// applyBranding reconciles the branding selected by a costume. A missing
// branding property means that no branding should remain active.
func applyBranding(v2Dir, name string, dryRun bool) error {
	name = strings.TrimSpace(name)
	if name == "" {
		if dryRun {
			logToFile(fmt.Sprintf("[DRY-RUN] Would remove active branding from %s", brandingRoot))
			return nil
		}
		if err := os.RemoveAll(brandingRoot); err != nil {
			return fmt.Errorf("unable to remove active branding %s: %w", brandingRoot, err)
		}
		return nil
	}

	if err := validateBranding(v2Dir, name); err != nil {
		return err
	}
	source := filepath.Join(v2Dir, "branding", name)

	if dryRun {
		logToFile(fmt.Sprintf("[DRY-RUN] Would replace %s with branding %s from %s", brandingRoot, name, source))
		return nil
	}

	parent := filepath.Dir(brandingRoot)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return fmt.Errorf("unable to create branding parent %s: %w", parent, err)
	}
	tmpRoot, err := os.MkdirTemp(parent, ".branding-")
	if err != nil {
		return fmt.Errorf("unable to prepare branding: %w", err)
	}
	defer os.RemoveAll(tmpRoot)
	if err := os.Chmod(tmpRoot, 0755); err != nil {
		return fmt.Errorf("unable to set branding permissions: %w", err)
	}

	if err := copyBrandingTree(source, tmpRoot); err != nil {
		return fmt.Errorf("unable to copy branding %q: %w", name, err)
	}
	if err := os.RemoveAll(brandingRoot); err != nil {
		return fmt.Errorf("unable to replace active branding %s: %w", brandingRoot, err)
	}
	if err := os.Rename(tmpRoot, brandingRoot); err != nil {
		return fmt.Errorf("unable to activate branding %q: %w", name, err)
	}

	utils.LogSuccess("Branding '%s' installed in %s", name, brandingRoot)
	return nil
}

func copyBrandingTree(source, target string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, rel)

		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, destination)
		}
		if info.IsDir() {
			return os.MkdirAll(destination, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported branding file type: %s", path)
		}

		return copyBrandingFile(path, destination, info.Mode().Perm())
	})
}

func copyBrandingFile(source, target string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		in.Close()
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeOutErr := out.Close()
	closeInErr := in.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeOutErr != nil {
		return closeOutErr
	}
	return closeInErr
}
