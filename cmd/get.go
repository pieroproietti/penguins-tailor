package cmd

import (
	"github.com/pieroproietti/penguins-tailor/pkg/tailor"
	"github.com/spf13/cobra"
)

func getCmd() *cobra.Command {
	var repoURL string

	cmd := &cobra.Command{
		Use:   "get [url]",
		Short: "Download or update the costumes repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := repoURL
			if len(args) > 0 {
				url = args[0]
			}
			return tailor.Get(url)
		},
	}

	cmd.Flags().StringVarP(&repoURL, "url", "u", "", "URL of the costumes repository")

	return cmd
}
