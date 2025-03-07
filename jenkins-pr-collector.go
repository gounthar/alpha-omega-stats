package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

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
	GithubToken     string
	StartDate       time.Time
	EndDate         time.Time
	OutputFile      string
	UpdateCenterURL string
	RateLimit       rate.Limit
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
	Message string `json:"message"`
}

// GraphQLSearchResponse represents the response structure for the search query
type GraphQLSearchResponse struct {
	Search struct {
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage"`
			EndCursor   string `json:"endCursor"`
		} `json:"pageInfo"`
		Nodes []struct {
			PullRequest struct {
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
			} `json:"... on PullRequest"`
		} `json:"nodes"`
	} `json:"search"`
}

func main() {
	// Parse command line arguments
	githubToken := flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub API token")
	startDateFlag := flag.String("start", "", "Start date in YYYY-MM-DD format")
	endDateFlag := flag.String("end", "", "End date in YYYY-MM-DD format")
	outputFileFlag := flag.String("output", "jenkins_prs.json", "Output file name")
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
		GithubToken:     *githubToken,
		StartDate:       startDate,
		EndDate:         endDate,
		OutputFile:      *outputFileFlag,
		UpdateCenterURL: *updateCenterURLFlag,
		RateLimit:       rate.Limit(1), // 1 request per second is conservative
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
	log.Println("Done!")
}

// fetchJenkinsPluginInfo fetches plugin information from the Jenkins update center
func fetchJenkinsPluginInfo(updateCenterURL string) (map[string]PluginInfo, error) {
	// Make HTTP request to update center
	resp, err := http.Get(updateCenterURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch update center data: %v", err)
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

// ExecuteGraphQL executes a GraphQL query against the GitHub API
func (c *GraphQLClient) ExecuteGraphQL(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	// Create request
	reqBody, err := json.Marshal(GraphQLRequest{
		Query:     query,
		Variables: variables,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal GraphQL request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse response
	var graphqlResp GraphQLResponse
	err = json.Unmarshal(body, &graphqlResp)
	if err != nil {
		return fmt.Errorf("failed to parse GraphQL response: %v", err)
	}

	// Check for GraphQL errors
	if len(graphqlResp.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", graphqlResp.Errors[0].Message)
	}

	// Parse data
	err = json.Unmarshal(graphqlResp.Data, result)
	if err != nil {
		return fmt.Errorf("failed to parse GraphQL data: %v", err)
	}

	return nil
}

// fetchPullRequestsGraphQL fetches pull requests using the GitHub GraphQL API
func fetchPullRequestsGraphQL(ctx context.Context, client *GraphQLClient, limiter *rate.Limiter, config Config, pluginRepos map[string]PluginInfo) ([]PullRequestData, error) {
	var allPRs []PullRequestData
	var mutex sync.Mutex
	org := "jenkinsci"

	// Define the GraphQL query for searching PRs
	query := `
	query SearchPullRequests($query: String!, $cursor: String) {
		search(query: $query, type: ISSUE, first: 100, after: $cursor) {
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
				}
			}
		}
	}`

	// GitHub search query format for PRs
	searchQuery := fmt.Sprintf("org:%s type:pr created:%s..%s",
		org,
		config.StartDate.Format("2006-01-02"),
		config.EndDate.Format("2006-01-02"))

	// Variables for the GraphQL query
	variables := map[string]interface{}{
		"query":  searchQuery,
		"cursor": nil,
	}

	hasNextPage := true

	for hasNextPage {
		// Respect rate limit
		if err := limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter error: %v", err)
		}

		// Implement retry logic with exponential backoff
		var response GraphQLSearchResponse
		var err error
		for attempt := 0; attempt < 5; attempt++ {
			err = client.ExecuteGraphQL(ctx, query, variables, &response)
			if err == nil {
				break
			}

			// Check if it's a rate limit error
			if strings.Contains(err.Error(), "rate limit") {
				waitTime := time.Duration(attempt+1) * time.Second * 5
				log.Printf("Rate limit exceeded, retrying in %v...", waitTime)
				time.Sleep(waitTime)
				continue
			}

			log.Printf("Error executing GraphQL query: %v", err)
			return nil, err
		}

		if err != nil {
			return nil, fmt.Errorf("error executing GraphQL query after retries: %v", err)
		}

		// Process search results
		for _, node := range response.Search.Nodes {
			pr := node.PullRequest
			repoName := pr.Repository.Name

			// Check if this is a plugin repository from our list
			pluginInfo, isPlugin := pluginRepos[repoName]

			// Only process plugin repositories
			if !isPlugin {
				continue
			}

			// Check if "odernizer" can be found in the PR body
			if strings.Contains(pr.BodyText, "odernizer") {
				// Collect labels
				var labels []string
				for _, label := range pr.Labels.Nodes {
					labels = append(labels, label.Name)
				}

				prData := PullRequestData{
					Number:      pr.Number,
					Title:       pr.Title,
					State:       pr.State,
					CreatedAt:   pr.CreatedAt,
					UpdatedAt:   pr.UpdatedAt,
					User:        pr.Author.Login,
					Repository:  fmt.Sprintf("%s/%s", pr.Repository.Owner.Login, pr.Repository.Name),
					PluginName:  pluginInfo.Name,
					Labels:      labels,
					URL:         pr.URL,
					Description: pr.BodyText,
				}

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

	return allPRs, nil
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
