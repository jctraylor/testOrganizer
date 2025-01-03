package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/api"
)

// Repository struct holds data in Respository object included with each search result
type Repository struct {
	ID            string `json:"id"`
	IsFork        bool `json:"isFork"`
	IsPrivate     bool `json:"isPrivate"`
	NameWithOwner string `json:"nameWithOwner"`
	URL           string `json:"url"`
}

// Article is a struct for each match from gh cli search command
type Article struct {
	Path        string `json:"path"`
	Repository  Repository `json:"repository"`
	URL         string `json:"url"`
}

// define types used in our struct for our organized data output

// Test struct  - each Spec will have an array of these
type Test struct {
	Describe string
	Name string
	TestSkipped bool
	DescribeSkipped bool
}

// Spec struct - each repo will contain an array of these
type Spec struct {
	Path string
	URL string
	Tests []Test
	Type string
}

//Repo struct, that will contain an array of Specs in a given repo
type Repo struct {
	RepoName string
	Specs []Spec
}

// initialize these counts globally and increment where needed
var totalTestCount = 0
var totalWIPTestCount = 0
var totalSpecCount = 0
var totalSkippedTestCount = 0
var repoTestCount = 0
var repoSkippedTestCount = 0
var repoWIPTestCount = 0
var fileContentRequestCount = 0
var jsSpecCount = 0
var tsSpecCount = 0

