package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

// PullRequest represents a GitHub pull request
type PullRequest struct {
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	State      string    `json:"state"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	URL        string    `json:"url"`
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	BodyText string `json:"bodyText"`
	Labels   struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	Commits struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup struct {
					State string `json:"state"`
				} `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
}

// GraphQLSearchResponse represents the response structure for the search query
// Update GraphQLSearchResponse struct
type GraphQLSearchResponse struct {
	Search struct {
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage"`
			EndCursor   string `json:"endCursor"`
		} `json:"pageInfo"`
		Nodes []struct {
			// Remove the nested PullRequest struct and flatten the fields
			Number     int       `json:"number"`
			Title      string    `json:"title"`
			State      string    `json:"state"`
			CreatedAt  time.Time `json:"createdAt"`
			UpdatedAt  time.Time `json:"updatedAt"`
			URL        string    `json:"url"`
			Repository struct {
				Name  string `json:"name"`
				Owner struct {
					Login string `json:"login"`
				} `json:"owner"`
			} `json:"repository"`
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			BodyText string `json:"bodyText"`
			Labels   struct {
				Nodes []struct {
					Name string `json:"name"`
				} `json:"nodes"`
			} `json:"labels"`
			Commits struct {
				Nodes []struct {
					Commit struct {
						StatusCheckRollup struct {
							State string `json:"state"`
						} `json:"statusCheckRollup"`
					} `json:"commit"`
				} `json:"nodes"`
			} `json:"commits"`
		} `json:"nodes"`
	} `json:"search"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

// PullRequestData represents the data we want to collect about PRs
type PullRequestData struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	State       string    `json:"state"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	User        string    `json:"user"`
	Repository  string    `json:"repository"`
	PluginName  string    `json:"pluginName"`
	Labels      []string  `json:"labels"`
	URL         string    `json:"url"`
	Description string    `json:"description,omitempty"`
	CheckStatus string    `json:"checkStatus,omitempty"`
}

// PluginInfo represents the information we need from the plugins.json file
type PluginInfo struct {
	Name string `json:"name"`
	SCM  string `json:"scm"`
}

// UpdateCenter represents the structure of the update-center.json file
type UpdateCenter struct {
	Plugins map[string]struct {
		Name string `json:"name"`
		SCM  string `json:"scm"`
	} `json:"plugins"`
}

// Config holds the application configuration
type Config struct {
	GithubToken           string
	StartDate             time.Time
	EndDate               time.Time
	OutputFile            string
	FoundPullRequestsFile string
	UpdateCenterURL       string
	RateLimit             rate.Limit
}

// GraphQLClient represents a simple GitHub GraphQL API client
type GraphQLClient struct {
	httpClient *http.Client
	endpoint   string
}

// GraphQLRequest represents a GitHub GraphQL API request
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GitHub GraphQL API response
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

// GraphQLError represents a GitHub GraphQL API error
type GraphQLError struct {
	Message string   `json:"message"`
	Type    string   `json:"type"`
	Path    []string `json:"path,omitempty"`
}

// Add these new types for partial data saving
type PartialData struct {
	LastCursor string        `json:"last_cursor"`
	PRs        []PullRequest `json:"prs"`
	Timestamp  time.Time     `json:"timestamp"`
}

var allFoundPRs []PullRequestData

// Add these new types and constants
const (
	maxRetries = 5
	baseDelay  = 2 * time.Second
	maxDelay   = 5 * time.Minute
)

type RetryableError struct {
	Err       error
	ShouldLog bool
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func calculateBackoffDuration(attempt int) time.Duration {
	// Calculate exponential backoff with jitter
	delay := baseDelay * time.Duration(1<<uint(attempt))
	if delay > maxDelay {
		delay = maxDelay
	}
	// Add jitter (±10%)
	jitter := time.Duration(rand.Float64()*0.2-0.1) * delay
	return delay + jitter
}

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "rate_limit") ||
		strings.Contains(errStr, "secondary rate limit")
}

func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "temporary") ||
		strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") ||
		strings.Contains(errStr, "connection") ||
		isRateLimitError(err)
}

