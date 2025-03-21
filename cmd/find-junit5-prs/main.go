package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// JUnit5PR represents a GitHub pull request related to JUnit 5 migration
type JUnit5PR struct {
	Title      string   `json:"title"`
	URL        string   `json:"url"`
	Repository string   `json:"repository"`
	State      string   `json:"state"`
	Author     string   `json:"author"`
	Labels     []string `json:"labels"`
	Body       string   `json:"body"`
	CreatedAt  string   `json:"createdAt"`
}

// SearchResult holds the PRs found in the search
type SearchResult struct {
	PRs []JUnit5PR `json:"prs"`
}

// GraphQL query structure for searching pull requests
type searchQuery struct {
	Search struct {
		PageInfo struct {
			HasNextPage bool
			EndCursor   githubv4.String
		}
		Nodes []struct {
			PullRequest struct {
				Title     githubv4.String
				URL       githubv4.String
				State     githubv4.String
				CreatedAt githubv4.DateTime
				Author    struct {
					Login githubv4.String
				}
				Repository struct {
					NameWithOwner githubv4.String
				}
				Labels struct {
					Nodes []struct {
						Name githubv4.String
					}
				} `graphql:"labels(first: 10)"`
				BodyText githubv4.String
			} `graphql:"... on PullRequest"`
		}
	} `graphql:"search(query: $query, type: ISSUE, first: $first, after: $after)"`
}

func main() {
	// Parse command line flags
	outputDir := flag.String("output-dir", "data/junit5", "Directory to store output files")
	candidateFile := flag.String("candidate-file", "junit5_candidate_prs.txt", "File to store candidate PR URLs")
	existingFile := flag.String("existing-file", "junit5_pr_urls.txt", "File containing existing PR URLs")
	startDate := flag.String("start-date", "2024-07-01", "Start date for PR search (YYYY-MM-DD)")
	flag.Parse()

	// Get GitHub token from environment
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("GITHUB_TOKEN environment variable is required")
		os.Exit(1)
	}

	// Parse start date
	startDateTime, err := time.Parse("2006-01-02", *startDate)
	if err != nil {
		fmt.Printf("Error parsing start date: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Searching for PRs created on or after: %s\n", startDateTime.Format("2006-01-02"))

	// Create output directory if it doesn't exist
	os.MkdirAll(*outputDir, 0755)

	// Create GitHub client
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

	// Define search terms
	searchTerms := []string{
		"junit5",
		"junit 5",
		"migrate tests to junit",
		"junit jupiter",
		"openrewrite junit",
	}

	// Initialize result
	result := SearchResult{
		PRs: []JUnit5PR{},
	}

	// Search for each term
	for _, term := range searchTerms {
		fmt.Printf("Searching for: %s\n", term)

		// Search in title
		titleQuery := fmt.Sprintf("org:jenkinsci is:pr in:title %s", term)
		prs := searchPRs(client, titleQuery, startDateTime)
		result.PRs = append(result.PRs, prs...)

		// Search in body
		bodyQuery := fmt.Sprintf("org:jenkinsci is:pr in:body %s", term)
		prs = searchPRs(client, bodyQuery, startDateTime)
		result.PRs = append(result.PRs, prs...)
	}

	// Search for PRs by specific authors known for JUnit 5 migrations
	fmt.Println("Searching for PRs by known JUnit 5 migration authors...")
	authorQuery := "org:jenkinsci is:pr author:strangelookingnerd"
	prs := searchPRs(client, authorQuery, startDateTime)
	result.PRs = append(result.PRs, prs...)

	// Remove duplicates
	result.PRs = removeDuplicates(result.PRs)

	// Save results to JSON file
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	outputFile := fmt.Sprintf("%s/junit5_candidates.json", *outputDir)
	err = ioutil.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing JSON file: %v\n", err)
		os.Exit(1)
	}

	// Generate text file with URLs for easy addition to junit5_pr_urls.txt
	generateCandidateURLsFile(result.PRs, *candidateFile)

	// Compare with existing PRs in junit5_pr_urls.txt
	compareWithExistingPRs(result.PRs, *existingFile)

	fmt.Printf("Found %d potential JUnit 5 migration PR candidates\n", len(result.PRs))
	fmt.Printf("Results saved to %s and %s\n", outputFile, *candidateFile)
}

// searchPRs performs a GitHub search and returns PRs matching the query
func searchPRs(client *githubv4.Client, query string, startDate time.Time) []JUnit5PR {
	var q searchQuery
	variables := map[string]interface{}{
		"query": githubv4.String(query),
		"first": githubv4.Int(100),
		"after": (*githubv4.String)(nil),
	}

	var allPRs []JUnit5PR

	for {
		err := client.Query(context.Background(), &q, variables)
		if err != nil {
			fmt.Printf("Error querying GitHub: %v\n", err)
			return allPRs
		}

		for _, node := range q.Search.Nodes {
			pr := node.PullRequest

			// Extract labels
			var labels []string
			for _, label := range pr.Labels.Nodes {
				labels = append(labels, string(label.Name))
			}

			// Create PR object
			newPR := JUnit5PR{
				Title:      string(pr.Title),
				URL:        string(pr.URL),
				Repository: string(pr.Repository.NameWithOwner),
				State:      string(pr.State),
				Author:     string(pr.Author.Login),
				Labels:     labels,
				Body:       string(pr.BodyText),
				CreatedAt:  pr.CreatedAt.Format(time.RFC3339),
			}

			// Only include PRs created on or after the start date
			if pr.CreatedAt.Before(startDate) {
				continue
			}

			// Only include PRs that are likely related to JUnit 5 migration
			if isJUnit5MigrationPR(newPR) {
				allPRs = append(allPRs, newPR)
			}
		}

		if !q.Search.PageInfo.HasNextPage {
			break
		}
		variables["after"] = githubv4.NewString(q.Search.PageInfo.EndCursor)
	}

	return allPRs
}

