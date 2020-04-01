package gincmd

import (
	"github.com/G-Node/gin-cli/ginclient"
	"github.com/spf13/cobra"
)

func gitrun(cmd *cobra.Command, args []string) {
	// TODO: Use all available keys?
	printPasstrough(ginclient.GitCommand(args...))

}

// GitCmd sets up the 'git' passthrough subcommand
func GitCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:                   "git <cmd> [<args>]...",
		Short:                 "Run a 'git' command through the gin client",
		Long:                  "",
		Args:                  cobra.ArbitraryArgs,
		Run:                   gitrun,
		DisableFlagsInUseLine: true,
		Hidden:                true,
		DisableFlagParsing:    true,
	}
	return cmd
}
