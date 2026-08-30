package tailor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pieroproietti/penguins-tailor/pkg/utils"
)

func Show(costumeName string) error {
	v2Dir, err := getWardrobeV2Dir()
	if err != nil {
		return err
	}

	costumeDir := filepath.Join(v2Dir, "costumes", costumeName)
	if _, err := os.Stat(costumeDir); os.IsNotExist(err) {
		if strings.HasPrefix(costumeName, "accessories/") || strings.HasPrefix(costumeName, "costumes/") {
			costumeDir = filepath.Join(v2Dir, costumeName)
		} else {
			accDir := filepath.Join(v2Dir, "accessories", costumeName)
			if _, errAcc := os.Stat(accDir); errAcc == nil {
				costumeDir = accDir
			}
		}
	}

	yamlPath := findYaml(costumeDir)
	if yamlPath == "" {
		return fmt.Errorf("costume '%s' not found in %s", costumeName, v2Dir)
	}

	suit, err := loadSuit(yamlPath)
	if err != nil {
		return err
	}

	isAcc := strings.Contains(costumeDir, "/accessories/")
	icon := "👗"
	titleType := "COSTUME"
	if isAcc {
		icon = "👝"
		titleType = "ACCESSORY"
	}

	versionStr := ""
	if suit.Release != "" {
		versionStr = fmt.Sprintf(" (v%s)", suit.Release)
	}

	origin := GetWardrobeOrigin()
	bannerTitle := fmt.Sprintf("%s: %s%s", titleType, suit.Name, versionStr)
	if origin != "" {
		bannerTitle = fmt.Sprintf("%s - atelier: %s", bannerTitle, origin)
	}

	utils.PrintBanner(icon, bannerTitle, suit.Description)
	if origin != "" {
		fmt.Printf("  %-16s: %s\n", "Atelier", origin)
	}
	if suit.Author != "" {
		fmt.Printf("  %-16s: %s\n", "Author", suit.Author)
	}
	if len(suit.Distributions) > 0 {
		fmt.Printf("  %-16s: %s\n", "Distributions", strings.Join(suit.Distributions, ", "))
	}
	if len(suit.Accessories) > 0 {
		fmt.Printf("  %-16s: %s\n", "Accessories", strings.Join(suit.Accessories, ", "))
	}
	if len(suit.Packages) > 0 {
		limit := 5
		if len(suit.Packages) < limit {
			limit = len(suit.Packages)
		}
		preview := strings.Join(suit.Packages[:limit], ", ")
		if len(suit.Packages) > limit {
			preview += "..."
		}
		fmt.Printf("  %-16s: %d packages (%s)\n", "Packages", len(suit.Packages), preview)
	}
	if len(suit.SequenceCmds) > 0 {
		fmt.Printf("  %-16s: %d sequence commands\n", "Sequence Cmds", len(suit.SequenceCmds))
	}
	if len(suit.FinalizeCmds) > 0 {
		fmt.Printf("  %-16s: %d finalization commands\n", "Finalize Cmds", len(suit.FinalizeCmds))
	}
	fmt.Println()
	return nil
}
