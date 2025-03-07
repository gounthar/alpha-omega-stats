package main

import (
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

	"github.com/google/go-github/v47/github"
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

	endDate, err := time.Parse("2006-01-02", *endDateFlag)
	if err != nil {
		log.Fatalf("Invalid end date format. Expected YYYY-MM-DD: %v", err)
	}

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

	// Initialize GitHub client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.GithubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Create a rate limiter
	limiter := rate.NewLimiter(config.RateLimit, 1)

	// Fetch Jenkins plugin repositories from update center
	log.Println("Fetching Jenkins plugin information from update center...")
	pluginRepos, err := fetchJenkinsPluginInfo(config.UpdateCenterURL)
	if err != nil {
		log.Fatalf("Failed to fetch plugin information: %v", err)
	}
	log.Printf("Found %d plugins in the update center", len(pluginRepos))

	// Fetch PRs for each plugin repository
	log.Println("Fetching pull requests...")
	pullRequests, err := fetchPullRequests(ctx, client, limiter, config, pluginRepos)
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

// fetchPullRequests fetches pull requests for each repository within the date range
func fetchPullRequests(ctx context.Context, client *github.Client, limiter *rate.Limiter, config Config, pluginRepos map[string]PluginInfo) ([]PullRequestData, error) {
	var allPRs []PullRequestData
	var mutex sync.Mutex
	var wg sync.WaitGroup

	// Create a semaphore to limit the number of concurrent goroutines
	semaphore := make(chan struct{}, 5) // Max 5 concurrent API calls

	org := "jenkinsci"

	// Create a search query for all PRs in the date range for the organization
	wg.Add(1)
	semaphore <- struct{}{} // Acquire semaphore

	go func() {
		defer wg.Done()
		defer func() { <-semaphore }() // Release semaphore

		// GitHub search query format
		query := fmt.Sprintf("org:%s type:pr created:%s..%s",
			org,
			config.StartDate.Format("2006-01-02"),
			config.EndDate.Format("2006-01-02"))

		opts := &github.SearchOptions{
			ListOptions: github.ListOptions{PerPage: 100},
			Sort:        "created",
			Order:       "asc",
		}

		for {
			// Respect rate limit
			if err := limiter.Wait(ctx); err != nil {
				log.Printf("Rate limiter error: %v", err)
				return
			}

			result, resp, err := client.Search.Issues(ctx, query, opts)
			if err != nil {
				log.Printf("Error searching PRs: %v", err)
				return nil, err // Return the error to the caller
			}

			for _, issue := range result.Issues {
				// Make sure it's a PR and not an issue
				if issue.PullRequestLinks == nil {
					continue
				}

				// Extract repository name from PR URL
				// URL format: https://github.com/jenkinsci/repo-name/pull/123
				urlParts := strings.Split(*issue.HTMLURL, "/")
				if len(urlParts) < 5 {
					continue
				}
				repoName := urlParts[4]

				// Check if this is a plugin repository from our list
				pluginInfo, isPlugin := pluginRepos[repoName]

				// Only process plugin repositories
				if !isPlugin {
					continue
				}

				// Collect labels
				var labels []string
				for _, label := range issue.Labels {
					labels = append(labels, *label.Name)
				}

				// Filter PRs where "pluginmodernizer" can be found in the PR body
				if issue.Body != nil && strings.Contains(*issue.Body, "pluginmodernizer") {
					pr := PullRequestData{
						Number:      *issue.Number,
						Title:       *issue.Title,
						State:       *issue.State,
						CreatedAt:   *issue.CreatedAt,
						UpdatedAt:   *issue.UpdatedAt,
						User:        *issue.User.Login,
						Repository:  fmt.Sprintf("%s/%s", org, repoName),
						PluginName:  pluginInfo.Name,
						Labels:      labels,
						URL:         *issue.HTMLURL,
						Description: *issue.Body,
					}

					mutex.Lock()
					allPRs = append(allPRs, pr)
					mutex.Unlock()
				}
			}

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
	}()

	wg.Wait()

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
