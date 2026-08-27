package tailor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pieroproietti/penguins-tailor/pkg/utils"
)

func List() error {
	v2Dir, err := getWardrobeV2Dir()
	if err != nil {
		return err
	}

	costumesDir := filepath.Join(v2Dir, "costumes")
	entries, err := os.ReadDir(costumesDir)
	if err != nil {
		return fmt.Errorf("unable to read costumes: %v (have you run 'tailor get'?)", err)
	}

	origin := GetWardrobeOrigin()
	subtitle := "Costumes and desktop recipes available in wardrobe"
	if origin != "" {
		subtitle = fmt.Sprintf("Costumes and recipes from atelier: %s", origin)
	}
	utils.PrintBanner("👗", "AVAILABLE COSTUMES", subtitle)
	for _, entry := range entries {
		if entry.IsDir() {
			yamlPath := findYaml(filepath.Join(costumesDir, entry.Name()))
			if yamlPath != "" {
				if suit, err := loadSuit(yamlPath); err == nil {
					fmt.Printf("  • %s%-12s%s : %s\n", utils.ColorBold+utils.ColorYellow, entry.Name(), utils.ColorReset, suit.Description)
				}
			}
		}
	}
	fmt.Println()
	return nil
}
