package repo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/G-Node/gin-cli/auth"
	"github.com/G-Node/gin-cli/util"
)

// Temporary (SSH key) file handling
var privKeyFile util.TempFile

// MakeTempKeyPair creates a temporary key pair and stores it in a temporary directory.
// It also sets the global tempFile for use by the annex commands. The key pair is returned directly.
func (repocl *Client) MakeTempKeyPair() (*util.KeyPair, error) {
	tempKeyPair, err := util.MakeKeyPair()
	if err != nil {
		return nil, err
	}

	description := fmt.Sprintf("tmpkey@%s", strconv.FormatInt(time.Now().Unix(), 10))
	pubkey := fmt.Sprintf("%s %s", strings.TrimSpace(tempKeyPair.Public), description)
	authcl := auth.NewClient(repocl.KeyHost)
	err = authcl.AddKey(pubkey, description, true)
	if err != nil {
		return tempKeyPair, err
	}

	privKeyFile, err = util.SaveTempKeyFile(tempKeyPair.Private)
	if err != nil {
		return tempKeyPair, err
	}

	privKeyFile.Active = true

	return tempKeyPair, nil
}

// CleanUpTemp deletes the temporary directory which holds the temporary private key if it exists.
func CleanUpTemp() {
	if privKeyFile.Active {
		privKeyFile.Delete()
	}
}

// **************** //

// Git commands

// IsRepo checks whether a given path is a git repository.
func IsRepo(path string) bool {
	gitbin := util.Config.Bin.Git
	cmd := exec.Command(gitbin, "status")
	cmd.Dir = path
	err := cmd.Run()
	if err != nil {
		return false
	}
	return true
}

func publicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

// Connect opens a connection to the git server. This is used to validate credentials
// and generate temporary keys on demand, without performing a git operation.
// On Unix systems, the function will attempt to use the system's SSH agent.
// If no agent is running or the keys offered by the agent are not valid for the server,
// a temporary key pair is generated, the public key is uploaded to the auth server,
// and the private key is stored internally, to be used for subsequent functions.
func (repocl *Client) Connect() error {
	util.LogWrite("Checking connection to git server")
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		// No agent running - use temp keys
		util.LogWrite("No agent running. Setting up temporary keys")
		_, err = repocl.MakeTempKeyPair()
		if err != nil {
			return fmt.Errorf("Error while creating temporary key for connection: %s", err.Error())
		}
		return nil
	}

	agent := ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)

	sshConfig := &ssh.ClientConfig{
		User: repocl.GitUser,
		Auth: []ssh.AuthMethod{
			agent,
		},
	}

	util.LogWrite("Attempting connection with key from SSH agent")
	connection, err := ssh.Dial("tcp", repocl.GitHost, sshConfig)
	if err != nil && strings.Contains(err.Error(), "unable to authenticate") {
		// Agent key authentication failed - use temp keys
		util.LogWrite("Auth key authentication failed. Setting up temporary keys")
		_, err = repocl.MakeTempKeyPair()
		if err != nil {
			return fmt.Errorf("Error while creating temporary key for connection: %s", err.Error())
		}
		return nil
	}
	// TODO: Attempt connection again after temp key is set up

	if err != nil {
		// Connection error (other than "unable to auth")
		return fmt.Errorf("Failed to connect to git host: %s\n", err.Error())
	}
	defer connection.Close()

	session, err := connection.NewSession()
	util.LogWrite("Creating SSH session")
	if err != nil {
		return fmt.Errorf("Failed to create session: %s", err.Error())
	}
	defer session.Close()
	util.LogWrite("Connection to git server OK")
	return nil
}

// Clone downloads a repository and sets the remote fetch and push urls.
// (git clone ...)
func (repocl *Client) Clone(repopath string) error {
	gitbin := util.Config.Bin.Git
	remotePath := fmt.Sprintf("ssh://%s@%s/%s", repocl.GitUser, repocl.GitHost, repopath)
	var cmd *exec.Cmd
	cmd = exec.Command(gitbin)
	if privKeyFile.Active {
		env := os.Environ()
		cmd.Env = append(env, privKeyFile.GitSSHEnv())
	}
	cmd.Args = append(cmd.Args, "clone", remotePath)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	util.LogWrite("Running shell command: %s", strings.Join(cmd.Args, " "))
	err := cmd.Run()
	if err != nil {
		util.LogWrite("Error during clone command")
		util.LogWrite("[stdout]\r\n%s", out.String())
		util.LogWrite("[stderr]\r\n%s", stderr.String())
		return fmt.Errorf("Error retrieving repository")
	}
	return nil
}

// **************** //

// Git annex commands

