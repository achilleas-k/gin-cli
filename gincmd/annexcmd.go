package gincmd

import (
	"github.com/G-Node/gin-cli/ginclient"
	"github.com/spf13/cobra"
)

func annexrun(cmd *cobra.Command, args []string) {
	printPasstrough(ginclient.AnnexCommand(args...))
}

// AnnexCmd sets up the 'annex' passthrough subcommand
func AnnexCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:                   "annex <cmd> [<args>]...",
		Short:                 "Run a 'git annex' command through the gin client",
		Long:                  "",
		Args:                  cobra.ArbitraryArgs,
		Run:                   annexrun,
		DisableFlagsInUseLine: true,
		Hidden:                true,
		DisableFlagParsing:    true,
	}
	return cmd
}
