package main

import (
	"testing"
)

func TestFetchSpecs(t *testing.T) {
    articles := fetchSpecs()
		numFetched := len(articles)
    if numFetched == 0 {
        t.Fatalf("Length of articles fetched is %d. Expected > 0", numFetched)
    }
}

func TestInitSpec(t *testing.T) {
	var mockRepository = Repository{
		ID: "MDEwOlJlcG9zaXRvcnk5NDA4NjQ1MA==",
		IsFork: false, 
		IsPrivate: true, 
		NameWithOwner: "BidPal/phaas-org-ui", 
		URL: "https://github.com/BidPal/phaas-org-ui",
	}

	var mockSearchResult = Article{
		Path: "cypress/e2e/integration/create-users.cy.js",
		Repository: mockRepository,
		URL: "https://github.com/BidPal/phaas-org-ui/blob/337193860c279e78818d67b2458e559333d0e86f/cypress/e2e/integration/create-users.cy.js",
	}
	spec := initSpec(mockSearchResult);
	if (spec.Path != mockSearchResult.Path) {
		t.Fatalf(`Expected spec.Path %s to equal %s`, spec.Path, mockSearchResult.Path)
	}
	if (spec.URL != mockSearchResult.URL) {
		t.Fatalf(`Expected spec.URL %s to equal %s`, spec.URL, mockSearchResult.URL)
	}
	if (spec.Type != "integration") {
		t.Fatalf(`Expected spec.Type %s to equal "integration"`, spec.URL)
	}
}