func main() {
	// Parse command line arguments
	githubToken := flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub API token")
	startDateFlag := flag.String("start", "", "Start date in YYYY-MM-DD format")
	endDateFlag := flag.String("end", "", "End date in YYYY-MM-DD format")
	outputFileFlag := flag.String("output", "jenkins_prs.json", "Output file name")
	foundPRsFileFlag := flag.String("found-prs", "found_prs.json", "File to write found PRs")
	updateCenterURLFlag := flag.String("update-center", "https://updates.jenkins.io/current/update-center.actual.json", "Jenkins update center URL")
	flag.Parse()

	// Validate required parameters
	if *githubToken == "" {
		log.Fatal("GitHub token is required. Set GITHUB_TOKEN environment variable or use -token flag.")
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", *startDateFlag)
	if err != nil {
		log.Fatalf("Invalid start date format. Expected YYYY-MM-DD: %v", err)
	}
	log.Printf("Parsed start date: %s", startDate.Format("2006-01-02"))

	endDate, err := time.Parse("2006-01-02", *endDateFlag)
	if err != nil {
		log.Fatalf("Invalid end date format. Expected YYYY-MM-DD: %v", err)
	}
	log.Printf("Parsed end date: %s", endDate.Format("2006-01-02"))

	// Make sure endDate is inclusive by setting it to the end of the day
	endDate = endDate.Add(24*time.Hour - 1*time.Second)

	// Create configuration
	config := Config{
		GithubToken:           *githubToken,
		StartDate:             startDate,
		EndDate:               endDate,
		OutputFile:            *outputFileFlag,
		FoundPullRequestsFile: *foundPRsFileFlag,
		UpdateCenterURL:       *updateCenterURLFlag,
		RateLimit:             rate.Limit(1), // 1 request per second is conservative
	}

	// Initialize GraphQL client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.GithubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	graphqlClient := &GraphQLClient{
		httpClient: tc,
		endpoint:   "https://api.github.com/graphql",
	}

	// Create a rate limiter
	limiter := rate.NewLimiter(config.RateLimit, 1)

	// Fetch Jenkins plugin repositories from update center
	log.Println("Fetching Jenkins plugin information from update center...")
	pluginRepos, err := fetchJenkinsPluginInfo(config.UpdateCenterURL)
	if err != nil {
		log.Fatalf("Failed to fetch plugin information: %v", err)
	}
	log.Printf("Found %d plugins in the update center", len(pluginRepos))

	// Fetch PRs using GraphQL
	log.Println("Fetching pull requests using GraphQL...")
	pullRequests, err := fetchPullRequestsGraphQL(ctx, graphqlClient, limiter, config, pluginRepos)
	if err != nil {
		log.Fatalf("Failed to fetch pull requests: %v", err)
	}
	log.Printf("Found %d pull requests", len(pullRequests))

	// Write results to file
	log.Printf("Writing results to %s...", config.OutputFile)
	err = writeJSONFile(config.OutputFile, pullRequests)
	if err != nil {
		log.Fatalf("Failed to write output file: %v", err)
	}

	// Write found PRs to another file if any PRs were found
	if len(allFoundPRs) > 0 {
		log.Printf("Writing all found PRs to %s...", config.FoundPullRequestsFile)
		err = writeJSONFile(config.FoundPullRequestsFile, allFoundPRs)
		if err != nil {
			log.Fatalf("Failed to write found PRs file: %v", err)
		}
	} else {
		log.Printf("No pull requests found, not writing to %s", config.FoundPullRequestsFile)
	}

	log.Println("Done!")
}

