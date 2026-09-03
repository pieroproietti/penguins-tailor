package cmd

import (
	"github.com/pieroproietti/penguins-tailor/pkg/tailor"
	"github.com/spf13/cobra"
)

func wearCmd() *cobra.Command {
	var linear bool
	var branch string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "wear [costume]",
		Short: "Wear a costume from the wardrobe",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return tailor.Wear(args[0], linear, branch, dryRun)
		},
	}

	cmd.Flags().StringVarP(&branch, "branch", "b", "", "Branch of the costumes repository")
	cmd.Flags().BoolVar(&linear, "linear", false, "Use linear standard output without split screen TUI")
	cmd.Flags().BoolVar(&linear, "no-split", false, "Alias for --linear")
	_ = cmd.Flags().MarkHidden("no-split")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Simulate costume installation without making changes")
	cmd.Flags().BoolVar(&dryRun, "simulate", false, "Alias for --dry-run")
	_ = cmd.Flags().MarkHidden("simulate")

	return cmd
}
