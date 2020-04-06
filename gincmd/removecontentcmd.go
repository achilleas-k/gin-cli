package gincmd

import (
	"fmt"

	"github.com/G-Node/gin-cli/ginclient"
	"github.com/G-Node/gin-cli/ginclient/config"
	"github.com/G-Node/gin-cli/gincmd/ginerrors"
	"github.com/spf13/cobra"
)

func remove(cmd *cobra.Command, args []string) {
	prStyle := determinePrintStyle(cmd)
	// TODO: Need server config? just use remotes (and all keys)
	conf := config.Read()
	gincl := ginclient.New(conf.DefaultServer)
	requirelogin(cmd, gincl, prStyle != psJSON)
	switch ginclient.Checkwd() {
	case ginclient.NotRepository:
		Die(ginerrors.NotInRepo)
	case ginclient.NotAnnex:
		Warn(ginerrors.MissingAnnex)
	case ginclient.UpgradeRequired:
		annexVersionNotice()
	}
	nitems := 0
	annexedFiles, err := gincl.ListAnnexedFiles(args...)
	if err == nil {
		nitems = len(annexedFiles)
	}
	if prStyle == psProgress {
		fmt.Println(":: Removing file content")
	}
	rmchan := gincl.RemoveContent(args)
	formatOutput(rmchan, prStyle, nitems)
}

// RemoveContentCmd sets up the 'remove-content' subcommand
func RemoveContentCmd() *cobra.Command {
	description := "Remove the content of local files. This command will not remove the content of files that have not been already uploaded to a remote repository, even if the user specifies such files explicitly. Removed content can be retrieved from the server by using the 'get-content' command. With no arguments, removes the content of all files under the current working directory, as long as they have been safely uploaded to a remote repository.\n\nNote that after removal, placeholder files will remain in the local repository. These files appear as 'No Content' when running the 'gin ls' command."
	args := map[string]string{
		"<filenames>": "One or more directories or files to remove.",
	}
	var cmd = &cobra.Command{
		// Use:                   "remove-content [--json | --verbose] [<filenames>]...",
		Use:                   "remove-content [--json] [<filenames>]...",
		Short:                 "Remove the content of local files that have already been uploaded",
		Long:                  formatdesc(description, args),
		Args:                  cobra.ArbitraryArgs,
		Run:                   remove,
		Aliases:               []string{"rmc"},
		DisableFlagsInUseLine: true,
	}
	cmd.Flags().Bool("json", false, jsonHelpMsg)
	// cmd.Flags().Bool("verbose", false, verboseHelpMsg)
	return cmd
}
