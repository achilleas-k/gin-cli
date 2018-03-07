package gincmd

import (
	"encoding/json"
	"fmt"
	"strings"

	ginclient "github.com/G-Node/gin-cli/gin-client"
	"github.com/G-Node/gin-cli/util"
	"github.com/bbrks/wrap"
	"github.com/docker/docker/pkg/term"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var green = color.New(color.FgGreen).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()

// requirelogin prompts for login if the user is not already logged in.
// It only checks if a local token exists and does not confirm its validity with the server.
// The function should be called at the start of any command that requires being logged in to run.
func requirelogin(cmd *cobra.Command, gincl *ginclient.Client, prompt bool) {
	err := gincl.LoadToken()
	if prompt {
		if err != nil {
			login(cmd, nil)
		}
		err = gincl.LoadToken()
	}
	util.CheckError(err)
}

func usageDie(cmd *cobra.Command) {
	cmd.Help()
	// exit without message
	util.Die("")
}

func printProgress(statuschan <-chan ginclient.RepoFileStatus, jsonout bool) {
	var fname, state string
	var lastprint string
	filesuccess := make(map[string]bool)
	for stat := range statuschan {
		var msgparts []string
		if jsonout {
			j, _ := json.Marshal(stat)
			fmt.Println(string(j))
			filesuccess[stat.FileName] = true
			if stat.Err != nil {
				filesuccess[stat.FileName] = false
			}
			continue
		}
		if stat.FileName != fname || stat.State != state {
			// New line if new file or new state
			if len(lastprint) > 0 {
				fmt.Println()
			}
			lastprint = ""
			fname = stat.FileName
			state = stat.State
		}
		msgparts = append(msgparts, stat.State, stat.FileName)
		if stat.Err == nil {
			if stat.Progress == "100%" {
				msgparts = append(msgparts, green("OK"))
				filesuccess[stat.FileName] = true
			} else {
				msgparts = append(msgparts, stat.Progress, stat.Rate)
			}
		} else {
			msgparts = append(msgparts, red(stat.Err.Error()))
			filesuccess[stat.FileName] = false
		}
		newprint := fmt.Sprintf("\r%s", util.CleanSpaces(strings.Join(msgparts, " ")))
		if newprint != lastprint {
			fmt.Printf("\r%s", strings.Repeat(" ", len(lastprint))) // clear the line
			fmt.Fprint(color.Output, newprint)
			lastprint = newprint
		}
	}
	if len(lastprint) > 0 {
		fmt.Println()
	}

	// count unique file errors
	nerrors := 0
	for _, stat := range filesuccess {
		if !stat {
			nerrors++
		}
	}
	if nerrors > 0 {
		// Exit with error message and failed exit status
		var plural string
		if nerrors > 1 {
			plural = "s"
		}
		util.Die(fmt.Sprintf("%d operation%s failed", nerrors, plural))
	}
}

var wouter = wrap.NewWrapper()
var winner = wrap.NewWrapper()

func termwidth() int {
	width := 80
	if ws, err := term.GetWinsize(0); err == nil {
		width = int(ws.Width)
	}
	return width - 1
}

func formatdesc(desc string, args map[string]string) (fdescription string) {
	width := termwidth()
	wouter.OutputLinePrefix = "  "
	winner.OutputLinePrefix = "    "

	if len(desc) > 0 {
		fdescription = fmt.Sprintf("Description:\n\n%s", wouter.Wrap(desc, width))
	}

	if args != nil {
		argsdesc := fmt.Sprintf("Arguments:\n\n")
		for a, d := range args {
			argsdesc = fmt.Sprintf("%s%s%s\n", argsdesc, wouter.Wrap(a, width), winner.Wrap(d, width))
		}
		fdescription = fmt.Sprintf("%s\n%s", fdescription, argsdesc)
	}
	return
}

func formatexamples(examples map[string]string) (exdesc string) {
	width := termwidth()
	if examples != nil {
		for d, ex := range examples {
			exdesc = fmt.Sprintf("%s\n%s%s", exdesc, wouter.Wrap(d, width), winner.Wrap(ex, width))
		}
	}
	return
}
