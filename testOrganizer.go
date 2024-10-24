package main

import (
	"fmt"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"regexp"
	"github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/api"
	"net/http"
	"strings"
	"sort"
)

func main() {
	// define a struct for Repository object
	type Repository struct {
		Id            string `json:"id"`
		IsFork        bool `json:"isFork"`
		IsPrivate     bool `json:"isPrivate"`
		NameWithOwner string `json:"nameWithOwner"`
		Url           string `json:"url"`
	}

	// define a struct for each match from gh cli search command
	type Article struct {
		Path        string `json:"path"`
		Repository  Repository `json:"repository"`
		Url         string `json:"url"`
	}

	// define types used in our struct for our organized data output
	// define a test struct  - each Spec will have an array of these
	type Test struct {
		Name string
	}

	// define a spec struct - each repo will contain an array of these
	type Spec struct {
		Path string
		Url string
		Tests []Test
		Type string
	}

	// define a repo struct, that will contain an array of Specs in that repo
	type Repo struct {
		RepoName string
		Specs []Spec
	}
	// define a map of Repo's for our organized data
	organizedTests := make(map[string]Repo)
	totalTestCount := 0
	totalSpecCount := 0
	fileContentRequestCount := 0

	// create an array of our Article structs 
	var articles []Article

	// run the search command and store the result - wonder when the limit of 500 will become a problem
	fmt.Println("Executing command: gh search code org:BidPal --extension cy.js -L 500 --json repository,path,url")
	buff, _, err := gh.Exec("search", "code", "org:BidPal", "--extension", "cy.js", "-L", "500", "--json", "repository,path,url")

	if (err != nil) {
			fmt.Printf("Error running gh search command: %s", err)
	}

	xerr := json.Unmarshal([]byte(buff.Bytes()), &articles)
	if xerr != nil {
		fmt.Printf("Error unmarshalling search results to struct array: %s", xerr)
	}

	fmt.Printf("Search found %d specs. Processing...\n", len(articles))

	// we now have an array of specs in articles
	// lets loop thru it searching each for tests, and writing those to our csv
	for _, spec := range articles {
		// strip "BidPal/phaas-" from repo name we will write to the file
		repoName := strings.SplitAfterN(spec.Repository.NameWithOwner, "-", 2)[1]

		// create an array of Specs, later used as organizedTests[repoName].Specs
		var repoSpecs []Spec
		// if repo already in organizedTests, need to initialize repoSpecs with existing specs before adding to it
		if val, ok := organizedTests[repoName]; ok {
			// logging to help with debugging
			// fmt.Printf("Repo %s already exists in organized tests!\n", val.RepoName)

			// populate repoSpecs with existing specs for the repo before adding new
			repoSpecs = val.Specs
		}
		// now carry on - repoSpecs will be empty if organizedTests[repoName] didn't exist already

		// regex to match "smoke" or "integration" literally from spec path
		typeRegex, err := regexp.Compile(`smoke|integration`) 
		if err != nil {
			fmt.Println(err)
		}
		specType := typeRegex.FindString(spec.Path)
		// default to integration if niether smoke nor integraiton found in path
		if (len(specType) == 0) {
			specType = "integration"
		}
		// create a Spec for the current match that will be appended to repoSpecs later
		currSpec := Spec{
			Path: spec.Path,
			Url: spec.Url,
			Type:  specType,
		}

		// fetch the content of the current spec
		path := fmt.Sprintf("repos/%s/contents/%s", spec.Repository.NameWithOwner, spec.Path)
			opts := api.ClientOptions{
			Headers:   map[string]string{"Accept": "application/vnd.github.v3.raw","X-GitHub-Api-Version": "2022-11-28"},
		}
		client, err := api.NewRESTClient(opts)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Fetching content of %s from %s via gh api\n", spec.Path, repoName)
		response, err := client.Request(http.MethodGet, path, nil)
		fileContentRequestCount++
		if err != nil {
			fmt.Println(err)
		}
		defer response.Body.Close()

    resBody, err := io.ReadAll(response.Body)
		if err != nil {
        fmt.Printf("Cannot parse GET content response: %v\n", err)
        return
    }
		// store string value of response (full file content for spec)
		fileContent := string(resBody)
		// create regexp object we'll use to filter out search results that aren't actually its
		// this pattern matches on whitespace, followed by "it(" followed by either `, ", or ',
		// then it captures everything after the single/double quote or backtick up to the next single/double quote or backtick
		// which should give us the test name (\x60 is backtick)
		re, err := regexp.Compile(`\s+it\(["'\x60]([^"'\x60]+)["'\x60]`) 
		if err != nil {
			fmt.Println(err)
		}
		
		matches := re.FindAllStringSubmatch(fileContent, -1)
		for _, match := range matches {
			currSpec.Tests = append(currSpec.Tests, Test{
				Name: match[1],
			})
			// increment test count and log test added to spec
			totalTestCount++
			// fmt.Printf("Added test %s \nto spec %s \n", match[1], currSpec.Path)
		}
		repoSpecs = append(repoSpecs, currSpec)
		// log when spec is added
		// fmt.Printf("Added spec %s \nto repo %s \n", currSpec.Path, repoName)

		// add/update repo in organizedTests
		organizedTests[repoName] = Repo{
			RepoName: repoName,
			Specs: repoSpecs,
		}

		// log when repo is added/updated in organizedTests struct
		// fmt.Printf("Added/Updated repo %s\n", repoName)
	}
	// done processing - log totals
	fmt.Printf("%d requests made to github api to fetch file contents\n", fileContentRequestCount)
	fmt.Printf("%d tests were found in %d repos and written to ./organizedTests.csv\n", totalTestCount, len(organizedTests))
	// ok now how do I write that nice struct out to a csv file?
	// start by creating the array of arrays of strings I'd like to write to the file
	var result [][]string
	// first el in the array will be our header row
	header := []string{"Repo","Spec","Type","Test"}
	result = append(result,header)
	// gonna store some summary data here and append it at the end of the file
	var summaryData [][]string
	// create a slice of the keys in oranizedTests, and sort it
	repos := make([]string, 0, len(organizedTests))
	for k := range organizedTests {
		repos = append(repos, k)
	}
	sort.Strings(repos)
	// then iterate through this sorted list of keys writing tests to the csv
	for _, repo := range repos {
		repoTestCount := 0
		repoName := organizedTests[repo].RepoName
		// loop through specs for each repo
		for _, spec := range organizedTests[repo].Specs {
			totalSpecCount++
			specPath := spec.Path
			specUrl := spec.Url
			// loop through tests for each spec
			for _, test := range spec.Tests {
				// increment count of tests for repo summary data row
				repoTestCount++
				testName := test.Name
				// add an el to our array for the test in the current loop - spec path will hyperlink to spec
				row := []string{repoName,fmt.Sprintf("=HYPERLINK(%s,%s)", fmt.Sprintf("\"%s\"", specUrl), fmt.Sprintf("\"%s\"", specPath)), spec.Type, testName}
				result = append(result, row)
			}
		}
		// store summary data for the repo
		summaryData = append(summaryData, []string{
			fmt.Sprintf("Summary Data for repo %s:", repoName),
			fmt.Sprintf("Spec Count: %d", len(organizedTests[repo].Specs)),
			fmt.Sprintf("Test Count: %d", repoTestCount),
		})
	}

	// append summary data to end of csv data
	result = append(result, summaryData...)
	result = append(result, []string{
			fmt.Sprintf("Total Repo Count: %d", len(organizedTests)),
			fmt.Sprintf("Total Spec Count: %d", totalSpecCount),
			fmt.Sprintf("Total Test Count: %d", totalTestCount),
		})

	// Create a new csv file
	f, err := os.Create("organizedTests.csv")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer f.Close() // Ensure the file is closed when the function exits

	// Create a CSV writer that will write to that file
	writer := csv.NewWriter(f)
	
	// Write data to the file
	writeErr := writer.WriteAll(result)

	if writeErr != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
}