package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// TestExtractText verifies the text extraction logic.
func TestExtractText(t *testing.T) {
	html := `
	<html>
		<head>
			<title>Test Page</title>
			<style>.dark{color: #333;}</style>
		</head>
		<body>
			<header><h1>Main Title</h1></header>
			<nav><a href="/nav">Nav Link</a></nav>
			<p>This is the first paragraph.</p>
			<div>Here is a div with more text.</div>
			<script>alert("hello");</script>
			<footer><p>Copyright info</p></footer>
		</body>
	</html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	extracted := extractText(doc)
	expected := "Main Title This is the first paragraph. Here is a div with more text."

	if extracted != expected {
		t.Errorf("extractText() failed:\nGot:  %s\nWant: %s", extracted, expected)
	}
}

// TestFetchAndParse verifies the fetching, parsing, and link extraction logic
// using a mock HTTP server.
func TestFetchAndParse(t *testing.T) {
	// 1. Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve different content based on the request path
		switch r.URL.Path {
case "/page1":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintln(w, `
				<html>
					<head><title>Page 1</title></head>
					<body>
						<p>Welcome to page 1.</p>
						<a href="/page2">Go to Page 2</a>
						<a href="https://example.com/external">External Link</a>
						<a href="#fragment">Fragment Link</a>
						<a href="mailto:test@example.com">Mail Link</a>
					</body>
				</html>`)
		case "/page2":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintln(w, `<html><head><title>Page 2</title></head><body><p>This is page 2.</p></body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// 2. Create a client that uses the mock server
	client := server.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 3. Call the function to be tested
	page1URL := server.URL + "/page1"
	doc, links, err := fetchAndParse(ctx, client, page1URL)

	// 4. Assert the results
	if err != nil {
		t.Fatalf("fetchAndParse() returned an error: %v", err)
	}

	// Check the document content
	if doc.URL != page1URL {
		t.Errorf("doc.URL is incorrect. got %q, want %q", doc.URL, page1URL)
	}
	if doc.Title != "Page 1" {
		t.Errorf("doc.Title is incorrect. got %q, want %q", doc.Title, "Page 1")
	}
	expectedText := "Welcome to page 1. Go to Page 2 External Link Fragment Link Mail Link"
	if doc.Text != expectedText {
		t.Errorf("doc.Text is incorrect. got %q, want %q", doc.Text, expectedText)
	}

	// Check the extracted links
	if len(links) != 2 {
		t.Fatalf("Expected 2 links, but got %d. Links: %v", len(links), links)
	}

	expectedLink1 := server.URL + "/page2"
	if links[0] != expectedLink1 {
		t.Errorf("Link 1 is incorrect. got %q, want %q", links[0], expectedLink1)
	}

	expectedLink2 := "https://example.com/external"
	if links[1] != expectedLink2 {
		t.Errorf("Link 2 is incorrect. got %q, want %q", links[1], expectedLink2)
	}
}