// fetchJenkinsPluginInfo fetches plugin information from the Jenkins update center
func fetchJenkinsPluginInfo(updateCenterURL string) (map[string]PluginInfo, error) {
	// Create an HTTP client that follows redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}

	var resp *http.Response
	var err error

	// Implement retry logic with exponential backoff
	for attempt := 0; attempt < 5; attempt++ {
		// Make HTTP request to update center
		resp, err = client.Get(updateCenterURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if err != nil {
			log.Printf("Failed to fetch update center data (attempt %d/5): %v", attempt+1, err)
		} else {
			log.Printf("HTTP error (attempt %d/5): %d", attempt+1, resp.StatusCode)
		}

		// Wait before retry
		waitTime := time.Duration(attempt+1) * time.Second * 2
		time.Sleep(waitTime)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch update center data after retries: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch update center data: HTTP %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read update center data: %v", err)
	}

	// The update-center.json starts with submitUpdateCenter(...) and ends with );
	// We need to extract the JSON part
	jsonStr := string(body)
	if strings.HasPrefix(jsonStr, "updateCenter.post(") {
		// Find the first {
		startIdx := strings.Index(jsonStr, "{")
		// Find the last }
		endIdx := strings.LastIndex(jsonStr, "}")
		if startIdx >= 0 && endIdx >= 0 && endIdx > startIdx {
			jsonStr = jsonStr[startIdx : endIdx+1]
		} else {
			return nil, fmt.Errorf("invalid update center JSON format")
		}
	}

	// Parse JSON
	var updateCenter UpdateCenter
	err = json.Unmarshal([]byte(jsonStr), &updateCenter)
	if err != nil {
		return nil, fmt.Errorf("failed to parse update center data: %v", err)
	}

	// Extract plugin information
	pluginRepos := make(map[string]PluginInfo)
	for name, plugin := range updateCenter.Plugins {
		if plugin.SCM != "" {
			// Extract repository name from SCM URL
			// SCM URL format: https://github.com/jenkinsci/repo-name
			repoURL := plugin.SCM

			// Make sure it's a GitHub repository in jenkinsci organization
			if strings.Contains(repoURL, "github.com/jenkinsci/") {
				parts := strings.Split(repoURL, "github.com/jenkinsci/")
				if len(parts) > 1 {
					repoName := strings.TrimSuffix(parts[1], ".git")
					repoName = strings.TrimSuffix(repoName, "/")

					pluginRepos[repoName] = PluginInfo{
						Name: name,
						SCM:  repoURL,
					}
				}
			}
		}
	}

	return pluginRepos, nil
}

// Add exponential backoff function
func getBackoffDuration(attempt int) time.Duration {
	// Start with 5 seconds, double each time, max 5 minutes
	duration := time.Duration(5*(1<<attempt)) * time.Second
	maxDuration := 5 * time.Minute
	if duration > maxDuration {
		return maxDuration
	}
	return duration
}

// Modify ExecuteGraphQL to handle retries internally
func (c *GraphQLClient) ExecuteGraphQL(ctx context.Context, req *GraphQLRequest, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			waitTime := calculateBackoffDuration(attempt - 1)
			log.Printf("Retrying request (attempt %d/%d) after %v...", attempt+1, maxRetries, waitTime)
			time.Sleep(waitTime)
		}

		err := c.executeGraphQLRequest(ctx, req, result)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if it's a retryable error
		if !isTransientError(err) {
			log.Printf("Non-retryable error encountered: %v", err)
			return err
		}

		// Handle rate limit specifically
		if isRateLimitError(err) {
			log.Printf("Rate limit exceeded on attempt %d/%d", attempt+1, maxRetries)
			// Use a longer backoff for rate limits
			waitTime := calculateBackoffDuration(attempt) * 2
			log.Printf("Rate limit exceeded, waiting %v before retry...", waitTime)
			time.Sleep(waitTime)
			continue
		}

		log.Printf("Retryable error encountered on attempt %d/%d: %v", attempt+1, maxRetries, err)
	}

	return fmt.Errorf("failed after %d retries, last error: %v", maxRetries, lastErr)
}

