package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"encoding/base64"
	"encoding/csv"

	"github.com/google/go-github/v27/github"
	"golang.org/x/oauth2"
)

// solution based on this resource: https://stackoverflow.com/a/52600147/7169815
var reNewLineReplacer = regexp.MustCompile(`\r\n|[\r\n\v\f\x{0085}\x{2028}\x{2029}]`)

// License represents license data for a single dependency
type License struct {
	Licenses string `json:"licenses"`
	Repository string `json:"repository"`
	Publisher string `json:"publisher"`
	Name string `json:"name"`
	Version string `json:"version"`
	Copyright string `json:"copyright"`
	LicenseText string `json:"licenseText"`
	LicenseFile string `json:"licenseFile"`
}

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
	reportFilename := strings.Replace(licenses, filepath.Ext(licenses), "", 1)
	outFile, err := os.Create(reportFilename + "-report.csv")
	fatalIfErr(err)

	// write the CSV header
	csvWriter := csv.NewWriter(bufio.NewWriter(outFile))
	err = csvWriter.Write([]string{"Repository", "Licenses", "Name", "Publisher", "Version", "Copyright", "License File", "License Text"})
	fatalIfErr(err)

	// track dependencies
	deps := make(map[string]License)

	// read all lines from the file
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		repo := scanner.Text()
		fmt.Printf("Processing %s\n", repo)

		var licenses string
		var licenseFile string
		var licenseText string
		var copyright string
		owner, repoName, isGitHubRepo := getRepoPublisherAndName(repo)
		license, licenseIdentified := GetGitHubLicense(client, owner, repoName)
		if !isGitHubRepo || !licenseIdentified {
			// something's fishy, check if the repository exists
			_, repoExists := GetGitHubRepo(client, owner, repoName)
			if !repoExists {
				log.Printf("Repository doesn't seem to exist; skipping: %s", repo)
				continue
			}

			// else, it means we could not detect a license for this repository
			licenses = "Unknown"
		} else {
			licenses = license.License.GetName()
			licenseFile = license.GetHTMLURL()
			if license.License.GetKey() == "other" {
				bytes, err := base64.StdEncoding.DecodeString(license.GetContent())
				logNonNilError(err)
				licenseText = cleanNewLines(strings.TrimSpace(string(bytes)))
				copyright, _ = extractCopyrightNotices(licenseText)
			}
		}

		// write CSV
		err := csvWriter.Write([]string{repo, licenses, repoName, owner, "", copyright, licenseFile, licenseText})
		logNonNilError(err)

		// write JSON
		deps[repo] = License{
			Licenses:    licenses,
			Repository:  repo,
			Name:        repoName,
			Publisher:   owner,
			Version:     "",
			Copyright:   "",
			LicenseFile: licenseFile,
			LicenseText: licenseText,
		}
	}
	csvWriter.Flush()

	// store the JSON report
	jsonFile, jsonErr := json.MarshalIndent(deps, "", "  ")
	fatalIfErr(jsonErr)
	jsonWriteErr := ioutil.WriteFile(reportFilename + "-report.json", jsonFile, 0744)
	fatalIfErr(jsonWriteErr)

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
func GetGitHubLicense(httpClient *http.Client, owner string, repository string) (license *github.RepositoryLicense, ok bool) {
	// retrieve the license
	client := github.NewClient(httpClient)
	repoLicense, _, err := client.Repositories.License(context.Background(), owner, repository)
	if err != nil {
		logNonNilError(err)
		return nil, false
	}

	return repoLicense, true
}

// GetGitHubRepo retrieves a repository's metadata
func GetGitHubRepo(httpClient *http.Client, owner string, repository string) (license *github.Repository, ok bool) {
	// retrieve the license
	client := github.NewClient(httpClient)
	repoMeta, _, err := client.Repositories.Get(context.Background(), owner, repository)
	if err != nil {
		logNonNilError(err)
		return nil, false
	}

	return repoMeta, true
}

// cleanNewLines ensure all newlines are represented by a '\n' character
func cleanNewLines(licenseContents string) string {
	return reNewLineReplacer.ReplaceAllString(licenseContents, "\n")
}

// extractCopyrightNotices extracts copyright notices from a license
func extractCopyrightNotices(licenseContents string) (string, bool) {
	// assuming copyright notices are separated by a blank line
	lines := strings.Split(licenseContents, "\n\n")

	// process the license contents for copyright notices
	copyrightLines := make(map[string]bool)
	for _, l := range lines {
		// remove leading/trailing spaces
		l = strings.TrimSpace(l)

		if !strings.Contains(l,"Copyright") {
			continue
		}

		// merge copyright notices which existing on multiple lines
		license := strings.Join(strings.Split(l, "\n"), ". ")

		// add the license
		// using a map to ensure duplicates are excluded
		copyrightLines[license] = true
	}

	// did not detect copyright notices
	if len(copyrightLines) == 0 {
		return "", false
	}

	// retrieve all unique copyright lines
	keys := make([]string, 0, len(copyrightLines))
	for k := range copyrightLines {
		keys = append(keys, k)
	}

	// return each individual notice as a new line
	return strings.Join(keys, "\n"), true
}

// getRepoPublisherAndName returns the owner and repository name of a GitHub repo
func getRepoPublisherAndName(repo string) (string, string, bool) {
	parts := strings.Split(repo, "/")
	if parts[0] != "github.com" {
		// can only use this function on github.com repositories
		return "", "", false
	}
	return parts[1], parts[2], true
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
