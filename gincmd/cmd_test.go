package gincmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	ginclient "github.com/G-Node/gin-cli/ginclient"
	"github.com/G-Node/gin-cli/ginclient/config"
	"github.com/G-Node/gin-cli/git"
	"github.com/spf13/cobra"
)

const testalias = "test"

func errcheck(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

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
func makeRandFile(name string, size uint64) error {
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	buf := make([]byte, 1024)
	for count := uint64(0); count < size; count++ {
		_, err = rand.Read(buf)
		if err != nil {
			return err
		}
		file.Write(buf)
	}
	return nil
}

func assertStatus(t *testing.T, path string, expected map[ginclient.FileStatus]int, msg string) {
	gincl := ginclient.New(testalias)
	filestatus, err := gincl.ListFiles(path)
	errcheck(t, err)

	// collect status counts
	actual := zerostatus()
	for _, status := range filestatus {
		actual[status]++
	}

	fail := false
	for status := range expected {
		if actual[status] != expected[status] {
			fmt.Printf("File status count mismatch: %s %d != %d\n", status.Abbrev(), actual[status], expected[status])
			fail = true
		}
	}
	if fail {
		t.Fatalf("File status assertion failed: %s", msg)
	}
}

func revCount(t *testing.T) uint64 {
	cmd := git.Command("rev-list", "--count", "master")
	output, err := cmd.Output()
	errcheck(t, err)

	output = bytes.TrimSpace(output)
	count, err := strconv.Atoi(string(output))
	if err != nil {
		t.Fatalf("error: failed to parse output of rev count '%s': %v", output, err)
	}

	if count <= 0 {
		t.Fatalf("error: rev count returned non-positive number %d", count)
	}
	return uint64(count)
}

func addTestServer() {

	webstring := "http://localhost:3000"
	gitstring := "git@localhost:2222"

	serverConf := config.ServerCfg{}

	check := func(err error) {
		if err != nil {
			log.Fatalf("error while setting up test server configuration: %v", err)
		}
	}

	var err error
	serverConf.Web, err = config.ParseWebString(webstring)
	check(err)

	serverConf.Git, err = config.ParseGitString(gitstring)
	check(err)

	hostkeystr, _, err := git.GetHostKey(serverConf.Git)
	check(err)
	serverConf.Git.HostKey = hostkeystr

	// Save to config
	err = config.AddServerConf(testalias, serverConf)
	check(err)

	// Recreate known hosts file
	err = git.WriteKnownHosts()
	check(err)

	err = ginclient.SetDefaultServer(testalias)
	check(err)
}

func loginTestuser(t *testing.T) {
	username := "testuser"
	password := "a test password 42"

	gincl := ginclient.New("test")
	err := gincl.Login(username, password, "gin-cli")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
}

func createTestRepository(t *testing.T) string {
	rand.Seed(time.Now().UnixNano())
	reponame := fmt.Sprintf("gin-test-%04d", rand.Intn(9999))

	// gincl := ginclient.New("test")
	// err := gincl.LoadToken()
	// repopath := fmt.Sprintf("%s/%s", gincl.Username, reponame)
	// fmt.Printf("Creating repository %s\n", repopath)
	// err = gincl.CreateRepo(reponame, "Test repository")
	os.Args = []string{"", reponame}
	cmd := CreateCmd()
	err := cmd.Execute()
	errcheck(t, err)
	return reponame
}

func deleteRepository(t *testing.T, reponame string) {
	gincl := ginclient.New("test")
	err := gincl.LoadToken()
	errcheck(t, err)
	repopath := fmt.Sprintf("%s/%s", gincl.Username, reponame)
	fmt.Printf("Cleaning up %s\n", repopath)
	err = gincl.DelRepo(repopath)
	errcheck(t, err)
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

	res := m.Run()

	// Teardown test config
	os.RemoveAll(tmpconfdir)
	os.Exit(res)
}

func TestStuff(t *testing.T) {
	// create temporary working directory
	tmpworkdir, err := ioutil.TempDir("", "gin-test-dir")
	errcheck(t, err)
	defer os.RemoveAll(tmpworkdir)

	origdir, _ := os.Getwd()
	defer os.Chdir(origdir)

	os.Chdir(tmpworkdir)
	loginTestuser(t)
	reponame := createTestRepository(t)
	// dir, _ := os.Getwd()
	defer deleteRepository(t, reponame)

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
	assertStatus(t, ".", filestatus, "Initial file creation")

	// Commit and check status
	err = runSubcommand(CommitCmd(), "root*")
	errcheck(t, err)
	filestatus[ginclient.LocalChanges] += 70
	filestatus[ginclient.Untracked] -= 70
	assertStatus(t, ".", filestatus, "First commit")

	// Upload and check status
	err = runSubcommand(UploadCmd())
	errcheck(t, err)
	filestatus[ginclient.Synced] += 70
	filestatus[ginclient.LocalChanges] -= 70
	assertStatus(t, ".", filestatus, "First ")

	// gin upload command should not have created an extra commit
	if count := revCount(t); count != 2 {
		t.Fatalf("error: Expected 2 revisions, got %d", count)
	}
	return
}

func runSubcommand(cmd *cobra.Command, args ...string) error {
	args = append([]string{cmd.Name()}, args...)
	os.Args = args
	return cmd.Execute()
}

func run(command string, args ...string) {
	cmd := exec.Command(command, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		println(err.Error())
	}
	println(string(out))
}
