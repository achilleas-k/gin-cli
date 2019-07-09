package gincmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	ginclient "github.com/G-Node/gin-cli/ginclient"
	"github.com/G-Node/gin-cli/ginclient/config"
	"github.com/G-Node/gin-cli/git"
)

func addTestServer() {
	alias := "test"

	webstring := "http://localhost:3000"
	gitstring := "git@localhost:2222"

	serverConf := config.ServerCfg{}

	var err error
	serverConf.Web, err = config.ParseWebString(webstring)
	CheckError(err)

	serverConf.Git, err = config.ParseGitString(gitstring)
	CheckError(err)

	hostkeystr, _, err := git.GetHostKey(serverConf.Git)
	CheckError(err)
	serverConf.Git.HostKey = hostkeystr

	// Save to config
	err = config.AddServerConf(alias, serverConf)
	CheckError(err)

	// Recreate known hosts file
	err = git.WriteKnownHosts()
	CheckError(err)

	err = ginclient.SetDefaultServer(alias)
	CheckError(err)
}

// TestMain sets up a temporary git configuration directory to avoid effects
// from user or local git configurations.
func TestMain(m *testing.M) {
	// Setup test config
	tmpconfdir, err := ioutil.TempDir("", "git-test-config-")
	if err != nil {
		os.Exit(-1)
	}

	// set temporary GIN config directory
	os.Setenv("GIN_CONFIG_DIR", filepath.Join(tmpconfdir, "gin"))

	// configure test server
	addTestServer()

	// set temporary git config file path and disable systemwide
	os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpconfdir, "gitconfig"))

	// set git user
	res := m.Run()

	// Teardown test config
	os.RemoveAll(tmpconfdir)
	os.Exit(res)
}

func TestStuff(t *testing.T) {
	// rootCmd := SetUpCommands(VersionInfo{})
	// rootCmd.SetVersionTemplate("{{ .Version }}")

	try := func(err error) {
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
	}

	cmd := ServersCmd()
	try(cmd.Execute())
	return
}
