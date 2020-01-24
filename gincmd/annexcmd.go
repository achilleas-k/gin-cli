package gincmd

import (
	"fmt"
	"os"

	"github.com/G-Node/gin-cli/ginclient"
	"github.com/spf13/cobra"
)

func annexrun(cmd *cobra.Command, args []string) {
	stdout, stderr, errc := ginclient.AnnexCommand(args...)

	for line, rerr := stdout.ReadString('\n'); rerr == nil; line, rerr = stdout.ReadString('\n') {
		fmt.Print(line)
	}
	for line, rerr := stderr.ReadString('\n'); rerr == nil; line, rerr = stderr.ReadString('\n') {
		fmt.Fprint(os.Stderr, line)
	}
	if <-errc != nil {
		os.Exit(1)
	}
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
