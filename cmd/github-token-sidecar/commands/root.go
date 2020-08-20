package commands

import (
		"fmt"
		"github.com/spf13/cobra"
		"github.com/google/go-github/v32/github"
)

func NewGitHubTokenCommand() *cobra.Command {
		return &cobra.Command{
				Use:   "version",
				Short: "Print the version number of Hugo",
				Long:  `All software has versions. This is Hugo's`,
				Run: func(cmd *cobra.Command, args []string) {
						github
				},
		}
}