package gincmd

import (
	"fmt"

	"github.com/G-Node/gin-cli/ginclient"
	"github.com/G-Node/gin-cli/ginclient/config"
	"github.com/G-Node/gin-cli/gincmd/ginerrors"
	"github.com/spf13/cobra"
)

func unlock(cmd *cobra.Command, args []string) {
	prStyle := determinePrintStyle(cmd)
	switch ginclient.Checkwd() {
	case ginclient.NotRepository:
		Die(ginerrors.NotInRepo)
	case ginclient.NotAnnex:
		Warn(ginerrors.MissingAnnex)
	case ginclient.UpgradeRequired:
		annexVersionNotice()
	}

	if prStyle != psJSON {
		fmt.Println(":: Unlocking files")
	}
	// TODO: Probably doesn't need a server config
	conf := config.Read()
	defserver := conf.DefaultServer
	gincl := ginclient.New(defserver)
	lockedFiles, err := gincl.ListLockedFiles(args...)
	CheckError(err)
	unlockchan := gincl.UnlockContent(args)
	formatOutput(unlockchan, prStyle, len(lockedFiles))
}

// UnlockCmd sets up the file 'unlock' subcommand
func UnlockCmd() *cobra.Command {
	description := "Unlock one or more files to allow editing. This changes the type of the file in the repository. A 'commit' command is required to save the change. Unmodified unlocked files that have not yet been committed are marked as 'Lock status changed' (short TC) in the output of the 'ls' command.\n\nUnlocking a file takes longer depending on its size."
	args := map[string]string{
		"<filenames>": "One or more directories or files to unlock.",
	}
	var cmd = &cobra.Command{
		// Use:                   "unlock [--json | --verbose] <filenames>...",
		Use:                   "unlock [--json] <filenames>...",
		Short:                 "Unlock files for editing",
		Long:                  formatdesc(description, args),
		Args:                  cobra.MinimumNArgs(1),
		Run:                   unlock,
		DisableFlagsInUseLine: true,
	}
	cmd.Flags().Bool("json", false, jsonHelpMsg)
	// cmd.Flags().Bool("verbose", false, verboseHelpMsg)
	return cmd
}
