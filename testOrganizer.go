package main

import (
	"fmt"
	"encoding/csv"
	"encoding/json"
	// "io"
	"os"
	"regexp"
	"github.com/cli/go-gh/v2"
	// "github.com/cli/go-gh/v2/pkg/api"
)

func main() {
	// setup all the types needed to unmarshal our json array from its.json
	// define struct for match object inside of textMatch object
	type Match struct {
		Indices []int `json:"indices"`
		Text     string `json:"text"`
	}

	// define a struct for textMatch object 
	type TextMatch struct {
			Fragment  string `json:"fragment"`
			Matches		[]Match `json:"matches"`
			Property  string `json:"property"`
			Type 			string `json:"type"`
	}

	// defined a struct for Repository object
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
		TextMatches []TextMatch `json:"textMatches"`
		Url         string `json:"url"`
	}

	// define types used in our struct for our organized data output
	// define a test struct  - each Spec will have an array of these
	type Test struct {
		Name string
		// indices []int
	}

	// define a spec struct - each repo will contain an array of these
	type Spec struct {
		Path string
		Url string
		Tests []Test
	}

	// define a repo struct, that will contain an array of Specs in that repo
	type Repo struct {
		RepoName string
		Specs []Spec
	}

	// finally, define a map of Repo's for our organized data
	organizedTests := make(map[string]Repo)
	testCount := 0;

	// not using its.json anymore - now this go program runs the search, and parses the results without reading from a file
	// but I may have to keep this around for debugging cus I can't run the gh search while debugging
	// jsonFile, err := os.Open("its.json");
	// if err != nil {
  //   fmt.Println(err)
  // }
	// fmt.Println("Successfully Opened its.json")
	// defer the closing of our jsonFile so that we can parse it later on
	// defer jsonFile.Close()
	// byteValue, _ := io.ReadAll(jsonFile)

	// create an array of our Article structs 
	var articles []Article

	// unmarshal our json file into the articles array - again - not using this anymore but may keep for debugging
	// json.Unmarshal([]byte(byteValue), &articles)

	// run the search command and store the result - wonder when the limit of 500 will become a problem
	fmt.Println("Executing command: gh search code it org:BidPal --extension cy.js -L 500 --json repository,path,textMatches,url")
	buff, _, err := gh.Exec("search", "code", "it", "org:BidPal", "--extension", "cy.js", "-L", "500", "--json", "repository,path,textMatches,url")

	if (err != nil) {
			fmt.Printf("Error running gh search command: %s", err)
	}

	xerr := json.Unmarshal([]byte(buff.Bytes()), &articles)
	if xerr != nil {
		fmt.Printf("Error unmarshalling search results to struct array: %s", xerr)
	}

	fmt.Printf("Search found matches in %d specs. Processing...\n", len(articles))

	// we now have an array of objects (matches) to organize
	// lets loop thru it and make an object that is more useful for us
	for _, match := range articles {
		repoName := match.Repository.NameWithOwner

		// create an array of Specs, later used as organizedTests[repoName].Specs
		var repoSpecs []Spec;
		// if repo already in organizedTests, need to initialize repoSpecs with existing specs before adding to it
		if val, ok := organizedTests[repoName]; ok {
			// logging to help with debugging
			// fmt.Printf("Repo %s already exists in organized tests!\n", val.RepoName);

			// populate repoSpecs with existing specs for the repo before adding new
			repoSpecs = val.Specs
		}
		// now carry on - repoSpecs will be empty if organizedTests[repoName] didn't exist already

		// create a Spec for the current match that will be appended to repoSpecs later
		currSpec := Spec{
			Path: match.Path,
			Url: match.Url,
		}

		// loop through textMatches array in current match, appending to currSpec.Tests as we go
		for _, textMatch := range match.TextMatches {
			// create regexp object we'll use to filter out search results that aren't actually its
			// this pattern matches on whitespace, followed by "it(" followed by either `, ", or ',
			// then it captures everything after the single/double quote or backtick up to the next single/double quote or backtick
			// which should give us the test name (\x60 is backtick)
			re, err := regexp.Compile(`\s+it\(["'\x60]([^"'\x60]+)["'\x60]`) 
			if err != nil {
				fmt.Println(err)
			}
			legit := re.FindStringSubmatch(textMatch.Fragment)
			if (len(legit) > 1) {
				currSpec.Tests = append(currSpec.Tests, Test{
					Name: legit[1],
				})
				// increment test count and log test added to spec
				testCount++
				// logging for debugging - could write to a log file instead
				// fmt.Printf("Added test %s \nto spec %s \n", legit[1], currSpec.Path)
			} // else {
				// log when matches are not added as test to spec - could write to a log file instead
				// fmt.Printf("Did not add match %s \nto spec %s \n", textMatch.Fragment, currSpec.Path)
			// }
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
	fmt.Printf("%d tests were found in %d repos and written to ./organizedTests.csv\n", testCount, len(organizedTests));

	// ok now how do I write that nice struct out to a csv file?
	// start by creating the array of arrays of strings I'd like to write to the file
	var result [][]string
	// first el in the array will be our header row
	header := []string{"Repo","Spec","Test","Url"}
	result = append(result,header)
	// loop through repos in organized tests 
	for _, repo := range organizedTests {
		repoName := repo.RepoName;
		// loop through specs for each repo
		for _, spec := range repo.Specs {
			specPath := spec.Path
			specUrl := spec.Url
			// loop through tests for each spec
			for _, test := range spec.Tests {
				testName := test.Name
				// add an el to our array for the test in the current loop
				row := []string{repoName,specPath,testName,specUrl}
				result = append(result, row)
			}
		}
	}

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