func buildAnnexCmd(args ...string) *exec.Cmd {
	gitannexbin := util.Config.Bin.GitAnnex
	cmd := exec.Command(gitannexbin, args...)
	annexsshopt := "annex.ssh-options=-o StrictHostKeyChecking=no"
	if privKeyFile.Active {
		annexsshopt = fmt.Sprintf("%s -i %s", annexsshopt, privKeyFile.FullPath())
	}
	cmd.Args = append(cmd.Args, "-c", annexsshopt)
	return cmd
}

// AnnexInit initialises the repository for annex
// (git annex init)
func AnnexInit(localPath string) error {
	gitbin := util.Config.Bin.Git
	initError := fmt.Errorf("Repository annex initialisation failed.")
	cmd := buildAnnexCmd("init", "--version=6")
	cmd.Dir = localPath
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	util.LogWrite("Running shell command: %s", strings.Join(cmd.Args, " "))
	err := cmd.Run()
	if err != nil {
		util.LogWrite(initError.Error())
		util.LogWrite("[stdout]\r\n%s", out.String())
		util.LogWrite("[stderr]\r\n%s", stderr.String())
		return initError
	}

	cmd = exec.Command(gitbin, "config", "annex.addunlocked", "true")
	cmd.Dir = localPath
	util.LogWrite("Running shell command: %s", strings.Join(cmd.Args, " "))
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		util.LogWrite(initError.Error())
		util.LogWrite("[stdout]\r\n%s", out.String())
		util.LogWrite("[stderr]\r\n%s", stderr.String())
		return initError
	}

	// list of extensions that are added to git (not annex)
	exclfilters := util.Config.Annex.Exclude
	excludes := make([]string, len(exclfilters))
	for idx, ext := range exclfilters {
		excludes[idx] = fmt.Sprintf("exclude=%s", ext)
	}
	sizethreshold := fmt.Sprintf("largerthan=%s", util.Config.Annex.MinSize)
	conditions := append(excludes, sizethreshold)
	lfvalue := strings.Join(conditions, " and ")

	if lfvalue != "" {
		cmd = exec.Command(gitbin, "config", "annex.largefiles", lfvalue)
		cmd.Dir = localPath
		util.LogWrite("Running shell command: %s", strings.Join(cmd.Args, " "))
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			util.LogWrite(initError.Error())
			util.LogWrite("[stdout]\r\n%s", out.String())
			util.LogWrite("[stderr]\r\n%s", stderr.String())
			return initError
		}
		if err != nil {
			return initError
		}
	}
	cmd = exec.Command(gitbin, "config", "annex.backends", "WORM")
	cmd.Dir = localPath
	util.LogWrite("Running shell command: %s", strings.Join(cmd.Args, " "))
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		util.LogWrite("[stdout]\r\n%s", out.String())
		util.LogWrite("[stderr]\r\n%s", stderr.String())
		return initError
	}
	cmd = exec.Command(gitbin, "config", "annex.thin", "true")
	cmd.Dir = localPath
	util.LogWrite("Running shell command: %s", strings.Join(cmd.Args, " "))
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		util.LogWrite("[stdout]\r\n%s", out.String())
		util.LogWrite("[stderr]\r\n%s", stderr.String())
		return initError
	}
	return nil
}

// AnnexPull downloads all annexed files.
// (git annex sync --no-push --content)
func AnnexPull(localPath string) error {
	cmd := buildAnnexCmd("sync", "--no-push", "--content")
	cmd.Dir = localPath
	util.LogWrite("Running shell command: %s", strings.Join(cmd.Args, " "))
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		util.LogWrite("Error during AnnexPull.")
		util.LogWrite("[stdout]\r\n%s", out.String())
		util.LogWrite("[stderr]\r\n%s", stderr.String())
		return fmt.Errorf("Error downloading files")
	}
	return nil
}

// AnnexSync synchronises the local repository with the remote.
// (git annex sync --content)
func AnnexSync(localPath string) error {
	cmd := buildAnnexCmd("sync", "--content")
	cmd.Dir = localPath
	util.LogWrite("Running shell command: %s", strings.Join(cmd.Args, " "))
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		util.LogWrite("Error during AnnexSync")
		util.LogWrite("[stdout]\r\n%s", out.String())
		util.LogWrite("[stderr]\r\n%s", stderr.String())
		return fmt.Errorf("Error synchronising files")
	}
	return nil
}

// AnnexPush uploads all annexed files.
// (git annex sync --no-pull --content)
func AnnexPush(localPath, commitMsg string) error {
	cmd := buildAnnexCmd("sync", "--no-pull", "--content", "--commit", fmt.Sprintf("--message=%s", commitMsg))
	cmd.Dir = localPath
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	util.LogWrite("Running shell command: %s", strings.Join(cmd.Args, " "))
	err := cmd.Run()

	if err != nil {
		util.LogWrite("Error during AnnexPush")
		util.LogWrite("[stdout]\r\n%s", out.String())
		util.LogWrite("[stderr]\r\n%s", stderr.String())
		return fmt.Errorf("Error uploading files")
	}
	return nil
}

