package cmd

import (
	"github.com/pieroproietti/penguins-tailor/pkg/tailor"
	"github.com/spf13/cobra"
)

func wearCmd() *cobra.Command {
	var noAcc bool
	var noFirm bool
	var linear bool

	cmd := &cobra.Command{
		Use:   "wear [costume]",
		Short: "Wear a costume from the wardrobe",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return tailor.Wear(args[0], noAcc, noFirm, linear)
		},
	}

	cmd.Flags().BoolVar(&noAcc, "no-acc", false, "Do not install accessories")
	cmd.Flags().BoolVar(&noFirm, "no-firm", false, "Do not install firmware")
	cmd.Flags().BoolVar(&linear, "linear", false, "Use linear standard output without split screen TUI")
	cmd.Flags().BoolVar(&linear, "no-split", false, "Alias for --linear")
	_ = cmd.Flags().MarkHidden("no-split")

	return cmd
}
