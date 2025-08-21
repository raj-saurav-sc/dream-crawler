//go:build integration

package main

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestFetchAndParse_Integration performs a test against a live external URL.
// It is separated from unit tests by a build tag and should be run explicitly.
// To run: go test -v -tags=integration ./...
func TestFetchAndParse_Integration(t *testing.T) {
	// 1. Define the target URL and create a client
	// We use the default client which can make real network requests.
	targetURL := "https://hostman.com/tutorials/install-apache-kafka-on-ubuntu-22-04/"
	client := http.DefaultClient
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 2. Call the function to be tested
	doc, links, err := fetchAndParse(ctx, client, targetURL)

	// 3. Assert the results
	if err != nil {
		t.Fatalf("fetchAndParse() returned an error for a live URL: %v", err)
	}

	// Check the document content
	if doc.URL != targetURL {
		t.Errorf("doc.URL is incorrect. got %q, want %q", doc.URL, targetURL)
	}
	if doc.Title != "Example Domain" {
		t.Errorf("doc.Title is incorrect. got %q, want %q", doc.Title, "Example Domain")
	}

	expectedTextPrefix := "Example Domain This domain is for use in illustrative examples in documents."
	if !strings.HasPrefix(doc.Text, expectedTextPrefix) {
		t.Errorf("doc.Text does not start with the expected prefix.\nGot:  %q\nWant prefix: %q", doc.Text, expectedTextPrefix)
	}

	// Check for the extracted link
	if len(links) != 1 {
		t.Fatalf("Expected 1 link, but got %d. Links: %v", len(links), links)
	}

	expectedLink := "https://www.iana.org/domains/example"
	if links[0] != expectedLink {
		t.Errorf("Link is incorrect. got %q, want %q", links[0], expectedLink)
	}
}
