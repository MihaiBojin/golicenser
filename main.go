package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"encoding/base64"
	"encoding/csv"

	"github.com/google/go-github/v27/github"
	"golang.org/x/oauth2"
)

func main() {
	// parse arguments
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) == 0 {
		fmt.Println("Usage: golicenser LICENSE-FILE [GITHUB-PERSONAL-TOKEN]")
		fmt.Println()
		os.Exit(0)
	}

	// open file
	licenses := argsWithoutProg[0]
	file, err := os.Open(licenses)
	fatalIfErr(err)
	defer logError(file.Close)

	// retrieve Oauth token, if defined
	var client *http.Client
	if len(argsWithoutProg) > 1 {
		fmt.Println("Authenticating using the provided Oauth token...")
		client = GetAuthenticatedClient(argsWithoutProg[1])
	}

	// initialize the output writer
	reportFilename := strings.Replace(licenses, filepath.Ext(licenses), "", 1) + "-processed.csv"
	outFile, err := os.Create(reportFilename)
	fatalIfErr(err)
	csvWriter := csv.NewWriter(bufio.NewWriter(outFile))
	err = csvWriter.Write([]string{"Repository", "License Name", "License URL", "License Contents"})
	fatalIfErr(err)

	// read all lines from the file
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		repo := scanner.Text()
		fmt.Printf("Processing %s\n", repo)

		var licenseName string
		var licenseURL string
		var contents string
		license, ok := GetGitHubLicense(client, repo)
		if ok {
			licenseName = license.License.GetName()
			licenseURL = license.GetHTMLURL()
			if license.License.GetKey() == "other" {
				bytes, err := base64.StdEncoding.DecodeString(license.GetContent())
				logNonNilError(err)
				contents = string(bytes)
			}
		} else {
			licenseName = "?"
		}

		err := csvWriter.Write([]string{repo, licenseName, licenseURL, contents})
		logNonNilError(err)
	}
	csvWriter.Flush()

	// finish exceptionally, if errors were detected
	fatalIfErr(csvWriter.Error())
	fatalIfErr(scanner.Err())

	// let the user know where the report is
	fmt.Printf("Saved report at: %s\n", reportFilename)
}

// GetAuthenticatedClient returns an HTTP client which can authenticate using the provided token
func GetAuthenticatedClient(token string) *http.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	return oauth2.NewClient(context.Background(), ts)
}


// GetGitHubLicense makes a best effort attempt to determine the project's github license
func GetGitHubLicense(httpClient *http.Client, repo string) (license *github.RepositoryLicense, ok bool) {
	parts := strings.Split(repo, "/")
	if parts[0] != "github.com" {
		// can only use this function on github.com repositories
		return nil, false
	}

	// retrieve the license
	client := github.NewClient(httpClient)
	repoLicense, _, err := client.Repositories.License(context.Background(), parts[1], parts[2])
	if err != nil {
		logNonNilError(err)
		return nil, false
	}

	return repoLicense, true
}

// fatalIfErr throw fatal if error was detected
func fatalIfErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// logError logs any errors returned by the action
func logError(action func() error) {
	logNonNilError(action())
}

// logNonNilError logs the error if not nil
func logNonNilError(err error) {
	if err != nil {
		log.Printf("Unexpected error: %v", err)
	}
}
