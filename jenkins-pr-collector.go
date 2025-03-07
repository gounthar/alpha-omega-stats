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

var allFoundPRs []PullRequestData

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

	// Debug response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d, Body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var graphqlResp GraphQLResponse
	err = json.Unmarshal(body, &graphqlResp)
	if err != nil {
		return fmt.Errorf("failed to parse GraphQL response: %v, Body: %s", err, string(body))
	}

	// Check for GraphQL errors
	if len(graphqlResp.Errors) > 0 {
		errorMsg := fmt.Sprintf("GraphQL error: %s", graphqlResp.Errors[0].Message)
		return fmt.Errorf(errorMsg)
	}

	// Parse data
	if graphqlResp.Data == nil {
		return fmt.Errorf("no data in GraphQL response")
	}

	err = json.Unmarshal(graphqlResp.Data, result)
	if err != nil {
		return fmt.Errorf("failed to parse GraphQL data: %v, Data: %s", err, string(graphqlResp.Data))
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

	// Debug: Print the search parameters
	log.Printf("Searching for PRs in org %s from %s to %s",
		org,
		config.StartDate.Format("2006-01-02"),
		config.EndDate.Format("2006-01-02"))

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
		searchQuery := fmt.Sprintf("org:%s is:pr created:%s..%s",
			org,
			startDate.Format("2006-01-02"),
			currentEndDate.Format("2006-01-02"))

		// Variables for the GraphQL query
		variables := map[string]interface{}{
			"query":  searchQuery,
			"cursor": nil,
		}

		hasNextPage := true
		totalFound := 0

		for hasNextPage {
			// Respect rate limit
			if err := limiter.Wait(ctx); err != nil {
				return nil, fmt.Errorf("rate limiter error: %v", err)
			}

			// Debug: Print current cursor position
			log.Printf("Fetching page with cursor: %v", variables["cursor"])

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

				log.Printf("Error executing GraphQL query (attempt %d/5): %v", attempt+1, err)
				if attempt == 4 {
					return nil, err
				}

				// Wait before retry
				waitTime := time.Duration(attempt+1) * time.Second * 2
				time.Sleep(waitTime)
			}

			if err != nil {
				return nil, fmt.Errorf("error executing GraphQL query after retries: %v", err)
			}

			// Debug: Print number of results in this page
			log.Printf("Received %d results in this page", len(response.Search.Nodes))
			totalFound += len(response.Search.Nodes)

			// Process search results
			for _, pr := range response.Search.Nodes {
				repoName := pr.Repository.Name

				// Log details of each pull request
				log.Printf("PR #%d: %s in repository %s/%s by %s",
					pr.Number, pr.Title, pr.Repository.Owner.Login, pr.Repository.Name, pr.Author.Login)

				// Check if this is a plugin repository from our list
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
					// Replace line 566 with:
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
				dependabotUser := "dependabot"
				renovateUser := "renovate"
				if pr.Author.Login == dependabotUser || pr.Author.Login == renovateUser {
					continue
				}

				// Check if "odernizer" can be found in the PR body
				if strings.Contains(pr.BodyText, "odernizer") || strings.Contains(pr.BodyText, "recipe") {
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
						CheckStatus: getCommitStatus(pr.Commits),
					}

					mutex.Lock()
					allPRs = append(allPRs, prData)
					mutex.Unlock()

					// Debug: log match found
					log.Printf("Matched PR #%d in repository %s with 'plugin-modernizer' traces", pr.Number, repoName)
				}
			}

			// Check if there are more pages
			hasNextPage = response.Search.PageInfo.HasNextPage
			if hasNextPage {
				variables["cursor"] = response.Search.PageInfo.EndCursor
				log.Printf("Moving to next page with cursor: %s", response.Search.PageInfo.EndCursor)
			}
		}

		log.Printf("Total PRs found before filtering: %d", totalFound)

		// Move to the next month
		startDate = currentEndDate.AddDate(0, 0, 1)
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
