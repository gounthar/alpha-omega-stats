package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	dockerHubAPIBaseURL = "https://hub.docker.com/v2"
	maxRetries         = 3
	requestTimeout     = 30 * time.Second
)

// Client represents a Docker Hub API client
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Docker Hub API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		baseURL: dockerHubAPIBaseURL,
	}
}

// SearchUserRepositories searches for repositories by a specific user using v2 API
func (c *Client) SearchUserRepositories(ctx context.Context, username string) ([]DockerSearchResult, error) {
	log.Printf("Searching Docker Hub repositories for user: %s", username)

	// Use v2 API to list user repositories with pagination
	var allRepos []DockerSearchResult
	page := 1
	pageSize := 100

	for {
		url := fmt.Sprintf("%s/repositories/%s/?page=%d&page_size=%d", c.baseURL, username, page, pageSize)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "GitHub-Profile-Analyzer/1.0")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			// User has no repositories or doesn't exist
			break
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}

		var repoResponse DockerHubRepositoriesResponse
		if err := json.NewDecoder(resp.Body).Decode(&repoResponse); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		// Convert v2 API response to our search result format
		for _, repo := range repoResponse.Results {
			searchResult := DockerSearchResult{
				RepoName:         repo.Name,
				RepoOwner:        repo.Namespace,
				ShortDescription: repo.Description,
				IsOfficial:       false,
				IsAutomated:      false,
				StarCount:        repo.StarCount,
				PullCount:        repo.PullCount,
			}
			allRepos = append(allRepos, searchResult)
		}

		// Check if there are more pages
		if repoResponse.Next == "" {
			break
		}
		page++
	}

	log.Printf("Found %d repositories for user %s", len(allRepos), username)
	return allRepos, nil
}

// GetRepositoryDetails fetches detailed information about a specific repository
func (c *Client) GetRepositoryDetails(ctx context.Context, namespace, repository string) (*DockerHubRepositoryResponse, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s", c.baseURL, namespace, repository)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "GitHub-Profile-Analyzer/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("repository not found: %s/%s", namespace, repository)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var repoDetails DockerHubRepositoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&repoDetails); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &repoDetails, nil
}