// isJUnit5MigrationPR checks if a PR is likely related to JUnit 5 migration
func isJUnit5MigrationPR(pr JUnit5PR) bool {
	// Exclude dependency bumps
	if strings.HasPrefix(pr.Title, "Bump") || strings.HasPrefix(pr.Title, "bump") {
		return false
	}

	// Exclude PRs with specific JIRA ticket prefixes that are known not to be JUnit 5 related
	// These are examples of tickets that were incorrectly included
	excludePatterns := []string{
		`(?i)JENKINS-70560`,         // Improve test coverage
		`(?i)JENKINS-75447`,         // Fix Snippetizer rendering
		`(?i)JENKINS-\d+.*fix`,      // General fixes
		`(?i)fix`,                   // General fixes
		`(?i)improve test coverage`, // Test coverage improvements not related to JUnit 5
	}

	for _, pattern := range excludePatterns {
		matched, _ := regexp.MatchString(pattern, pr.Title)
		if matched {
			return false
		}
	}

	// Define more specific patterns for JUnit 5 migration PRs
	titlePatterns := []string{
		`(?i)migrate tests? to junit ?5`,
		`(?i)\bjunit ?5\b`, // Word boundary to ensure "junit5" is a standalone term
		`(?i)migrate to junit ?5`,
		`(?i)junit.*(4|four).*(5|five)`,
		`(?i)junit ?5.*(migration|upgrade)`,
		`(?i)openrewrite.*junit ?5`,
	}

	bodyPatterns := []string{
		`(?i)migrate (all )?tests? to junit ?5`,
		`(?i)\bjunit ?5\b`, // Word boundary to ensure "junit5" is a standalone term
		`(?i)migrate to junit ?5`,
		`(?i)junit.*(4|four).*(5|five)`,
		`(?i)junit ?5.*(migration|upgrade)`,
		`(?i)openrewrite.*junit ?5`,
		`(?i)org\.junit\.jupiter`,
		`(?i)junit-jupiter`,
	}

	labelPatterns := []string{
		`(?i)\bjunit ?5\b`,
		`(?i)junit-5`,
		`(?i)junit-migration`,
	}

	authorPatterns := []string{
		`strangelookingnerd`,
	}

	// Check title with more specific matching
	for _, pattern := range titlePatterns {
		matched, _ := regexp.MatchString(pattern, pr.Title)
		if matched {
			return true
		}
	}

	// For body matches, require stronger evidence
	// Count how many patterns match in the body
	bodyMatchCount := 0
	for _, pattern := range bodyPatterns {
		matched, _ := regexp.MatchString(pattern, pr.Body)
		if matched {
			bodyMatchCount++
		}
	}

	// Require at least 2 body pattern matches for a positive identification
	// This helps avoid false positives from PRs that mention JUnit in passing
	if bodyMatchCount >= 2 {
		return true
	}

	// Check labels
	for _, label := range pr.Labels {
		for _, pattern := range labelPatterns {
			matched, _ := regexp.MatchString(pattern, label)
			if matched {
				return true
			}
		}
	}

	// Check author - but only if there's at least some mention of JUnit in the body
	// This avoids including all PRs from certain authors
	for _, pattern := range authorPatterns {
		matched, _ := regexp.MatchString(pattern, pr.Author)
		if matched {
			// Check if there's at least some mention of JUnit in the body
			junitmatch, _ := regexp.MatchString(`(?i)junit`, pr.Body)
			if junitmatch {
				return true
			}
		}
	}

	return false
}

// removeDuplicates removes duplicate PRs from the slice
func removeDuplicates(prs []JUnit5PR) []JUnit5PR {
	seen := make(map[string]bool)
	var result []JUnit5PR

	for _, pr := range prs {
		if !seen[pr.URL] {
			seen[pr.URL] = true
			result = append(result, pr)
		}
	}

	return result
}

// generateCandidateURLsFile creates a text file with PR URLs for easy addition to junit5_pr_urls.txt
func generateCandidateURLsFile(prs []JUnit5PR, outputFile string) {
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	file.WriteString(fmt.Sprintf("# JUnit 5 migration PR candidates found on %s\n", time.Now().Format("2006-01-02 15:04:05")))
	file.WriteString("# Add relevant URLs to junit5_pr_urls.txt after verification\n\n")

	for _, pr := range prs {
		file.WriteString(fmt.Sprintf("# %s - %s (%s)\n", pr.Repository, pr.Title, pr.State))
		file.WriteString(fmt.Sprintf("%s\n\n", pr.URL))
	}
}

// compareWithExistingPRs compares found PRs with those already in junit5_pr_urls.txt
func compareWithExistingPRs(prs []JUnit5PR, existingFile string) {
	// Read existing PR URLs
	existingURLs := make(map[string]bool)

	data, err := ioutil.ReadFile(existingFile)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "https://") {
				existingURLs[line] = true
			}
		}
	}

	// Find new PRs
	var newPRs []JUnit5PR
	for _, pr := range prs {
		if !existingURLs[pr.URL] {
			newPRs = append(newPRs, pr)
		}
	}

	fmt.Printf("Found %d new PR candidates not already in %s\n", len(newPRs), existingFile)

	if len(newPRs) > 0 {
		fmt.Println("New PR candidates:")
		for _, pr := range newPRs {
			fmt.Printf("# %s - %s (%s)\n", pr.Repository, pr.Title, pr.State)
			fmt.Printf("%s\n\n", pr.URL)
		}
	}
}
