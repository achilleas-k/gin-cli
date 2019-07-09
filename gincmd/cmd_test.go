package gincmd

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	ginclient "github.com/G-Node/gin-cli/ginclient"
	"github.com/G-Node/gin-cli/ginclient/config"
	"github.com/G-Node/gin-cli/git"
)

const testalias = "test"

func zerostatus() map[ginclient.FileStatus]int {
	return map[ginclient.FileStatus]int{
		ginclient.Synced:       0,
		ginclient.TypeChange:   0,
		ginclient.NoContent:    0,
		ginclient.Modified:     0,
		ginclient.LocalChanges: 0,
		ginclient.Removed:      0,
		ginclient.Untracked:    0,
	}
}

// makeRandFile creates a random binary file with a given name and size in
// kilobytes
func makeRandFile(name string, size uint64) {
	file, err := os.Create(name)
	CheckError(err)
	buf := make([]byte, 1024)
	for count := uint64(0); count < size; count++ {
		_, err = rand.Read(buf)
		CheckError(err)
		file.Write(buf)
	}
}

func assertStatus(path string, expected map[ginclient.FileStatus]int) {
	gincl := ginclient.New(testalias)
	filestatus, err := gincl.ListFiles(path)
	CheckError(err)

	// collect status counts
	actual := zerostatus()
	for _, status := range filestatus {
		actual[status]++
	}

	for status := range expected {
		if actual[status] != expected[status] {
			os.Exit(1) // TODO: Print useful error
		}
	}
}

func addTestServer() {

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
	err = config.AddServerConf(testalias, serverConf)
	CheckError(err)

	// Recreate known hosts file
	err = git.WriteKnownHosts()
	CheckError(err)

	err = ginclient.SetDefaultServer(testalias)
	CheckError(err)
}

func loginTestuser() {
	username := "testuser"
	password := "a test password 42"

	gincl := ginclient.New("test")
	err := gincl.Login(username, password, "gin-cli")
	CheckError(err)
}

func createTestRepository() string {
	rand.Seed(time.Now().UnixNano())
	reponame := fmt.Sprintf("gin-test-%04d", rand.Intn(9999))

	// gincl := ginclient.New("test")
	// err := gincl.LoadToken()
	// CheckError(err)
	// repopath := fmt.Sprintf("%s/%s", gincl.Username, reponame)
	// fmt.Printf("Creating repository %s\n", repopath)
	// err = gincl.CreateRepo(reponame, "Test repository")
	// CheckError(err)
	os.Args = []string{"", reponame}
	cmd := CreateCmd()
	err := cmd.Execute()
	CheckError(err)
	return reponame
}

func deleteRepository(reponame string) {
	gincl := ginclient.New("test")
	err := gincl.LoadToken()
	CheckError(err)
	repopath := fmt.Sprintf("%s/%s", gincl.Username, reponame)
	fmt.Printf("Cleaning up %s\n", repopath)
	err = gincl.DelRepo(repopath)
	CheckError(err)
}

// TestMain sets up a temporary git configuration directory to avoid effects
// from user or local git configurations.
func TestMain(m *testing.M) {
	// Setup test config
	tmpconfdir, err := ioutil.TempDir("", "gin-test-config-")
	if err != nil {
		os.Exit(-1)
	}

	// set temporary git config file path and disable systemwide
	os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpconfdir, "gitconfig"))

	// set temporary GIN config directory
	os.Setenv("GIN_CONFIG_DIR", filepath.Join(tmpconfdir, "gin"))

	// configure test server
	addTestServer()

	// login
	loginTestuser()

	res := m.Run()

	// Teardown test config
	os.RemoveAll(tmpconfdir)
	os.Exit(res)
}

func TestStuff(t *testing.T) {
	// create temporary working directory
	tmpworkdir, err := ioutil.TempDir("", "gin-test-dir")
	CheckError(err)
	defer os.RemoveAll(tmpworkdir)

	origdir, _ := os.Getwd()
	defer os.Chdir(origdir)

	os.Chdir(tmpworkdir)
	loginTestuser()
	reponame := createTestRepository()
	// dir, _ := os.Getwd()
	defer deleteRepository(reponame)

	// TODO: port test_all_states
	filestatus := zerostatus()
	filestatus[ginclient.Untracked] += 70
	// t.Fail vs exiting directly?
	for idx := 0; idx < 50; idx++ {
		makeRandFile(fmt.Sprintf("root-%d.git", idx), 5)
	}
	for idx := 70; idx < 90; idx++ {
		makeRandFile(fmt.Sprintf("root-%d.annex", idx), 2000)
	}
	run("pwd")
	assertStatus(".", filestatus)
	return
}

func run(command string, args ...string) {
	cmd := exec.Command(command, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		println(err.Error())
	}
	println(string(out))
}
