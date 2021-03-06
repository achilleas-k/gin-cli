package gincmd

import (
	"fmt"
	"os"

	ginclient "github.com/G-Node/gin-cli/ginclient"
	"github.com/G-Node/gin-cli/git"
	"github.com/spf13/cobra"
)

func gitrun(cmd *cobra.Command, args []string) {
	// TODO: Use all available keys?
	gincl := ginclient.New("")
	_ = gincl.LoadToken() // OK to run without token
	gitcmd := git.Command(args...)
	err := gitcmd.Start()
	CheckError(err)
	var line string
	var rerr error
	for rerr = nil; rerr == nil; line, rerr = gitcmd.OutReader.ReadString('\n') {
		fmt.Print(line)
	}
	for rerr = nil; rerr == nil; line, rerr = gitcmd.ErrReader.ReadString('\n') {
		fmt.Fprint(os.Stderr, line)
	}
	if gitcmd.Wait() != nil {
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