func (c *GraphQLClient) executeGraphQLRequest(ctx context.Context, req *GraphQLRequest, result interface{}) error {
	// Convert request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return &RetryableError{Err: fmt.Errorf("failed to execute request: %v", err), ShouldLog: true}
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))

		// Check for specific status codes
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return fmt.Errorf("authentication error: %v", err)
		case http.StatusNotFound:
			return fmt.Errorf("resource not found: %v", err)
		case http.StatusTooManyRequests:
			return &RetryableError{Err: fmt.Errorf("rate limit exceeded: %v", err), ShouldLog: true}
		case http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return &RetryableError{Err: fmt.Errorf("server error: %v", err), ShouldLog: true}
		default:
			return fmt.Errorf("request failed: %v", err)
		}
	}

	// Parse response
	var graphqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphqlResp); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	// Check for GraphQL errors
	if len(graphqlResp.Errors) > 0 {
		// Check if any of the errors are rate limit related
		for _, gqlErr := range graphqlResp.Errors {
			if isRateLimitError(fmt.Errorf(gqlErr.Message)) {
				return &RetryableError{
					Err:       fmt.Errorf("graphql rate limit error: %q", gqlErr.Message),
					ShouldLog: true,
				}
			}
		}

		// If we get here, these are other GraphQL errors
		var errMsgs []string
		for _, err := range graphqlResp.Errors {
			errMsgs = append(errMsgs, err.Message)
		}
		return fmt.Errorf("graphql errors: %s", strings.Join(errMsgs, "; "))
	}

	// Decode the actual result
	if err := json.Unmarshal(graphqlResp.Data, result); err != nil {
		return fmt.Errorf("failed to unmarshal data: %v", err)
	}

	return nil
}

func getCommitStatus(commits struct {
	Nodes []struct {
		Commit struct {
			StatusCheckRollup struct {
				State string `json:"state"`
			} `json:"statusCheckRollup"`
		} `json:"commit"`
	} `json:"nodes"`
}) string {
	if len(commits.Nodes) == 0 {
		return "UNKNOWN"
	}

	if commits.Nodes[0].Commit.StatusCheckRollup.State == "" {
		return "UNKNOWN"
	}

	return commits.Nodes[0].Commit.StatusCheckRollup.State
}

// Add function to save partial data
func savePartialData(prs []PullRequest, cursor string, outputFile string) error {
	partial := PartialData{
		LastCursor: cursor,
		PRs:        prs,
		Timestamp:  time.Now(),
	}

	// Create temp file
	tmpFile := outputFile + ".tmp"
	data, err := json.MarshalIndent(partial, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling partial data: %v", err)
	}

	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("error writing partial data: %v", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, outputFile+".partial"); err != nil {
		return fmt.Errorf("error saving partial data: %v", err)
	}

	return nil
}

// Add function to load partial data
func loadPartialData(outputFile string) (*PartialData, error) {
	partialFile := outputFile + ".partial"
	data, err := os.ReadFile(partialFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error reading partial data: %v", err)
	}

	var partial PartialData
	if err := json.Unmarshal(data, &partial); err != nil {
		return nil, fmt.Errorf("error parsing partial data: %v", err)
	}

	// Check if partial data is too old (> 24 hours)
	if time.Since(partial.Timestamp) > 24*time.Hour {
		os.Remove(partialFile)
		return nil, nil
	}

	return &partial, nil
}

// Modify executeGraphQLQuery to use exponential backoff
func executeGraphQLQuery(client *http.Client, query string, variables map[string]interface{}) (*GraphQLSearchResponse, error) {
	var lastErr error
	maxAttempts := 10 // Increase max attempts

	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err := sendGraphQLRequest(client, query, variables)
		if err == nil {
			// Check for specific error types in the response
			if resp.Search.Nodes == nil && len(resp.Errors) > 0 {
				// Handle specific GitHub API errors
				if strings.Contains(resp.Errors[0].Message, "rate limit") {
					waitTime := getBackoffDuration(attempt)
					log.Printf("Rate limit hit. Waiting %v before retry...", waitTime)
					time.Sleep(waitTime)
					continue
				}
				if strings.Contains(resp.Errors[0].Message, "Something went wrong") {
					waitTime := getBackoffDuration(attempt)
					log.Printf("GitHub API error. Waiting %v before retry...", waitTime)
					time.Sleep(waitTime)
					continue
				}
			}
			return resp, nil
		}

		lastErr = err
		waitTime := getBackoffDuration(attempt)
		log.Printf("Error executing query (attempt %d/%d): %v. Waiting %v...",
			attempt+1, maxAttempts, err, waitTime)
		time.Sleep(waitTime)
	}

	return nil, fmt.Errorf("failed after %d attempts: %v", maxAttempts, lastErr)
}

