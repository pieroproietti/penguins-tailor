package tailor

import (
	"fmt"
	"os"
	"strings"

	"github.com/pieroproietti/penguins-tailor/pkg/utils"
)

const defaultRepoURL = "https://github.com/pieroproietti/penguins-wardrobe"

func Get(repoURL string, branch string) error {
	root, err := getWardrobeRoot()
	if err != nil {
		return err
	}

	targetURL := repoURL
	if targetURL == "" {
		currentOrigin := getGitOrigin(root)
		if currentOrigin != "" {
			targetURL = currentOrigin
		} else {
			targetURL = defaultRepoURL
		}
	}

	if idx := strings.Index(targetURL, "#"); idx != -1 {
		if branch == "" {
			branch = targetURL[idx+1:]
		}
		targetURL = targetURL[:idx]
	}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		if branch != "" {
			utils.LogNormal("Downloading costumes repository from %s (branch: %s)...", targetURL, branch)
			cmd := fmt.Sprintf("git clone -b %s %s %s", branch, targetURL, root)
			if err := utils.Exec(cmd); err != nil {
				return err
			}
		} else {
			utils.LogNormal("Downloading costumes repository from %s...", targetURL)
			cmd := fmt.Sprintf("git clone %s %s", targetURL, root)
			if err := utils.Exec(cmd); err != nil {
				return err
			}
		}
		fixWardrobeOwnership(root)
		return nil
	}

	currentOrigin := getGitOrigin(root)
	if currentOrigin != "" && normalizeGitURL(currentOrigin) == normalizeGitURL(targetURL) {
		if branch != "" {
			utils.LogNormal("Costumes repository already present in %s (atelier: %s). Updating branch %s...", root, currentOrigin, branch)
			cmd := fmt.Sprintf("git -C %s fetch && git -C %s checkout %s && git -C %s pull", root, root, branch, root)
			if err := utils.Exec(cmd); err != nil {
				return err
			}
		} else {
			utils.LogNormal("Costumes repository already present in %s (atelier: %s). Updating...", root, currentOrigin)
			cmd := fmt.Sprintf("git -C %s pull", root)
			if err := utils.Exec(cmd); err != nil {
				return err
			}
		}
		fixWardrobeOwnership(root)
		return nil
	}

	if currentOrigin != "" {
		utils.LogNormal("Existing wardrobe in %s is from %s, switching to %s...", root, currentOrigin, targetURL)
	} else {
		utils.LogNormal("Existing directory in %s is not a valid wardrobe repository, replacing with %s...", root, targetURL)
	}

	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("failed to remove existing wardrobe directory %s: %w", root, err)
	}

	if branch != "" {
		utils.LogNormal("Downloading costumes repository from %s (branch: %s)...", targetURL, branch)
		cmd := fmt.Sprintf("git clone -b %s %s %s", branch, targetURL, root)
		if err := utils.Exec(cmd); err != nil {
			return err
		}
	} else {
		utils.LogNormal("Downloading costumes repository from %s...", targetURL)
		cmd := fmt.Sprintf("git clone %s %s", targetURL, root)
		if err := utils.Exec(cmd); err != nil {
			return err
		}
	}
	fixWardrobeOwnership(root)
	return nil
}

func fixWardrobeOwnership(root string) {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		_ = utils.Exec(fmt.Sprintf("chown -R %s:%s %s", sudoUser, sudoUser, root))
	}
}