// AnnexAddResult ...
type AnnexAddResult struct {
	Command string `json:"command"`
	File    string `json:"file"`
	Key     string `json:"key"`
	Success bool   `json:"success"`
}

// AnnexAdd adds a path to the annex.
// (git annex add)
func AnnexAdd(localPath string) ([]string, error) {
	gitannexbin := util.Config.Bin.GitAnnex
	cmd := exec.Command(gitannexbin, "--json", "add", localPath)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	util.LogWrite("Running shell command: %s", strings.Join(cmd.Args, " "))
	err := cmd.Run()

	if err != nil {
		util.LogWrite("Error during AnnexAdd")
		util.LogWrite("[stdout]\r\n%s", out.String())
		util.LogWrite("[stderr]\r\n%s", stderr.String())
		return nil, fmt.Errorf("Error adding files to repository.")
	}

	var outStruct AnnexAddResult
	files := bytes.Split(out.Bytes(), []byte("\n"))
	added := make([]string, 0, len(files))
	for _, f := range files {
		if len(f) == 0 {
			continue
		}
		err := json.Unmarshal(f, &outStruct)
		if err != nil {
			return nil, err
		}
		if !outStruct.Success {
			return nil, fmt.Errorf("Error adding files to repository: Failed to add %s", outStruct.File)
		}
		added = append(added, outStruct.File)
	}

	return added, nil
}

// AnnexWhereisResult holds the JSON output of a "git annex whereis" command
type AnnexWhereisResult struct {
	File      string   `json:"file"`
	Command   string   `json:"command"`
	Note      string   `json:"note"`
	Success   bool     `json:"success"`
	Untrusted []string `json:"untrusted"`
	Whereis   []struct {
		Here        bool     `json:"here"`
		UUID        string   `json:"uuid"`
		URLs        []string `json:"urls"`
		Description string   `json:"description"`
	}
	Key string `json:"key"`
}

// AnnexWhereis returns information about annexed files in the repository
// (git annex find)
func AnnexWhereis(localPath string) ([]AnnexWhereisResult, error) {
	cmd := buildAnnexCmd("whereis", "--json")
	cmd.Dir = localPath
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	util.LogWrite("Running shell command: %s", strings.Join(cmd.Args, " "))
	err := cmd.Run()

	if err != nil {
		util.LogWrite("Error during AnnexWhereis")
		util.LogWrite("[stdout]\r\n%s", out.String())
		util.LogWrite("[stderr]\r\n%s", stderr.String())
		return nil, fmt.Errorf("Error getting file status from server")
	}

	resultsJSON := bytes.Split(out.Bytes(), []byte("\n"))
	results := make([]AnnexWhereisResult, 0, len(resultsJSON))
	var res AnnexWhereisResult
	for _, resJSON := range resultsJSON {
		if len(resJSON) == 0 {
			continue
		}
		err := json.Unmarshal(resJSON, &res)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, nil
}

// AnnexStatusResult ...
type AnnexStatusResult struct {
	Status string `json:"status"`
	File   string `json:"file"`
}

// DescribeChanges returns a string which describes the status of the files in the working tree
// with respect to git annex. The resulting message can be used to inform the user of changes
// that are about to be uploaded and as a long commit message.
func DescribeChanges(localPath string) (string, error) {
	gitannexbin := util.Config.Bin.GitAnnex
	cmd := exec.Command(gitannexbin, "status", "--json")
	cmd.Dir = localPath
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	util.LogWrite("Running shell command: %s", strings.Join(cmd.Args, " "))
	err := cmd.Run()

	if err != nil {
		util.LogWrite("Error during DescribeChanges")
		util.LogWrite("[stdout]\r\n%s", out.String())
		util.LogWrite("[stderr]\r\n%s", stderr.String())
		return "", fmt.Errorf("Error retrieving file status")
	}

	var outStruct AnnexStatusResult
	files := bytes.Split(out.Bytes(), []byte("\n"))

	statusmap := make(map[string][]string)
	for _, f := range files {
		if len(f) == 0 {
			continue
		}
		err := json.Unmarshal(f, &outStruct)
		if err != nil {
			return "", err
		}
		statusmap[outStruct.Status] = append(statusmap[outStruct.Status], outStruct.File)
	}

	var changeList string
	changeList += makeFileList("New files", statusmap["A"])
	changeList += makeFileList("Modified files", statusmap["M"])
	changeList += makeFileList("Deleted files", statusmap["D"])
	changeList += makeFileList("Type modified files", statusmap["T"])
	changeList += makeFileList("Untracked files ", statusmap["?"])

	return changeList, nil
}

func makeFileList(header string, fnames []string) (list string) {
	if len(fnames) == 0 {
		return
	}
	list += fmt.Sprint(header) + "\n"
	for idx, name := range fnames {
		list += fmt.Sprintf("  %d: %s\n", idx+1, name)
	}
	list += "\n"
	return
}