// Modify fetchPullRequests to use partial data
func fetchPullRequests(client *http.Client, startDate, endDate string, outputFile string) error {
	var allPRs []PullRequest
	var cursor string

	// Try to load partial data
	partial, err := loadPartialData(outputFile)
	if err != nil {
		log.Printf("Warning: Could not load partial data: %v", err)
	} else if partial != nil {
		log.Printf("Resuming from cursor: %s with %d PRs", partial.LastCursor, len(partial.PRs))
		allPRs = partial.PRs
		cursor = partial.LastCursor
	}

	for {
		variables := map[string]interface{}{
			"queryString": buildSearchQuery(startDate, endDate),
			"cursor":      cursor,
		}

		resp, err := executeGraphQLQuery(client, searchQuery, variables)
		if err != nil {
			return fmt.Errorf("error executing query: %v", err)
		}

		// Process the page of results
		for _, node := range resp.Search.Nodes {
			pr := convertNodeToPR(node)
			allPRs = append(allPRs, pr)
		}

		// Save partial data after each successful page
		if err := savePartialData(allPRs, cursor, outputFile); err != nil {
			log.Printf("Warning: Could not save partial data: %v", err)
		}

		if !resp.Search.PageInfo.HasNextPage {
			break
		}
		cursor = resp.Search.PageInfo.EndCursor
		log.Printf("Fetched %d PRs so far...", len(allPRs))
	}

	// Save final data
	finalData, err := json.MarshalIndent(allPRs, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling final data: %v", err)
	}

	if err := os.WriteFile(outputFile, finalData, 0644); err != nil {
		return fmt.Errorf("error writing final data: %v", err)
	}

	// Clean up partial file
	os.Remove(outputFile + ".partial")
	return nil
}

// writeJSONFile writes data to a JSON file
func writeJSONFile(filename string, data interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// Add these helper functions
func sendGraphQLRequest(client *http.Client, query string, variables map[string]interface{}) (*GraphQLSearchResponse, error) {
	// Prepare the request body
	requestBody := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Add GitHub token header
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable not set")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error: %d, Body: %s", resp.StatusCode, string(body))
	}

	var response GraphQLSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &response, nil
}

func buildSearchQuery(startDate, endDate string) string {
	return fmt.Sprintf("org:jenkinsci is:pr created:%s..%s", startDate, endDate)
}

// GraphQL query for searching PRs
const searchQuery = `
query SearchPullRequests($queryString: String!, $cursor: String) {
	search(query: $queryString, type: ISSUE, first: 100, after: $cursor) {
		pageInfo {
			hasNextPage
			endCursor
		}
		nodes {
			... on PullRequest {
				number
				title
				state
				createdAt
				updatedAt
				url
				repository {
					name
					owner {
						login
					}
				}
				author {
					login
				}
				bodyText
				labels(first: 100) {
					nodes {
						name
					}
				}
				commits(last: 1) {
					nodes {
						commit {
							statusCheckRollup {
								state
							}
						}
					}
				}
			}
		}
	}
}`

func convertNodeToPR(node struct {
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	State      string    `json:"state"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	URL        string    `json:"url"`
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	BodyText string `json:"bodyText"`
	Labels   struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	Commits struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup struct {
					State string `json:"state"`
				} `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
}) PullRequest {
	pr := PullRequest{
		Number:     node.Number,
		Title:      node.Title,
		State:      node.State,
		CreatedAt:  node.CreatedAt,
		UpdatedAt:  node.UpdatedAt,
		URL:        node.URL,
		Repository: node.Repository,
		Author:     node.Author,
		BodyText:   node.BodyText,
		Labels:     node.Labels,
		Commits:    node.Commits,
	}
	return pr
}

