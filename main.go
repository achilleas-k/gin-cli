package main

// TODO: Nicer error handling. Print useful, descriptive messages.

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/G-Node/gin-cli/auth"
	"github.com/G-Node/gin-cli/repo"
	"github.com/G-Node/gin-cli/util"
	"github.com/docopt/docopt-go"
)

func createRepo(name, description interface{}) {
	var repoName string
	if name == nil || name.(string) == "" {
		fmt.Print("Repository name: ")
		repoName = ""
		fmt.Scanln(&repoName)
	} else {
		repoName = name.(string)
	}
	// TODO: Check name validity before sending to server?
	repoDesc := ""
	if description != nil {
		repoDesc = description.(string)
	}
	err := repo.CreateRepo(repoName, repoDesc)
	util.CheckError(err)
}

func upload(patharg interface{}) {
	var pathstr string
	if patharg != nil {
		pathstr = patharg.(string)
	}
	err := repo.UploadRepo(pathstr)
	util.CheckError(err)
}

func download(patharg interface{}) {
	var pathstr string
	if patharg != nil {
		pathstr = patharg.(string)
		err := repo.CloneRepo(pathstr)
		util.CheckError(err)
	}

	// No repo specified -- attempting to pull in cwd
	if repo.IsRepo(".") {
		err := repo.DownloadRepo()
		util.CheckError(err)
	}

	util.CheckError(fmt.Errorf("Current directory is not a repository."))
}

// condAppend Conditionally append str to b if not empty
func condAppend(b *bytes.Buffer, str *string) {
	if str != nil && *str != "" {
		_, _ = b.WriteString(*str + " ")
	}
}

func printKeys(printFull bool) {

	keys, err := auth.GetUserKeys()
	util.CheckError(err)
	nkeys := len(keys)
	var plural string
	if nkeys == 1 {
		plural = ""
	} else {
		plural = "s"
	}
	fmt.Printf("You have %d key%s associated with your account.\n\n", nkeys, plural)
	for idx, key := range keys {
		fmt.Printf("[%v] \"%s\" ", idx+1, key.Description)
		fmt.Printf("(Fingerprint: %s)\n", key.Fingerprint)
		if printFull {
			fmt.Printf("--- Key ---\n%s\n", key.Key)
		}
	}
}

func addKey(fnarg interface{}) {
	noargError := fmt.Errorf("Please specify a public key file.\n")
	if fnarg == nil {
		util.CheckError(noargError)
	}

	filename := fnarg.(string)
	if filename == "" {
		util.CheckError(noargError)
	}

	keyBytes, err := ioutil.ReadFile(filename)
	util.CheckError(err)
	// TODO: Accept custom description for key and simply default to filename
	key := string(keyBytes)
	description := strings.TrimSpace(strings.Split(key, " ")[2])
	err = auth.AddKey(string(keyBytes), description)
	util.CheckError(err)
	fmt.Printf("New key added '%s'\n", description)
}

func printAccountInfo(userarg interface{}) {
	var username string
	currentUser, token, _ := auth.LoadToken(true)

	if userarg == nil {
		username = currentUser
	} else {
		username = userarg.(string)
	}

	if username == "" {
		// prompt for username
		fmt.Print("Specify username for info lookup: ")
		username = ""
		fmt.Scanln(&username)
	}

	info, err := auth.RequestAccount(username, token)
	util.CheckError(err)

	var fullnameBuffer bytes.Buffer

	condAppend(&fullnameBuffer, info.Title)
	condAppend(&fullnameBuffer, &info.FirstName)
	condAppend(&fullnameBuffer, info.MiddleName)
	condAppend(&fullnameBuffer, &info.LastName)

	var outBuffer bytes.Buffer

	_, _ = outBuffer.WriteString(fmt.Sprintf("User [%s]\nName: %s\n", info.Login, fullnameBuffer.String()))

	if info.Email != nil && info.Email.Email != "" {
		_, _ = outBuffer.WriteString(fmt.Sprintf("Email: %s\n", info.Email.Email))
		// TODO: Display public status if current user == info.Login
	}

	if info.Affiliation != nil {
		var affiliationBuffer bytes.Buffer
		affiliation := info.Affiliation

		condAppend(&affiliationBuffer, &affiliation.Department)
		condAppend(&affiliationBuffer, &affiliation.Institute)
		condAppend(&affiliationBuffer, &affiliation.City)
		condAppend(&affiliationBuffer, &affiliation.Country)

		if affiliationBuffer.Len() > 0 {
			_, _ = outBuffer.WriteString(fmt.Sprintf("Affiliation: %s\n", affiliationBuffer.String()))
		}
	}

	fmt.Println(outBuffer.String())
}

func listUserRepos(userarg interface{}) {
	var username string
	currentUser, token, _ := auth.LoadToken(false)

	if currentUser == "" {
		util.CheckError(fmt.Errorf("This command requires login."))
	}

	if userarg == nil {
		username = currentUser
	} else {
		username = userarg.(string)
	}

	info, err := repo.GetRepos(username, token)
	util.CheckError(err)

	fmt.Printf("Repositories owned by %s\n", username)
	for idx, r := range info {
		fmt.Printf("%d:  %s\n", idx+1, r.Name)
	}
}

func listPubRepos() {
	repos, err := repo.GetRepos("", "")
	fmt.Printf("Public repositories\n")
	for idx, repoInfo := range repos {
		fmt.Printf("%d: %s [head: %s]\n", idx+1, repoInfo.Name, repoInfo.Head)
		fmt.Printf(" - %s\n", repoInfo.Description)
	}
	util.CheckError(err)
}

func main() {
	usage := `
GIN command line client

Usage:
	gin login    [<username>]
	gin create   [<name>] [-d <description>]
	gin upload   [<path>]
	gin download [<path>]
	gin repos    [<username>]
	gin info     [<username>]
	gin keys     [-v | --verbose]
	gin keys     --add <filename>
	gin public
`

	args, _ := docopt.Parse(usage, nil, true, "gin cli 0.0", false)

	switch {
	case args["login"].(bool):
		auth.Login(args["<username>"])
	case args["create"].(bool):
		createRepo(args["<name>"], args["<description>"])
	case args["upload"].(bool):
		upload(args["<path>"])
	case args["download"].(bool):
		download(args["<path>"])
	case args["info"].(bool):
		printAccountInfo(args["<username>"])
	case args["keys"].(bool):
		if args["--add"].(bool) {
			addKey(args["<filename>"])
		} else {
			printFullKeys := false
			if args["-v"].(bool) || args["--verbose"].(bool) {
				printFullKeys = true
			}
			printKeys(printFullKeys)
		}
	case args["repos"].(bool):
		listUserRepos(args["<username>"])
	case args["public"].(bool):
		listPubRepos()
	}
}
