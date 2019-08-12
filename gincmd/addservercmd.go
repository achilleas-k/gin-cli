package gincmd

import (
	"fmt"
	"strings"

	ginclient "github.com/G-Node/gin-cli/ginclient"
	"github.com/G-Node/gin-cli/ginclient/config"
	"github.com/G-Node/gin-cli/gincmd/ginerrors"
	"github.com/spf13/cobra"
)

func promptForWeb() (webconf config.WebCfg) {
	fmt.Println(":: Web server configuration")
	fmt.Print("  Protocol (e.g., http, https): ")
	fmt.Scanln(&webconf.Protocol)
	fmt.Print("  Host or address: ")
	fmt.Scanln(&webconf.Host)

	fmt.Print("  Port (e.g., 80, 443): ")
	_, err := fmt.Scanln(&webconf.Port)
	if err != nil {
		Die(ginerrors.BadPort)
	}
	return
}

func promptForGit() (gitconf config.GitCfg) {
	fmt.Println(":: Git server configuration")
	fmt.Print("  Username: ")
	fmt.Scanln(&gitconf.User)
	fmt.Print("  Host or address: ")
	fmt.Scanln(&gitconf.Host)
	fmt.Print("  Port (e.g., 22, 2222): ")
	_, err := fmt.Scanln(&gitconf.Port)
	if err != nil {
		Die(ginerrors.BadPort)
	}
	return
}

func addHostKey(gitconf *config.GitCfg) {
	address := fmt.Sprintf("%s:%d", gitconf.Host, gitconf.Port)
	hostkeystr, fingerprint, err := ginclient.GetHostKey(gitconf.User, address)
	CheckError(err)
	fmt.Printf(":: Host key fingerprint for [%s]: %s\n", gitconf.AddressStr(), fingerprint)
	fmt.Print("Accept [yes/no]: ")
	var response string
	fmt.Scanln(&response)
	for cont := false; !cont; {
		switch strings.ToLower(response) {
		case "no":
			Exit("Aborted")
		case "yes":
			cont = true
		default:
			fmt.Print("Please type 'yes' or 'no': ")
			fmt.Scanln(&response)
		}
	}

	gitconf.HostKey = hostkeystr
	return
}

func addServer(cmd *cobra.Command, args []string) {
	alias := args[0]

	if alias == "dir" {
		Die(fmt.Sprintf("invalid server alias '%s': this word is reserved", alias))
	}

	webstring, _ := cmd.Flags().GetString("web")
	gitstring, _ := cmd.Flags().GetString("git")

	serverConf := config.ServerCfg{}

	var err error
	if webstring == "" {
		serverConf.Web = promptForWeb()
	} else {
		serverConf.Web, err = config.ParseWebString(webstring)
		CheckError(err)
	}

	if gitstring == "" {
		serverConf.Git = promptForGit()
	} else {
		serverConf.Git, err = config.ParseGitString(gitstring)
		CheckError(err)
	}

	addHostKey(&serverConf.Git)

	// Save to config
	err = config.AddServerConf(alias, serverConf)
	if err != nil {
		Die("failed to update configuration file")
	}

	// Recreate known hosts file
	err = ginclient.WriteKnownHosts()
	if err != nil {
		Die("failed to write known_hosts file")
	}
}

// AddServerCmd sets up the 'add-server' command for adding new server configurations
func AddServerCmd() *cobra.Command {
	description := `Configure a new GIN server that can be used to add remotes to repositories.

The command requires only one argument, the alias for the server. All other information can be provided on the command line using the flags described below. You will be prompted for any required information that is not provided.

When configuring a server, you must specify an alias (name) for it, which will be used to refer to the configured server. This alias is then used when adding a remote to a repository. The default G-Node GIN server is available under the name 'gin', but this may be overridden. The word 'dir' cannot be used as is has special meaning when adding a remote to a repository. See 'gin help add-remote'.

The following information is required to configure a new server:

For the web server: the protocol, hostname, and port

    The protocol should be either 'http' or 'https'.
    The hostname for the server (e.g., gin.g-node.org).
    The web port for the server (e.g., 80, 443).

For the git server: the git user, hostname, and port

    The git user is the username set on the server for managing the repositories. It is almost always 'git'.
    The hostname for the git server (e.g., git.g-node.org).
    The git port for the server (e.g., 22, 2222).

See the Examples section for a full example.
`
	args := map[string]string{
		"<alias>": "The alias (name) for the server.",
	}
	examples := map[string]string{
		"This is what configuring the built-in G-Node GIN server would look like (note: this is already configured)": "$ gin add-server --web https://gin.g-node.org:443 --git git@gin.g-node.org:22 gin",
	}
	var cmd = &cobra.Command{
		Use:                   "add-server [--web http[s]://<hostname>[:<port>]] [--git [<gituser>@]<hostname>[:<port>]] <alias>",
		Short:                 "Add a new GIN server configuration",
		Long:                  formatdesc(description, args),
		Args:                  cobra.ExactArgs(1),
		Example:               formatexamples(examples),
		Run:                   addServer,
		DisableFlagsInUseLine: true,
	}
	cmd.Flags().String("web", "", "Set the address and port for the web server.")
	cmd.Flags().String("git", "", "Set the user, address and port for the git server.")
	return cmd
}
