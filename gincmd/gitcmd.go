package gincmd

import (
	"fmt"
	"os"

	"github.com/G-Node/gin-cli/ginclient"
	"github.com/spf13/cobra"
)

func gitrun(cmd *cobra.Command, args []string) {
	// TODO: Use all available keys?
	stdout, stderr, errc := ginclient.GitCommand(args...)

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