// GetRepositoryTags fetches tags for a repository
func (c *Client) GetRepositoryTags(ctx context.Context, namespace, repository string) ([]DockerTag, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/tags?page_size=100", c.baseURL, namespace, repository)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "GitHub-Profile-Analyzer/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tagsResponse struct {
		Count    int         `json:"count"`
		Next     string      `json:"next"`
		Previous string      `json:"previous"`
		Results  []DockerTag `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tagsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode tags response: %w", err)
	}

	return tagsResponse.Results, nil
}


// AnalyzeDockerProfile creates a comprehensive Docker Hub profile analysis
func (c *Client) AnalyzeDockerProfile(ctx context.Context, username string) (*DockerHubProfile, error) {
	log.Printf("Analyzing Docker Hub profile for user: %s", username)

	profile := &DockerHubProfile{
		Username:   username,
		ProfileURL: fmt.Sprintf("https://hub.docker.com/u/%s", username),
	}

	// Search for user's repositories
	searchResults, err := c.SearchUserRepositories(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to search user repositories: %w", err)
	}

	var repositories []DockerRepository
	var totalDownloads int64

	// Get detailed info for each repository
	for _, result := range searchResults {
		// Parse namespace and repo name
		parts := strings.Split(result.RepoName, "/")
		var namespace, repoName string
		if len(parts) == 2 {
			namespace, repoName = parts[0], parts[1]
		} else {
			namespace, repoName = username, result.RepoName
		}

		// Get detailed repository information
		repoDetails, err := c.GetRepositoryDetails(ctx, namespace, repoName)
		if err != nil {
			log.Printf("Warning: failed to get details for %s/%s: %v", namespace, repoName, err)
			continue
		}

		// Convert to our repository structure
		repo := DockerRepository{
			Name:             repoDetails.Name,
			Namespace:        repoDetails.Namespace,
			FullName:         fmt.Sprintf("%s/%s", repoDetails.Namespace, repoDetails.Name),
			Description:      repoDetails.Description,
			ShortDescription: result.ShortDescription,
			IsOfficial:       result.IsOfficial,
			IsAutomated:      repoDetails.IsAutomated,
			StarCount:        repoDetails.StarCount,
			PullCount:        repoDetails.PullCount,
			LastUpdated:      repoDetails.LastUpdated,
			DateRegistered:   repoDetails.DateRegistered,
		}

		// Get tags information
		tags, err := c.GetRepositoryTags(ctx, namespace, repoName)
		if err != nil {
			log.Printf("Warning: failed to get tags for %s/%s: %v", namespace, repoName, err)
		} else {
			repo.Tags = tags
		}

		// Calculate impact metrics
		repo.PopularityScore = c.calculatePopularityScore(repo)
		repo.MaintenanceScore = c.calculateMaintenanceScore(repo)
		repo.ImpactLevel = c.determineImpactLevel(repo)

		repositories = append(repositories, repo)
		totalDownloads += repo.PullCount

		// Add small delay to be respectful to the API
		time.Sleep(500 * time.Millisecond)
	}

	profile.Repositories = repositories
	profile.TotalDownloads = totalDownloads
	profile.TotalImages = len(repositories)
	profile.PublicRepos = len(repositories)

	// Calculate overall impact metrics
	profile.ImpactMetrics = c.calculateImpactMetrics(profile)
	profile.ContainerExpertise = c.inferContainerExpertise(profile)

	log.Printf("Docker Hub analysis complete for %s: %d repos, %d total downloads",
		username, len(repositories), totalDownloads)

	return profile, nil
}

// calculatePopularityScore calculates a popularity score for a repository
func (c *Client) calculatePopularityScore(repo DockerRepository) float64 {
	// Score based on downloads, stars, and recency
	downloadScore := float64(repo.PullCount) / 1000000 // Normalize by millions
	starScore := float64(repo.StarCount) / 100         // Normalize by hundreds

	// Recency bonus (repositories updated in last year get bonus)
	recencyBonus := 0.0
	if time.Since(repo.LastUpdated).Hours() < 24*365 {
		recencyBonus = 1.0
	}

	score := (downloadScore*0.7 + starScore*0.2 + recencyBonus*0.1)
	if score > 10 {
		score = 10 // Cap at 10
	}

	return score
}

// calculateMaintenanceScore calculates how well-maintained a repository appears
func (c *Client) calculateMaintenanceScore(repo DockerRepository) float64 {
	daysSinceUpdate := time.Since(repo.LastUpdated).Hours() / 24

	// Recent updates get higher scores
	if daysSinceUpdate < 30 {
		return 10.0
	} else if daysSinceUpdate < 90 {
		return 8.0
	} else if daysSinceUpdate < 365 {
		return 6.0
	} else if daysSinceUpdate < 730 {
		return 4.0
	} else {
		return 2.0
	}
}

// determineImpactLevel determines the impact level based on download numbers
func (c *Client) determineImpactLevel(repo DockerRepository) string {
	if repo.PullCount > 10000000 { // 10M+
		return "massive"
	} else if repo.PullCount > 1000000 { // 1M+
		return "high"
	} else if repo.PullCount > 100000 { // 100K+
		return "medium"
	} else {
		return "low"
	}
}

// calculateImpactMetrics calculates overall impact metrics for the profile
func (c *Client) calculateImpactMetrics(profile *DockerHubProfile) DockerImpactMetrics {
	metrics := DockerImpactMetrics{
		TotalDownloads: profile.TotalDownloads,
	}

	// Find most popular repository
	var maxDownloads int64
	for _, repo := range profile.Repositories {
		if repo.PullCount > maxDownloads {
			maxDownloads = repo.PullCount
			metrics.MostDownloadedImage = repo.FullName
		}
		metrics.TopRepositories = append(metrics.TopRepositories, repo.FullName)
	}

	// Calculate community impact (0-10 scale)
	if profile.TotalDownloads > 50000000 { // 50M+
		metrics.CommunityImpact = 10.0
		metrics.InfrastructureInfluence = 9.0
		metrics.EnterpriseAdoption = true
	} else if profile.TotalDownloads > 10000000 { // 10M+
		metrics.CommunityImpact = 8.0
		metrics.InfrastructureInfluence = 7.0
		metrics.EnterpriseAdoption = true
	} else if profile.TotalDownloads > 1000000 { // 1M+
		metrics.CommunityImpact = 6.0
		metrics.InfrastructureInfluence = 5.0
	} else if profile.TotalDownloads > 100000 { // 100K+
		metrics.CommunityImpact = 4.0
		metrics.InfrastructureInfluence = 3.0
	} else {
		metrics.CommunityImpact = 2.0
		metrics.InfrastructureInfluence = 1.0
	}

	// Count image variants
	for _, repo := range profile.Repositories {
		metrics.TotalImageVariants += len(repo.Tags)
	}

	return metrics
}

// inferContainerExpertise infers container expertise from Docker Hub activity
func (c *Client) inferContainerExpertise(profile *DockerHubProfile) ContainerExpertise {
	expertise := ContainerExpertise{
		Technologies: []ContainerTechnology{},
	}

	// Estimate experience years from oldest repository
	var oldestDate time.Time = time.Now()
	for _, repo := range profile.Repositories {
		if repo.DateRegistered.Before(oldestDate) {
			oldestDate = repo.DateRegistered
		}
	}
	expertise.ExperienceYears = time.Since(oldestDate).Hours() / (24 * 365.25)

	// Determine proficiency level
	if expertise.ExperienceYears > 5 && profile.TotalDownloads > 10000000 {
		expertise.ProficiencyLevel = "expert"
		expertise.DevOpsIntegration = true
		expertise.ProductionOptimization = true
		expertise.MultiStageBuilds = true
	} else if expertise.ExperienceYears > 3 && profile.TotalDownloads > 1000000 {
		expertise.ProficiencyLevel = "advanced"
		expertise.DevOpsIntegration = true
		expertise.ProductionOptimization = true
	} else if expertise.ExperienceYears > 1 && profile.TotalDownloads > 100000 {
		expertise.ProficiencyLevel = "intermediate"
		expertise.DevOpsIntegration = true
	} else {
		expertise.ProficiencyLevel = "beginner"
	}

	// Infer specializations based on repository patterns
	for _, repo := range profile.Repositories {
		repoNameLower := strings.ToLower(repo.FullName)
		descLower := strings.ToLower(repo.Description)

		if strings.Contains(repoNameLower, "jenkins") || strings.Contains(descLower, "jenkins") {
			expertise.Technologies = append(expertise.Technologies, ContainerTechnology{
				Name:             "Jenkins CI/CD",
				Category:         "continuous-integration",
				ProficiencyScore: 0.9,
				Evidence:         []string{repo.FullName},
				ProjectCount:     1,
			})
		}

		if strings.Contains(repoNameLower, "kubernetes") || strings.Contains(descLower, "kubernetes") {
			expertise.Orchestration = append(expertise.Orchestration, "Kubernetes")
		}

		if strings.Contains(repoNameLower, "alpine") || strings.Contains(descLower, "alpine") {
			expertise.BaseImages = append(expertise.BaseImages, "Alpine Linux")
		}

		// Check for microservices patterns
		if strings.Contains(descLower, "microservice") || strings.Contains(descLower, "api") {
			expertise.Microservices = true
		}
	}

	return expertise
}