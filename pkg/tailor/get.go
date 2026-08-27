package tailor

import (
	"fmt"
	"os"

	"github.com/pieroproietti/penguins-tailor/pkg/utils"
)

const defaultRepoURL = "https://github.com/pieroproietti/penguins-wardrobe"

func Get(repoURL string) error {
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

	if _, err := os.Stat(root); os.IsNotExist(err) {
		utils.LogNormal("Downloading costumes repository from %s...", targetURL)
		cmd := fmt.Sprintf("git clone %s %s", targetURL, root)
		if err := utils.Exec(cmd); err != nil {
			return err
		}
		fixWardrobeOwnership(root)
		return nil
	}

	currentOrigin := getGitOrigin(root)
	if currentOrigin != "" && normalizeGitURL(currentOrigin) == normalizeGitURL(targetURL) {
		utils.LogNormal("Costumes repository already present in %s (atelier: %s). Updating...", root, currentOrigin)
		cmd := fmt.Sprintf("git -C %s pull", root)
		if err := utils.Exec(cmd); err != nil {
			return err
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

	utils.LogNormal("Downloading costumes repository from %s...", targetURL)
	cmd := fmt.Sprintf("git clone %s %s", targetURL, root)
	if err := utils.Exec(cmd); err != nil {
		return err
	}
	fixWardrobeOwnership(root)
	return nil
}

func fixWardrobeOwnership(root string) {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		_ = utils.Exec(fmt.Sprintf("chown -R %s:%s %s", sudoUser, sudoUser, root))
	}
}