func main() {
	start := time.Now()
	// define a map of Repo's for our organized data
	organizedTests := make(map[string]Repo)
	// run the gh cli search command to fetch specs
	specs := fetchSpecs();

	// we now have an array of specs
	// lets loop thru it searching each for tests, and writing those to our csv
	for _, spec := range specs {
		currSpec := initSpec(spec)
		// create an array of Specs, later used as organizedTests[repoName].Specs
		var repoSpecs []Spec
		// strip "BidPal/phaas-" from repo name we will write to the file
		repoName := strings.SplitAfterN(spec.Repository.NameWithOwner, "-", 2)[1]
		// if repo already in organizedTests, need to initialize repoSpecs with existing specs before adding to it
		if val, ok := organizedTests[repoName]; ok {
			// logging to help with debugging
			// fmt.Printf("Repo %s already exists in organized tests!\n", val.RepoName)

			// populate repoSpecs with existing specs for the repo before adding new
			repoSpecs = val.Specs
		}
		// now carry on - repoSpecs will be empty if organizedTests[repoName] didn't exist already
		// fetch the content of the current spec
		fileContent := fetchSpecContent(spec, repoName)
		fileContentRequestCount++
		describeSegments := splitIntoDescribes((fileContent))
		for index, segment := range describeSegments {
			// this is kind of odd but gonna start processing on index 1
			if (index == 0) {
				continue
			}
			prevIndexText := describeSegments[index-1]
			slice := []string{prevIndexText}
			isThisDescribeSkipped := isMatchSkipped(slice)
			// logging to help with debugging
			// fmt.Printf("Processing spec %s", spec.Path)
			// TODO: I think this pattern needs to include describe - it's finding too much that isn't what we want
			describeText := getRegexMatches(segment, `["'\x60]([^"'\x60]+)["'\x60]`)[0]
			// create regexp objects we'll use to find it's and describes
			// this pattern matches on whitespace OR 'x', followed by "it(" or "describe(" followed by either `, ", or ',
			// then it captures everything after the single/double quote or backtick up to the next single/double quote or backtick
			// which should give us the test/describe text (\x60 is backtick)
			foundTests := getRegexMatches(segment, `[\sx]+it(?:.skip)?\(["'\x60]([^"'\x60]+)["'\x60]`)
			for _, test := range foundTests {
				isTestSkipped := isMatchSkipped(test)
				if (isTestSkipped || isThisDescribeSkipped) {
					totalSkippedTestCount++
				}
				if (currSpec.Type == "WIP") {
					totalWIPTestCount++
				}
				currSpec.Tests = append(currSpec.Tests, Test{
					Describe: describeText[1],
					Name: test[1],
					TestSkipped: isTestSkipped,
					DescribeSkipped: isThisDescribeSkipped,
				})
				// increment test count and log test added to spec
				totalTestCount++
				// fmt.Printf("Added test %s \nto spec %s \n", match[1], currSpec.Path)
			}
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

	// done processing raw data - log totals
	fmt.Printf("Fetched the content of %d specs via github api\n", fileContentRequestCount)
	fmt.Printf("%d tests were found in %d repos and written to ./organizedTests.csv\n", totalTestCount, len(organizedTests))

	// ok now how do I write that nice struct out to a csv file?
	// start by creating the array of arrays of strings I'd like to write to the file
	var csvRows = buildCSVRows(organizedTests)
	// Create a new csv file
	writeCSV(csvRows)
	elapsed := time.Since(start)
	fmt.Println(elapsed)
}

func initSpec(spec Article) Spec {
		// regex to match "smoke", "integration", or "wip" literally from spec path - case insensitive
		typeRegex, err := regexp.Compile("(?i)smoke|integration|wip") 
		if err != nil {
			fmt.Println(err)
		}
		specType := typeRegex.FindString(spec.Path)
		// default to integration if niether smoke nor integraiton found in path
		if (len(specType) == 0) {
			specType = "integration"
		}
		// capitalize type if wip
		if (specType == "wip") {
			specType = strings.ToUpper(specType)
		}
		// return a Spec for the current match that will be appended to repoSpecs later
		return Spec{
			Path: spec.Path,
			URL: spec.URL,
			Type:  specType,
		}
}
// run the gh cli search command and return the result
func runSearchCommand(extension string) []Article {
	var searchResults []Article

	// wonder when the limit of 500 will become a problem
	fmt.Printf("Executing command: gh search code org:BidPal --extension %s -L 500 --json repository,path,url\n", extension)
	buff, _, err := gh.Exec("search", "code", "org:BidPal", "--extension", extension, "-L", "500", "--json", "repository,path,url")

	if (err != nil) {
			fmt.Printf("Error running gh search command: %s", err)
	}

	// TODO: xerr is bad?
	xerr := json.Unmarshal([]byte(buff.Bytes()), &searchResults)
	if xerr != nil {
		fmt.Printf("Error unmarshalling search results to struct array: %s", xerr)
	}

	return searchResults;
}


func fetchSpecs() []Article {
	jsSpecs := runSearchCommand("cy.js")
	tsSpecs := runSearchCommand("cy.ts")

	// create an array of our Article structs to hold everything we fetch
	var articles []Article

	jsSpecCount += len(jsSpecs)
	tsSpecCount += len(tsSpecs)

	fmt.Printf("Search found %d specs. Processing...\n", jsSpecCount + tsSpecCount)

	// now put all the specs ts and js into articles and continue
	// seems like there should be a more efficient way to do this
	articles = append(articles, jsSpecs...)
	articles = append(articles, tsSpecs...)
	return articles
}

func fetchSpecContent(spec Article, repoName string) string {
	path := fmt.Sprintf("repos/%s/contents/%s", spec.Repository.NameWithOwner, spec.Path)
	fmt.Printf("Fetching content of %s from %s via gh api\n", spec.Path, repoName)
	opts := api.ClientOptions{
		Headers:   map[string]string{"Accept": "application/vnd.github.v3.raw","X-GitHub-Api-Version": "2022-11-28"},
	}
	client, err := api.NewRESTClient(opts)
	if err != nil {
		fmt.Println(err)
	}
	response, err := client.Request(http.MethodGet, path, nil)
	if err != nil {
		fmt.Println(err)
	}
	defer response.Body.Close()

	resBody, err := io.ReadAll(response.Body)
	if err != nil {
			fmt.Printf("Cannot parse GET content response: %v\n", err)
			// update so i can return an err
			// return err;
	}
	// return string value of response (full file content for spec)
	return string(resBody)
}

func splitIntoDescribes(fileContent string) []string {
	// found that tests inside of describe.skip were missing - so adding handling for that case
	dotSkippedDescribes := strings.SplitAfter(fileContent, "describe.skip(")
	allOtherDescribes := strings.SplitAfter(fileContent, "describe(")
	if len(dotSkippedDescribes) > 1 && len(allOtherDescribes) > 1 {
		// haven't run into this so wanna log if we do
		fmt.Println("Found skipped and not skipped describes - how'd we handle that?")
		// append segments except for first index in dotskipped to allothers - and return that
		return append(allOtherDescribes, dotSkippedDescribes[1:]...);
	} else if len(dotSkippedDescribes) == 1 {
		return allOtherDescribes
	} else {
		return dotSkippedDescribes
	}
	
}

func getRegexMatches(str string, pattern string) [][]string {
	testsRegexp, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Println(err)
	}
	return testsRegexp.FindAllStringSubmatch(str, -1)
}

func createCSVRowForTest(spec Spec, repoName string, test Test) []string {
	specPath := spec.Path
	specURL := spec.URL
	// increment count of tests for repo summary data row
	repoTestCount++
	if (test.TestSkipped || test.DescribeSkipped) {
		repoSkippedTestCount++
	}
	if (spec.Type == "WIP") {
		repoWIPTestCount++
	}
	//  - spec path will hyperlink to spec
	row := []string{repoName,fmt.Sprintf("=HYPERLINK(%s,%s)", fmt.Sprintf("\"%s\"", specURL), fmt.Sprintf("\"%s\"", specPath)), spec.Type, test.Describe, test.Name, fmt.Sprintf("%t", test.DescribeSkipped), fmt.Sprintf("%t", test.TestSkipped)}
	return row;
}

func buildCSVRows(organizedTests map[string]Repo) [][]string {
	var csvRows [][]string
	var summaryData [][]string
 	// first el in the array will be our header row
	header := []string{"Repo","Spec","Type","Describe","Test","Describe Skipped","Test Skipped"}
	csvRows = append(csvRows,header)
	
	// create a slice of the keys in oranizedTests, and sort it
	repos := make([]string, 0, len(organizedTests))
	for k := range organizedTests {
		repos = append(repos, k)
	}
	sort.Strings(repos)
	// then iterate through this sorted list of keys writing tests to the csv
	for _, repo := range repos {
		// reset these repo counts each time we start processing a new repo
		repoTestCount = 0
		repoSkippedTestCount = 0
		repoWIPTestCount = 0
		repoName := organizedTests[repo].RepoName
		// loop through specs for each repo
		for _, spec := range organizedTests[repo].Specs {
			totalSpecCount++
			for _, test := range spec.Tests {
				// add a row to our csv data for each test
				csvRows = append(csvRows, createCSVRowForTest(spec, repoName, test))
			}
		}

		// store summary data for the repo
		summaryData = append(summaryData, []string{
			fmt.Sprintf("Summary Data for repo %s:", repoName),
			fmt.Sprintf("Spec Count: %d", len(organizedTests[repo].Specs)),
			fmt.Sprintf("Test Count: %d", repoTestCount),
			fmt.Sprintf("Skipped Test Count: %d", repoSkippedTestCount),
			fmt.Sprintf("WIP Test Count: %d", repoWIPTestCount),
		})
	}

	// append summary data to end of csv data
	csvRows = append(csvRows, summaryData...)
	csvRows = append(csvRows, []string{
			fmt.Sprintf("Total Repo Count: %d", len(organizedTests)),
			fmt.Sprintf("Total Spec Count: %d (%d js files / %d ts files)", totalSpecCount, jsSpecCount, tsSpecCount),
			fmt.Sprintf("Total Test Count: %d", totalTestCount),
			fmt.Sprintf("Total Skipped Test Count: %d", totalSkippedTestCount),
			fmt.Sprintf("Total WIP Test Count: %d", totalWIPTestCount),
		})

	return csvRows
}

func writeCSV(csvRows [][]string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fileName := "organizedTests-" + timestamp + ".csv"
	f, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer f.Close() // Ensure the file is closed when the function exits

	// Create a CSV writer that will write to that file
	writer := csv.NewWriter(f)
	
	// Write data to the file
	writeErr := writer.WriteAll(csvRows)

	if writeErr != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
}

// TODO: prob better if this just takes a string instead of a slice of strings
func isMatchSkipped(match []string) bool {
	re, err := regexp.Compile(`xit|xdescribe|it.skip|describe.skip`) 
			if err != nil {
				fmt.Println(err)
			}
			return re.Match([]byte(match[0]))
}
