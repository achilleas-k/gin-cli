package gincmd

import (
	"fmt"
	"os"

	"github.com/G-Node/gin-cli/ginclient"
	"github.com/G-Node/gin-cli/ginclient/log"
	"github.com/G-Node/gin-cli/gincmd/ginerrors"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func commit(cmd *cobra.Command, args []string) {
	prStyle := determinePrintStyle(cmd)
	switch ginclient.Checkwd() {
	case ginclient.NotRepository:
		Die(ginerrors.NotInRepo)
	case ginclient.NotAnnex:
		Warn(ginerrors.MissingAnnex)
	case ginclient.UpgradeRequired:
		annexVersionNotice()
	}

	commitmsg, _ := cmd.Flags().GetString("message")

	paths := args
	if len(paths) > 0 {
		if prStyle == psDefault {
			fmt.Println(":: Adding file changes")
		}
		addchan := ginclient.Add(paths)
		formatOutput(addchan, prStyle, 0)
	}

	if prStyle == psDefault {
		fmt.Print(":: Recording changes ")
	}
	if commitmsg == "" {
		commitmsg = makeCommitMessage("commit", paths)
	}
	gincl := ginclient.New("")
	err := gincl.Commit(commitmsg)
	var stat string
	if err != nil {
		if err.Error() == "Nothing to commit" {
			stat = "\n   No changes recorded"
		} else {
			Die(err)
		}
	} else {
		stat = green("OK")
	}
	if prStyle == psDefault {
		fmt.Fprintln(color.Output, stat)
	}
}

func makeCommitMessage(action string, paths []string) (commitmsg string) {
	// add header commit line
	hostname, err := os.Hostname()
	if err != nil {
		log.Write("Could not retrieve hostname")
		hostname = unknownhostname
	}
	gincl := ginclient.New("")
	changes, err := gincl.DescribeIndexShort(paths)
	if err != nil {
		log.Write("Failed to determine changes for commit message")
		changes = ""
	}
	commitmsg = fmt.Sprintf("gin %s from %s\n\n%s", action, hostname, changes)
	return
}

// CommitCmd sets up the 'commit' subcommand
func CommitCmd() *cobra.Command {
	description := "Record changes made in a local repository. This command must be called from within the local repository clone. Specific files or directories may be specified. All changes made to the files and directories that are specified will be recorded, including addition of new files, modifications and renaming of existing files, and file deletions.\n\nIf no arguments are specified, no changes are recorded."
	args := map[string]string{"<filenames>": "One or more directories or files to commit."}
	var cmd = &cobra.Command{
		// Use:                   "commit [--json | --verbose] [--message message] [<filenames>]...",
		Use:                   "commit [--json] [--message message] [<filenames>]...",
		Short:                 "Record changes in local repository",
		Long:                  formatdesc(description, args),
		Args:                  cobra.ArbitraryArgs,
		Run:                   commit,
		DisableFlagsInUseLine: true,
	}
	cmd.Flags().Bool("json", false, jsonHelpMsg)
	cmd.Flags().StringP("message", "m", "", "Commit message")
	// cmd.Flags().Bool("verbose", false, verboseHelpMsg)
	return cmd
}
