package repo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/G-Node/gin-cli/util"
	"github.com/G-Node/gin-cli/web"
	"github.com/G-Node/gin-repo/wire"
)

// Client is a client interface to the repo server. Embeds web.Client.
type Client struct {
	*web.Client
	KeyHost string
	GitHost string
	GitUser string
}

// NewClient returns a new client for the repo server.
func NewClient(host string) *Client {
	return &Client{Client: web.NewClient(host)}
}

// GetRepos gets a list of repositories (public or user specific)
func (repocl *Client) GetRepos(user string) ([]wire.Repo, error) {
	util.LogWrite("Retrieving repos")
	var repoList []wire.Repo
	var res *http.Response
	var err error

	if user == "" {
		util.LogWrite("User: public")
		res, err = repocl.Get("/repos/public")
		fmt.Print("Listing all public repositories\n\n")
	} else {
		util.LogWrite("User: %s", user)
		err = repocl.LoadToken()
		if err != nil {
			fmt.Print("You are not logged in - Showing public repositories\n\n")
		}
		res, err = repocl.Get(fmt.Sprintf("/users/%s/repos", user))
	}

	if err != nil {
		return repoList, err
	} else if res.StatusCode == 404 {
		return repoList, fmt.Errorf("Server returned empty result. Either user does not exist or has no accessible repositories.")
	} else if res.StatusCode != 200 {
		return repoList, fmt.Errorf("[Repository request] Failed. Server returned: %s", res.Status)
	}

	defer web.CloseRes(res.Body)
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return repoList, err
	}
	err = json.Unmarshal(b, &repoList)
	return repoList, err
}

// CreateRepo creates a repository on the server.
func (repocl *Client) CreateRepo(name, description string) error {
	util.LogWrite("Creating repository")
	err := repocl.LoadToken()
	if err != nil {
		return fmt.Errorf("[Create repository] This action requires login")
	}

	data := wire.Repo{Name: name, Description: description}
	util.LogWrite("Name: %s :: Description: %s", name, description)
	res, err := repocl.Post(fmt.Sprintf("/users/%s/repos", repocl.Username), data)
	if err != nil {
		return err
	} else if res.StatusCode != 201 {
		return fmt.Errorf("[Create repository] Failed. Server returned %s", res.Status)
	}
	web.CloseRes(res.Body)
	util.LogWrite("Repository created")
	return nil
}

// UploadRepo adds files to a repository and uploads them.
func (repocl *Client) UploadRepo(localPath string) error {
	defer CleanUpTemp()
	util.LogWrite("UploadRepo")

	err := repocl.Connect()
	if err != nil {
		return err
	}

	added, err := AnnexAdd(localPath)
	if err != nil {
		return err
	}

	if len(added) == 0 {
		return fmt.Errorf("No changes to upload")
	}

	changes, err := DescribeChanges(localPath)
	// add header commit line
	changes = fmt.Sprintf("gin upload\n\n%s", changes)
	if err != nil {
		return err
	}

	// fmt.Println(changes)

	err = AnnexPush(localPath, changes)
	return err
}

// DownloadRepo downloads the files in an already checked out repository.
func (repocl *Client) DownloadRepo(localPath string) error {
	defer CleanUpTemp()
	util.LogWrite("DownloadRepo")

	err := repocl.Connect()
	if err != nil {
		return err
	}
	err = AnnexPull(localPath)
	return err
}

// CloneRepo downloads the files of a given repository.
func (repocl *Client) CloneRepo(repoPath string) error {
	defer CleanUpTemp()
	util.LogWrite("CloneRepo")

	err := repocl.Connect()
	if err != nil {
		return err
	}

	localPath := path.Base(repoPath)
	fmt.Printf("Fetching repository '%s'... ", localPath)
	err = repocl.Clone(repoPath)
	if err != nil {
		return err
	}
	fmt.Printf("done.\n")

	// git annex init the clone and set defaults
	err = AnnexInit(localPath)
	if err != nil {
		return err
	}

	annexFiles, err := AnnexWhereis(localPath)
	if err != nil {
		return err
	}
	if len(annexFiles) == 0 {
		return nil
	}

	fmt.Printf("Downloading files... ")
	err = AnnexPull(localPath)
	if err != nil {
		return err
	}
	fmt.Printf("done.\n")
	return nil
}