// Add fetchPullRequestsGraphQL function
func fetchPullRequestsGraphQL(ctx context.Context, client *GraphQLClient, limiter *rate.Limiter, config Config, pluginRepos map[string]PluginInfo) ([]PullRequestData, error) {
	var allPRs []PullRequestData
	// mutex protects concurrent access to shared data structures:
	// - allPRs: The slice containing all collected PRs that match our criteria
	// - allFoundPRs: The global slice containing all PRs found during collection
	// This mutex is necessary because multiple goroutines may be processing PRs concurrently
	// when handling paginated GraphQL responses, and we need to ensure thread-safe
	// appending to these slices to prevent race conditions and data corruption.
	var mutex sync.Mutex
	var lastError error

	// Split the date range into monthly chunks
	startDate := config.StartDate
	endDate := config.EndDate

	for startDate.Before(endDate) {
		// Calculate the end of the current month
		currentEndDate := startDate.AddDate(0, 1, -startDate.Day())
		if currentEndDate.After(endDate) {
			currentEndDate = endDate
		}

		// GitHub search query format for PRs
		queryString := fmt.Sprintf("org:jenkinsci is:pr created:%s..%s",
			startDate.Format("2006-01-02"),
			currentEndDate.Format("2006-01-02"))

		// Variables for the GraphQL query
		variables := map[string]interface{}{
			"queryString": queryString,
			"cursor":      nil,
		}

		hasNextPage := true
		for hasNextPage {
			// Respect rate limit
			if err := limiter.Wait(ctx); err != nil {
				log.Printf("Warning: Rate limit error: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			var response GraphQLSearchResponse
			err := client.ExecuteGraphQL(ctx, &GraphQLRequest{
				Query:     searchQuery,
				Variables: variables,
			}, &response)
			if err != nil {
				log.Printf("Warning: GraphQL query error: %v", err)
				lastError = err
				// Save what we have so far before continuing
				if len(allPRs) > 0 {
					if err := writeJSONFile(config.OutputFile+".partial", allPRs); err != nil {
						log.Printf("Warning: Failed to save partial results: %v", err)
					}
				}
				time.Sleep(5 * time.Second)
				continue
			}

			// Process search results
			for _, pr := range response.Search.Nodes {
				repoName := pr.Repository.Name
				pluginInfo, isPlugin := pluginRepos[repoName]

				// Add all found PRs to the global array
				prData := PullRequestData{
					Number:      pr.Number,
					Title:       pr.Title,
					State:       pr.State,
					CreatedAt:   pr.CreatedAt,
					UpdatedAt:   pr.UpdatedAt,
					User:        pr.Author.Login,
					Repository:  fmt.Sprintf("%s/%s", pr.Repository.Owner.Login, pr.Repository.Name),
					PluginName:  pluginInfo.Name,
					Labels:      []string{},
					URL:         pr.URL,
					Description: pr.BodyText,
					CheckStatus: getCommitStatus(pr.Commits),
				}

				mutex.Lock()
				allFoundPRs = append(allFoundPRs, prData)
				mutex.Unlock()

				// Only process plugin repositories
				if !isPlugin {
					continue
				}

				// Filter out PRs created by Dependabot and Renovate
				if pr.Author.Login == "dependabot" || pr.Author.Login == "renovate" {
					continue
				}

				// Check if "odernizer" can be found in the PR body
				if strings.Contains(pr.BodyText, "odernizer") || strings.Contains(pr.BodyText, "recipe") {
					// Collect labels
					var labels []string
					for _, label := range pr.Labels.Nodes {
						labels = append(labels, label.Name)
					}
					prData.Labels = labels
					mutex.Lock()
					allPRs = append(allPRs, prData)
					mutex.Unlock()
				}
			}

			// Check if there are more pages
			hasNextPage = response.Search.PageInfo.HasNextPage
			if hasNextPage {
				variables["cursor"] = response.Search.PageInfo.EndCursor
			}
		}

		// Move to the next month
		startDate = currentEndDate.AddDate(0, 0, 1)
	}

	// If we have any results but also had errors, return what we have
	if len(allPRs) > 0 && lastError != nil {
		log.Printf("Warning: Completed with partial results due to errors: %v", lastError)
		return allPRs, nil
	}

	if lastError != nil {
		return nil, lastError
	}

	return allPRs, nil
}
