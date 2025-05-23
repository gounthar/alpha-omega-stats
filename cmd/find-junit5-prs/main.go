package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
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
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "cannot create output dir %s: %v\n", *outputDir, err)
		os.Exit(1)
	}

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
	err = os.WriteFile(outputFile, jsonData, 0o644)
	if err != nil {
		fmt.Printf("Error writing JSON file: %v\n", err)
		os.Exit(1)
	}

	// Generate text file with URLs for easy addition to junit5_pr_urls.txt
	candidatePath := filepath.Join(*outputDir, *candidateFile)
	generateCandidateURLsFile(result.PRs, candidatePath)

	// Compare with existing PRs in junit5_pr_urls.txt
	existingPath := filepath.Join(*outputDir, *existingFile)
	compareWithExistingPRs(result.PRs, existingPath)

	fmt.Printf("Found %d potential JUnit 5 migration PR candidates\n", len(result.PRs))
	fmt.Printf("Results saved to %s and %s\n", outputFile, candidatePath)
}

// searchPRs performs a GitHub search and returns PRs matching the query
func searchPRs(client *githubv4.Client, query string, startDate time.Time) []JUnit5PR {
	var q searchQuery
	variables := map[string]interface{}{
		"query": githubv4.String(query),
		"first": githubv4.Int(10), // Reduced from 25 to 10 to make queries even less complex
		"after": (*githubv4.String)(nil),
	}

	var allPRs []JUnit5PR
	maxRetries := 10 // Increased from 5 to 10
	initialBackoff := 2 * time.Second
	maxBackoff := 60 * time.Second

	// Log the query being executed
	fmt.Printf("Executing GitHub GraphQL query: %s\n", query)
	fmt.Printf("Start date: %s\n", startDate.Format(time.RFC3339))

	pageCount := 0
	totalAttempts := 0
	maxPages := 100 // Set a maximum page limit to prevent infinite loops

	for pageCount < maxPages {
		// Add a small delay between requests to respect rate limits
		time.Sleep(1 * time.Second)

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		var err error
		success := false

		// Implement retry with exponential backoff
		for attempt := 0; attempt < maxRetries && !success; attempt++ {
			totalAttempts++

			// Get current rate limit information before making the request
			var rateLimit struct {
				RateLimit struct {
					Limit     githubv4.Int
					Remaining githubv4.Int
					ResetAt   githubv4.DateTime
				} `graphql:"rateLimit"`
			}

			rateLimitErr := client.Query(context.Background(), &rateLimit, nil)
			if rateLimitErr == nil {
				fmt.Printf("GitHub API rate limit: %d/%d remaining, resets at %s\n",
					rateLimit.RateLimit.Remaining,
					rateLimit.RateLimit.Limit,
					rateLimit.RateLimit.ResetAt.Format(time.RFC3339))

				// Check if we're close to hitting the rate limit
				if rateLimit.RateLimit.Remaining < 100 {
					resetTime := time.Until(rateLimit.RateLimit.ResetAt.Time)
					fmt.Printf("WARNING: Rate limit is low (%d remaining). Limit resets in %s\n",
						rateLimit.RateLimit.Remaining,
						resetTime.Round(time.Second))

					if rateLimit.RateLimit.Remaining < 10 {
						fmt.Println("Rate limit critically low, waiting until reset...")
						waitTime := time.Until(rateLimit.RateLimit.ResetAt.Time) + 10*time.Second
						time.Sleep(waitTime)
					}
				}
			} else {
				fmt.Printf("Could not fetch rate limit info: %v\n", rateLimitErr)
			}

			// Execute the actual query
			startTime := time.Now()
			err = client.Query(ctx, &q, variables)
			queryDuration := time.Since(startTime)

			if err == nil {
				success = true
				fmt.Printf("Query succeeded in %s\n", queryDuration.Round(time.Millisecond))

				// Log the number of results received
				resultCount := len(q.Search.Nodes)
				fmt.Printf("Received %d results for page %d\n", resultCount, pageCount+1)

				// Process the results
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

				// Log progress
				fmt.Printf("Processed page %d, found %d matching PRs so far\n", pageCount+1, len(allPRs))

				// Check if there are more pages
				if !q.Search.PageInfo.HasNextPage {
					fmt.Printf("No more pages available, completed after %d pages\n", pageCount+1)
					cancel()      // Cancel the context before returning
					return allPRs // Exit the function when no more pages
				}

				// Move to next page
				fmt.Printf("Moving to next page with cursor: %s\n", q.Search.PageInfo.EndCursor)
				variables["after"] = githubv4.NewString(q.Search.PageInfo.EndCursor)

				// Increment page count
				pageCount++

				// Add a small delay before the next page to avoid overwhelming GitHub
				time.Sleep(1 * time.Second)
				break
			}

			// Log detailed error information
			fmt.Printf("GitHub API error on attempt %d/%d (total %d): %v\n",
				attempt+1, maxRetries, totalAttempts, err)
			fmt.Printf("Query duration before error: %s\n", queryDuration.Round(time.Millisecond))

			// Try to parse and log more details from the error
			var errorDetails struct {
				Data   interface{}              `json:"data"`
				Errors []map[string]interface{} `json:"errors"`
			}

			if strings.Contains(err.Error(), "{") {
				errorJSON := err.Error()[strings.Index(err.Error(), "{"):]
				if jsonErr := json.Unmarshal([]byte(errorJSON), &errorDetails); jsonErr == nil {
					if errorDetails.Errors != nil && len(errorDetails.Errors) > 0 {
						fmt.Println("Detailed error information:")
						for i, errDetail := range errorDetails.Errors {
							fmt.Printf("  Error %d:\n", i+1)
							for k, v := range errDetail {
								fmt.Printf("    %s: %v\n", k, v)
							}
						}
					}
				}
			}

			// Check if this is a retryable error (like 502)
			if strings.Contains(err.Error(), "502") ||
				strings.Contains(err.Error(), "timeout") ||
				strings.Contains(err.Error(), "Something went wrong") {

				// Calculate backoff duration with exponential increase and jitter
				backoffTime := float64(initialBackoff) * math.Pow(2, float64(attempt))
				if backoffTime > float64(maxBackoff) {
					backoffTime = float64(maxBackoff)
				}

				// Add jitter (±20%)
				jitter := (rand.Float64() * 0.4) - 0.2 // -20% to +20%
				backoffTime = backoffTime * (1 + jitter)

				sleepDuration := time.Duration(backoffTime)
				fmt.Printf("Retrying in %v...\n", sleepDuration)
				time.Sleep(sleepDuration)
			} else {
				// Non-retryable error, break out
				fmt.Println("Non-retryable error encountered, stopping retry attempts")
				break
			}
		}

		if !success {
			fmt.Printf("Failed after %d attempts: %v\n", maxRetries, err)
			fmt.Println("Continuing with results collected so far...")
			cancel() // Cancel the context before returning
			return allPRs
		}

		// Cancel the context after we're done with it
		cancel()
	}

	// If we've reached the maximum number of pages, log a message
	fmt.Printf("Reached maximum page limit (%d). Stopping to prevent excessive API usage.\n", maxPages)
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

	// Pre-compile exclude patterns
	excludeRegexps := make([]*regexp.Regexp, len(excludePatterns))
	for i, pattern := range excludePatterns {
		excludeRegexps[i] = regexp.MustCompile(pattern)
	}

	for _, re := range excludeRegexps {
		if re.MatchString(pr.Title) {
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

	// Pre-compile title patterns
	titleRegexps := make([]*regexp.Regexp, len(titlePatterns))
	for i, pattern := range titlePatterns {
		titleRegexps[i] = regexp.MustCompile(pattern)
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

	// Pre-compile body patterns
	bodyRegexps := make([]*regexp.Regexp, len(bodyPatterns))
	for i, pattern := range bodyPatterns {
		bodyRegexps[i] = regexp.MustCompile(pattern)
	}

	labelPatterns := []string{
		`(?i)\bjunit ?5\b`,
		`(?i)junit-5`,
		`(?i)junit-migration`,
	}

	// Pre-compile label patterns
	labelRegexps := make([]*regexp.Regexp, len(labelPatterns))
	for i, pattern := range labelPatterns {
		labelRegexps[i] = regexp.MustCompile(pattern)
	}

	authorPatterns := []string{
		`strangelookingnerd`,
	}

	// Pre-compile author patterns
	authorRegexps := make([]*regexp.Regexp, len(authorPatterns))
	for i, pattern := range authorPatterns {
		authorRegexps[i] = regexp.MustCompile(pattern)
	}

	// Pre-compile the JUnit pattern used in author check
	junitRegexp := regexp.MustCompile(`(?i)junit`)

	// Check title with more specific matching
	for _, re := range titleRegexps {
		if re.MatchString(pr.Title) {
			return true
		}
	}

	// For body matches, require stronger evidence
	// Count how many patterns match in the body
	bodyMatchCount := 0
	for _, re := range bodyRegexps {
		if re.MatchString(pr.Body) {
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
		for _, re := range labelRegexps {
			if re.MatchString(label) {
				return true
			}
		}
	}

	// Check author - but only if there's at least some mention of JUnit in the body
	// This avoids including all PRs from certain authors
	for _, re := range authorRegexps {
		if re.MatchString(pr.Author) {
			// Check if there's at least some mention of JUnit in the body
			if junitRegexp.MatchString(pr.Body) {
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
	defer func() {
		if cerr := file.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "close error: %v\n", cerr)
		}
	}()

	if _, err := file.WriteString(fmt.Sprintf(
		"# JUnit 5 migration PR candidates found on %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
	)); err != nil {
		fmt.Fprintf(os.Stderr, "write error: %v\n", err)
	}
	if _, err := file.WriteString("# Add relevant URLs to junit5_pr_urls.txt after verification\n\n"); err != nil {
		fmt.Fprintf(os.Stderr, "write error: %v\n", err)
	}

	for _, pr := range prs {
		if _, err := file.WriteString(fmt.Sprintf("# %s - %s (%s)\n", pr.Repository, pr.Title, pr.State)); err != nil {
			fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		}
		if _, err := file.WriteString(fmt.Sprintf("%s\n\n", pr.URL)); err != nil {
			fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		}
	}
}

// compareWithExistingPRs compares found PRs with those already in junit5_pr_urls.txt
func compareWithExistingPRs(prs []JUnit5PR, existingFile string) {
	// Read existing PR URLs
	existingURLs := make(map[string]bool)

	data, err := os.ReadFile(existingFile)
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
