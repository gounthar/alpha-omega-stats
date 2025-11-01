package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jenkins/github-profile-tools/internal/github"
	"github.com/jenkins/github-profile-tools/internal/docker"
	"github.com/jenkins/github-profile-tools/internal/discourse"
)

// Analyzer handles the analysis of GitHub user profiles
type Analyzer struct {
	client          *github.Client
	dockerClient    *docker.Client
	discourseClient *discourse.Client
	saveProgressDir string
	cacheDir        string
}

// NewAnalyzer creates a new profile analyzer
func NewAnalyzer(githubToken string) *Analyzer {
	return &Analyzer{
		client:          github.NewClient(githubToken),
		dockerClient:    docker.NewClient(),
		discourseClient: discourse.NewClient(),
		saveProgressDir: "./data/progress",
		cacheDir:        "./data/cache",
	}
}

// AnalyzeUser performs comprehensive analysis of a GitHub user
func (a *Analyzer) AnalyzeUser(ctx context.Context, username string) (*UserProfile, error) {
	return a.AnalyzeUserWithDockerUsername(ctx, username, username)
}

// AnalyzeUserWithDockerUsername performs comprehensive analysis with separate Docker username
func (a *Analyzer) AnalyzeUserWithDockerUsername(ctx context.Context, username, dockerUsername string) (*UserProfile, error) {
	return a.AnalyzeUserWithCustomUsernames(ctx, username, dockerUsername, "")
}

func (a *Analyzer) AnalyzeUserWithCustomUsernames(ctx context.Context, username, dockerUsername, discourseUsername string) (*UserProfile, error) {
	log.Printf("Starting analysis for user: %s", username)

	// First, try to load from cache (completed analysis)
	if cachedProfile := a.tryLoadFromCache(username); cachedProfile != nil {
		log.Printf("Using cached analysis for user: %s (analyzed at %s)", username, cachedProfile.LastAnalyzed.Format("2006-01-02 15:04:05"))
		return cachedProfile, nil
	}

	// If no cache, try to resume from saved progress
	profile, resumeStep := a.tryResumeProgress(username)
	if profile == nil {
		profile = &UserProfile{
			Username:     username,
			LastAnalyzed: time.Now(),
		}
		resumeStep = 1
	}

	// Step 1: Fetch basic user information
	if resumeStep <= 1 {
		if err := a.fetchUserBasicInfo(ctx, username, profile); err != nil {
			return nil, fmt.Errorf("failed to fetch user basic info: %w", err)
		}
		if err := a.saveProgress(username, profile, 1); err != nil {
			log.Printf("Warning: Failed to save progress after step 1: %v", err)
		}
	}

	// Step 2: Fetch user repositories (incremental, continues with partial data on error)
	if resumeStep <= 2 {
		if err := a.fetchUserRepositories(ctx, username, profile); err != nil {
			log.Printf("Warning: Repository fetching encountered issues: %v", err)
			log.Printf("Continuing with %d repositories already fetched", len(profile.Repositories))
			// Don't return error - continue with whatever repositories we have
		}
		if err := a.saveProgress(username, profile, 2); err != nil {
			log.Printf("Warning: Failed to save progress after step 2: %v", err)
		}
	}

	// Step 3: Fetch organizations (non-critical, continue on failure)
	if resumeStep <= 3 {
		if err := a.fetchUserOrganizations(ctx, username, profile); err != nil {
			log.Printf("Warning: Failed to fetch organizations (continuing): %v", err)
			// Continue without organizations data
		}
		if err := a.saveProgress(username, profile, 3); err != nil {
			log.Printf("Warning: Failed to save progress after step 3: %v", err)
		}
	}

	// Step 4: Fetch contribution data (non-critical, continue on failure)
	if resumeStep <= 4 {
		if err := a.fetchUserContributions(ctx, username, profile); err != nil {
			log.Printf("Warning: Failed to fetch contributions (continuing): %v", err)
			// Continue without detailed contribution data
		}
		if err := a.saveProgress(username, profile, 4); err != nil {
			log.Printf("Warning: Failed to save progress after step 4: %v", err)
		}
	}

	// Step 5: Analyze languages and technologies
	if resumeStep <= 5 {
		a.analyzeLanguages(profile)
		if err := a.saveProgress(username, profile, 5); err != nil {
			log.Printf("Warning: Failed to save progress after step 5: %v", err)
		}
	}

	// Step 6: Analyze Docker Hub profile (optional - may not exist for all users)
	if resumeStep <= 6 {
		if err := a.analyzeDockerHub(ctx, dockerUsername, profile); err != nil {
			log.Printf("Docker Hub analysis failed (this is optional): %v", err)
			// Continue without Docker Hub data - not all users have Docker Hub profiles
		}
		if err := a.saveProgress(username, profile, 6); err != nil {
			log.Printf("Warning: Failed to save progress after step 6: %v", err)
		}
	}

	// Step 7: Analyze Discourse community engagement (optional - for Jenkins community members)
	if resumeStep <= 7 {
		if err := a.analyzeDiscourseProfile(ctx, username, discourseUsername, profile); err != nil {
			log.Printf("Discourse analysis failed (this is optional): %v", err)
			// Continue without Discourse data - not all users are active in Jenkins community
		}
		if err := a.saveProgress(username, profile, 7); err != nil {
			log.Printf("Warning: Failed to save progress after step 7: %v", err)
		}
	}

	// Step 8: Generate insights
	if resumeStep <= 8 {
		a.generateInsights(profile)
	}

	// Provide analysis summary
	a.logAnalysisSummary(profile)

	// Save completed analysis to cache for future template generation
	if err := a.saveToCache(username, profile); err != nil {
		log.Printf("Warning: Failed to save analysis to cache: %v", err)
	}

	// Clean up progress file on successful completion
	a.cleanupProgress(username)

	log.Printf("Analysis completed for user: %s", username)
	return profile, nil
}

// fetchUserBasicInfo fetches basic user information
func (a *Analyzer) fetchUserBasicInfo(ctx context.Context, username string, profile *UserProfile) error {
	log.Printf("Fetching basic info for user: %s", username)

	req := &github.GraphQLRequest{
		Query: github.UserProfileQuery,
		Variables: map[string]interface{}{
			"username": username,
		},
	}

	var resp github.UserProfileResponse
	if err := a.client.ExecuteGraphQL(ctx, req, &resp); err != nil {
		return fmt.Errorf("GraphQL query failed: %w", err)
	}

	user := resp.User
	profile.Name = user.Name
	profile.Bio = user.Bio
	profile.Company = user.Company
	profile.Location = user.Location
	profile.Email = user.Email
	profile.BlogURL = user.WebsiteUrl
	profile.TwitterUsername = user.TwitterUsername
	profile.CreatedAt = user.CreatedAt
	profile.UpdatedAt = user.UpdatedAt
	profile.PublicRepos = user.Repositories.TotalCount
	profile.PublicGists = 0 // Gists not accessible with current token permissions
	profile.Followers = user.Followers.TotalCount
	profile.Following = user.Following.TotalCount

	// Initialize contributions summary
	profile.Contributions = ContributionSummary{
		TotalCommits:                user.ContributionsCollection.TotalCommitContributions,
		TotalIssues:                 user.ContributionsCollection.TotalIssueContributions,
		TotalPullRequests:           user.ContributionsCollection.TotalPullRequestContributions,
		TotalCodeReviews:            user.ContributionsCollection.TotalPullRequestReviewContributions,
		YearlyContributions:         make(map[string]int),
		MonthlyContributions:        make(map[string]int),
	}

	// Set contribution years
	if len(user.ContributionsCollection.ContributionYears) > 0 {
		profile.Contributions.ContributionYears = len(user.ContributionsCollection.ContributionYears)

		years := user.ContributionsCollection.ContributionYears
		sort.Ints(years)

		if len(years) > 0 {
			profile.Contributions.MostActiveYear = years[len(years)-1] // Most recent year
		}
	}

	return nil
}

// fetchUserRepositories fetches user's repositories with pagination
func (a *Analyzer) fetchUserRepositories(ctx context.Context, username string, profile *UserProfile) error {
	log.Printf("Starting incremental repository fetching for user: %s", username)

	// Initialize repositories if not already present
	if profile.Repositories == nil {
		profile.Repositories = []RepositoryProfile{}
	}

	var cursor string
	const pageSize = 50 // Smaller page size for better incremental processing
	pageNum := 1
	totalFetched := len(profile.Repositories)

	for {
		log.Printf("Fetching repository page %d for user: %s (cursor: %s)", pageNum, username, cursor)

		req := &github.GraphQLRequest{
			Query: github.UserRepositoriesQuery,
			Variables: map[string]interface{}{
				"username": username,
				"first":    pageSize,
				"after":    cursor,
			},
		}

		var resp github.UserRepositoriesResponse
		log.Printf("Executing GraphQL query for page %d...", pageNum)
		if err := a.client.ExecuteGraphQL(ctx, req, &resp); err != nil {
			// Save progress with whatever we have so far
			if len(profile.Repositories) > 0 {
				log.Printf("GraphQL error after fetching %d repositories, saving progress: %v", len(profile.Repositories), err)
				if saveErr := a.saveProgress(username, profile, 2); saveErr != nil {
					log.Printf("Warning: Failed to save progress during repository fetching: %v", saveErr)
				}
				// Continue with partial data rather than failing completely
				log.Printf("Continuing analysis with %d repositories fetched so far", len(profile.Repositories))
				return nil
			}
			return fmt.Errorf("GraphQL query failed on first page: %w", err)
		}
		log.Printf("GraphQL query completed for page %d, got %d repositories in response", pageNum, len(resp.User.Repositories.Nodes))

		// Process repositories from this page immediately
		log.Printf("Starting to process %d repositories from page %d", len(resp.User.Repositories.Nodes), pageNum)
		newReposThisPage := 0
		for i, repoNode := range resp.User.Repositories.Nodes {
			if i > 0 && i%10 == 0 {
				log.Printf("Processed %d/%d repositories on page %d", i, len(resp.User.Repositories.Nodes), pageNum)
			}
			repo := a.convertRepositoryNode(ctx, repoNode, username)
			profile.Repositories = append(profile.Repositories, repo)
			newReposThisPage++

			// Check if context was cancelled during processing
			if ctx.Err() != nil {
				log.Printf("Context cancelled during repository processing, saving progress with %d repositories", totalFetched+newReposThisPage)
				if err := a.saveProgress(username, profile, 2); err != nil {
					log.Printf("Warning: Failed to save progress: %v", err)
				}
				return ctx.Err()
			}
		}
		log.Printf("Finished processing all %d repositories from page %d", newReposThisPage, pageNum)

		totalFetched += newReposThisPage
		log.Printf("Processed page %d: %d repositories (%d total fetched)", pageNum, newReposThisPage, totalFetched)

		// Save progress after each page
		if err := a.saveProgress(username, profile, 2); err != nil {
			log.Printf("Warning: Failed to save progress after page %d: %v", pageNum, err)
		}

		// Update skills analysis incrementally every few pages for better progress tracking
		if pageNum%3 == 0 {
			log.Printf("Running incremental skills analysis after page %d", pageNum)
			a.analyzeSkills(profile)
		}

		if !resp.User.Repositories.PageInfo.HasNextPage {
			log.Printf("Completed repository fetching: %d total repositories", totalFetched)
			break
		}

		cursor = resp.User.Repositories.PageInfo.EndCursor
		pageNum++

		// Add small delay between pages to be nice to GitHub's servers
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			log.Printf("Context cancelled, saving progress with %d repositories", totalFetched)
			return ctx.Err()
		}
	}

	log.Printf("Successfully fetched %d repositories for user: %s", len(profile.Repositories), username)
	return nil
}

// fetchUserOrganizations fetches user's organizations
func (a *Analyzer) fetchUserOrganizations(ctx context.Context, username string, profile *UserProfile) error {
	log.Printf("Fetching organizations for user: %s", username)

	req := &github.GraphQLRequest{
		Query: github.UserOrganizationsQuery,
		Variables: map[string]interface{}{
			"username": username,
		},
	}

	var resp github.UserOrganizationsResponse
	if err := a.client.ExecuteGraphQL(ctx, req, &resp); err != nil {
		return fmt.Errorf("GraphQL query failed: %w", err)
	}

	var orgs []OrganizationProfile
	for _, orgNode := range resp.User.Organizations.Nodes {
		org := OrganizationProfile{
			Name:        orgNode.Name,
			Login:       orgNode.Login,
			Description: orgNode.Description,
			URL:         orgNode.Url,
			AvatarURL:   orgNode.AvatarUrl,
			Role:        "member", // Default role
			IsPublicMember: true, // Assume public if returned by API
		}

		// Add repositories if available
		if orgNode.Repositories != nil {
			for _, repoNode := range orgNode.Repositories.Nodes {
				org.Repositories = append(org.Repositories, repoNode.NameWithOwner)
				org.ContributionCount++
			}
		}

		orgs = append(orgs, org)
	}

	// Process contributed repositories to identify additional organizations
	for _, repoNode := range resp.User.RepositoriesContributedTo.Nodes {
		orgLogin := repoNode.Owner.Login
		if orgLogin != username { // Skip user's own repositories
			// Find existing org or create new entry
			orgExists := false
			for i, org := range orgs {
				if org.Login == orgLogin {
					orgs[i].Repositories = append(orgs[i].Repositories, repoNode.NameWithOwner)
					orgs[i].ContributionCount++
					orgs[i].Role = "contributor"
					orgExists = true
					break
				}
			}

			if !orgExists {
				// Create new organization entry for contributor
				newOrg := OrganizationProfile{
					Name:              repoNode.Owner.Name,
					Login:             orgLogin,
					Description:       repoNode.Owner.Description,
					Repositories:      []string{repoNode.NameWithOwner},
					ContributionCount: 1,
					Role:              "contributor",
					IsPublicMember:    false,
				}
				orgs = append(orgs, newOrg)
			}
		}
	}

	profile.Organizations = orgs
	log.Printf("Fetched %d organizations for user: %s", len(orgs), username)
	return nil
}

// fetchUserContributions fetches detailed contribution data
func (a *Analyzer) fetchUserContributions(ctx context.Context, username string, profile *UserProfile) error {
	log.Printf("Fetching contributions for user: %s", username)

	// Fetch contributions for the last year
	now := time.Now()
	oneYearAgo := now.AddDate(-1, 0, 0)

	req := &github.GraphQLRequest{
		Query: github.UserContributionsQuery,
		Variables: map[string]interface{}{
			"username": username,
			"from":     oneYearAgo.Format(time.RFC3339),
			"to":       now.Format(time.RFC3339),
		},
	}

	var resp github.UserContributionsResponse
	if err := a.client.ExecuteGraphQL(ctx, req, &resp); err != nil {
		return fmt.Errorf("GraphQL query failed: %w", err)
	}

	contrib := resp.User.ContributionsCollection

	// Update contribution summary
	profile.Contributions.TotalCommits = contrib.TotalCommitContributions
	profile.Contributions.TotalIssues = contrib.TotalIssueContributions
	profile.Contributions.TotalPullRequests = contrib.TotalPullRequestContributions
	profile.Contributions.TotalCodeReviews = contrib.TotalPullRequestReviewContributions

	// Process contribution calendar
	if contrib.ContributionCalendar.TotalContributions > 0 {
		weeklyPattern := make([]int, 7)
		monthlyContributions := make(map[string]int)

		for _, week := range contrib.ContributionCalendar.Weeks {
			for _, day := range week.ContributionDays {
				if day.ContributionCount > 0 {
					date, err := time.Parse("2006-01-02", day.Date)
					if err == nil {
						// Add to weekly pattern (0 = Sunday)
						weeklyPattern[int(date.Weekday())] += day.ContributionCount

						// Add to monthly contributions
						monthKey := date.Format("2006-01")
						monthlyContributions[monthKey] += day.ContributionCount
					}
				}
			}
		}

		profile.Contributions.WeeklyPattern = weeklyPattern
		profile.Contributions.MonthlyContributions = monthlyContributions

		// Calculate consistency score (how evenly distributed contributions are)
		profile.Contributions.ConsistencyScore = a.calculateConsistencyScore(weeklyPattern)
	}

	// Process repository contributions for repository-specific stats
	for _, repoContrib := range contrib.CommitContributionsByRepository {
		repoName := repoContrib.Repository.NameWithOwner

		// Find the repository in profile and update stats
		for i, repo := range profile.Repositories {
			if repo.FullName == repoName {
				for _, contrib := range repoContrib.Contributions.Nodes {
					if contrib.User.Login == username {
						profile.Repositories[i].ContributionStats.Commits += contrib.CommitCount
					}
				}
				break
			}
		}
	}

	return nil
}

// convertRepositoryNode converts a GitHub repository node to our RepositoryProfile
func (a *Analyzer) convertRepositoryNode(ctx context.Context, node github.RepositoryNode, username string) RepositoryProfile {
	repo := RepositoryProfile{
		Name:        node.Name,
		FullName:    node.NameWithOwner,
		Description: node.Description,
		URL:         node.Url,
		IsPrivate:   node.IsPrivate,
		IsFork:      node.IsFork,
		IsArchived:  node.IsArchived,
		Stars:       node.StargazerCount,
		Forks:       node.ForkCount,
		Size:        node.DiskUsage,
		CreatedAt:   node.CreatedAt,
		UpdatedAt:   node.UpdatedAt,
		PushedAt:    node.PushedAt,
		Languages:   make(map[string]int),
		IsOwner:     true, // Assume owner since it's in user's repositories
	}

	// Set primary language
	if node.PrimaryLanguage != nil {
		repo.Language = node.PrimaryLanguage.Name
	}

	// Set languages with sizes
	if node.Languages != nil {
		for i, lang := range node.Languages.Nodes {
			if i < len(node.Languages.Edges) {
				repo.Languages[lang.Name] = node.Languages.Edges[i].Size
			}
		}
	}

	// Set topics
	if node.RepositoryTopics != nil {
		for _, topicNode := range node.RepositoryTopics.Nodes {
			repo.Topics = append(repo.Topics, topicNode.Topic.Name)
		}
	}

	// Set license
	if node.LicenseInfo != nil {
		repo.License = node.LicenseInfo.Name
	}

	// Set optional counts
	if node.Watchers != nil {
		repo.Watchers = node.Watchers.TotalCount
	}

	if node.Issues != nil {
		repo.OpenIssues = node.Issues.TotalCount
	}

	// Analyze Docker configuration
	dockerConfig := a.analyzeDockerConfig(ctx, repo.FullName)
	if dockerConfig != nil {
		repo.DockerConfig = dockerConfig
	}
	// Collaborators data not accessible due to permission restrictions
	repo.CollaboratorCount = 0

	// Set organization if different from owner
	if node.Owner.Login != "" {
		repo.Organization = node.Owner.Login
		// Check if user is the owner by comparing login names
		repo.IsOwner = (node.Owner.Login == username)
	}

	return repo
}

// analyzeLanguages analyzes programming languages used by the user
func (a *Analyzer) analyzeLanguages(profile *UserProfile) {
	log.Printf("Analyzing languages for user: %s", profile.Username)

	languageStats := make(map[string]*LanguageStats)
	totalBytes := 0

	// Aggregate language data from all repositories
	for _, repo := range profile.Repositories {
		for language, bytes := range repo.Languages {
			if bytes == 0 {
				continue
			}

			if languageStats[language] == nil {
				languageStats[language] = &LanguageStats{
					Language:    language,
					FirstUsed:   repo.CreatedAt,
					LastUsed:    repo.UpdatedAt,
				}
			}

			stats := languageStats[language]
			stats.Bytes += bytes
			stats.RepositoryCount++
			stats.ProjectCount++
			totalBytes += bytes

			// Update first/last used dates
			if repo.CreatedAt.Before(stats.FirstUsed) {
				stats.FirstUsed = repo.CreatedAt
			}
			if repo.UpdatedAt.After(stats.LastUsed) {
				stats.LastUsed = repo.UpdatedAt
			}
		}
	}

	// Convert to slice and calculate percentages
	var languages []LanguageStats
	for _, stats := range languageStats {
		if totalBytes > 0 {
			stats.Percentage = float64(stats.Bytes) / float64(totalBytes) * 100
		}

		// Calculate proficiency score based on usage
		yearsUsed := time.Since(stats.FirstUsed).Hours() / (24 * 365.25)
		repoFactor := float64(stats.RepositoryCount) / float64(len(profile.Repositories))

		stats.ProficiencyScore = (stats.Percentage/100 + repoFactor + yearsUsed/10) / 3
		if stats.ProficiencyScore > 1 {
			stats.ProficiencyScore = 1
		}

		languages = append(languages, *stats)
	}

	// Sort by percentage
	sort.Slice(languages, func(i, j int) bool {
		return languages[i].Percentage > languages[j].Percentage
	})

	profile.Languages = languages

	// Analyze skills based on languages and repositories
	a.analyzeSkills(profile)
}

// analyzeSkills infers technical skills from repositories and languages
func (a *Analyzer) analyzeSkills(profile *UserProfile) {
	log.Printf("Analyzing skills for user: %s", profile.Username)

	skills := SkillProfile{
		Frameworks:     []TechnologySkill{},
		Databases:      []TechnologySkill{},
		Tools:          []TechnologySkill{},
		CloudPlatforms: []TechnologySkill{},
		DevOpsSkills:   []TechnologySkill{},
		TechnicalAreas: []TechnicalArea{},
	}

	// Set primary and secondary languages
	for i, lang := range profile.Languages {
		if i < 3 {
			skills.PrimaryLanguages = append(skills.PrimaryLanguages, lang.Language)
		} else if i < 8 {
			skills.SecondaryLanguages = append(skills.SecondaryLanguages, lang.Language)
		}
	}

	// Infer technologies from repository topics and names
	technologyMap := make(map[string]*TechnologySkill)

	for _, repo := range profile.Repositories {
		// Analyze topics
		for _, topic := range repo.Topics {
			a.categorizeTopicAsSkill(topic, repo, technologyMap, &skills)
		}

		// Analyze repository names and descriptions
		repoText := strings.ToLower(repo.Name + " " + repo.Description)
		a.inferTechnologiesFromText(repoText, repo, technologyMap, &skills)

		// Analyze Docker configuration if present
		if repo.DockerConfig != nil {
			a.categorizeDockerSkills(repo, technologyMap, &skills)
		}
	}

	profile.Skills = skills
}

// categorizeTopicAsSkill categorizes repository topics into technology skills
func (a *Analyzer) categorizeTopicAsSkill(topic string, repo RepositoryProfile, techMap map[string]*TechnologySkill, skills *SkillProfile) {
	topicLower := strings.ToLower(topic)

	// Framework patterns
	frameworks := []string{"react", "vue", "angular", "django", "flask", "spring", "express", "laravel", "rails", "nextjs", "nuxt"}
	if a.containsAny(topicLower, frameworks) {
		a.addTechnologySkill(topic, "framework", repo, techMap, &skills.Frameworks)
		return
	}

	// Database patterns
	databases := []string{"mysql", "postgresql", "mongodb", "redis", "sqlite", "cassandra", "elasticsearch"}
	if a.containsAny(topicLower, databases) {
		a.addTechnologySkill(topic, "database", repo, techMap, &skills.Databases)
		return
	}

	// Cloud patterns
	cloudPlatforms := []string{"aws", "gcp", "azure", "docker", "kubernetes", "terraform", "serverless"}
	if a.containsAny(topicLower, cloudPlatforms) {
		a.addTechnologySkill(topic, "cloud", repo, techMap, &skills.CloudPlatforms)
		return
	}

	// DevOps patterns
	devops := []string{"ci", "cd", "jenkins", "github-actions", "gitlab-ci", "monitoring", "logging"}
	if a.containsAny(topicLower, devops) {
		a.addTechnologySkill(topic, "devops", repo, techMap, &skills.DevOpsSkills)
		return
	}
}

// inferTechnologiesFromText infers technologies from repository text
func (a *Analyzer) inferTechnologiesFromText(text string, repo RepositoryProfile, techMap map[string]*TechnologySkill, skills *SkillProfile) {
	// This is a simplified version - in production you'd have more sophisticated NLP

	commonTechs := map[string]string{
		"microservice": "architecture",
		"api":         "backend",
		"restful":     "backend",
		"graphql":     "backend",
		"frontend":    "frontend",
		"backend":     "backend",
		"fullstack":   "fullstack",
		"mobile":      "mobile",
		"android":     "mobile",
		"ios":         "mobile",
		"machine-learning": "ml",
		"ai":          "ml",
		"data-science": "data",
		"analytics":   "data",
	}

	for tech, category := range commonTechs {
		if strings.Contains(text, tech) {
			a.addTechnicalArea(category, repo, skills)
		}
	}
}

// addTechnologySkill adds or updates a technology skill
func (a *Analyzer) addTechnologySkill(name, category string, repo RepositoryProfile, techMap map[string]*TechnologySkill, skillList *[]TechnologySkill) {
	if techMap[name] == nil {
		techMap[name] = &TechnologySkill{
			Name:         name,
			FirstUsed:    repo.CreatedAt,
			LastUsed:     repo.UpdatedAt,
			Evidence:     []string{},
			ProjectCount: 0,
		}
	}

	skill := techMap[name]
	skill.ProjectCount++
	skill.Evidence = append(skill.Evidence, repo.FullName)

	if repo.CreatedAt.Before(skill.FirstUsed) {
		skill.FirstUsed = repo.CreatedAt
	}
	if repo.UpdatedAt.After(skill.LastUsed) {
		skill.LastUsed = repo.UpdatedAt
	}

	// Calculate confidence based on project count and stars
	skill.Confidence = float64(skill.ProjectCount) * 0.2
	if repo.Stars > 0 {
		skill.Confidence += float64(repo.Stars) * 0.01
	}
	if skill.Confidence > 1 {
		skill.Confidence = 1
	}

	// Set proficiency level
	if skill.Confidence < 0.3 {
		skill.ProficiencyLevel = "beginner"
	} else if skill.Confidence < 0.6 {
		skill.ProficiencyLevel = "intermediate"
	} else if skill.Confidence < 0.8 {
		skill.ProficiencyLevel = "advanced"
	} else {
		skill.ProficiencyLevel = "expert"
	}

	// Add to skill list if not already present
	found := false
	for i, existingSkill := range *skillList {
		if existingSkill.Name == name {
			(*skillList)[i] = *skill
			found = true
			break
		}
	}
	if !found {
		*skillList = append(*skillList, *skill)
	}
}

// addTechnicalArea adds or updates a technical area
func (a *Analyzer) addTechnicalArea(area string, repo RepositoryProfile, skills *SkillProfile) {
	// Find existing area or create new one
	for i, existing := range skills.TechnicalAreas {
		if existing.Area == area {
			skills.TechnicalAreas[i].ProjectCount++
			skills.TechnicalAreas[i].Competency += 0.1
			if skills.TechnicalAreas[i].Competency > 1 {
				skills.TechnicalAreas[i].Competency = 1
			}
			return
		}
	}

	// Create new technical area
	newArea := TechnicalArea{
		Area:         area,
		ProjectCount: 1,
		Competency:   0.3,
		Technologies: []string{repo.Language},
		YearsActive:  time.Since(repo.CreatedAt).Hours() / (24 * 365.25),
	}

	skills.TechnicalAreas = append(skills.TechnicalAreas, newArea)
}

// containsAny checks if text contains any of the given patterns
func (a *Analyzer) containsAny(text string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

// calculateConsistencyScore calculates how consistent the user's contributions are
func (a *Analyzer) calculateConsistencyScore(weeklyPattern []int) float64 {
	if len(weeklyPattern) == 0 {
		return 0
	}

	total := 0
	for _, count := range weeklyPattern {
		total += count
	}

	if total == 0 {
		return 0
	}

	average := float64(total) / float64(len(weeklyPattern))
	variance := 0.0

	for _, count := range weeklyPattern {
		diff := float64(count) - average
		variance += diff * diff
	}

	variance /= float64(len(weeklyPattern))

	// Lower variance = higher consistency
	// Normalize to 0-1 scale
	consistency := 1.0 / (1.0 + variance/average)

	return consistency
}

// generateInsights generates AI-powered insights about the user
func (a *Analyzer) generateInsights(profile *UserProfile) {
	log.Printf("Generating insights for user: %s", profile.Username)

	insights := UserInsights{
		LeadershipIndicators: []LeadershipIndicator{},
		MentorshipSigns:     []string{},
		TechnicalFocus:      []string{},
		RecommendedRoles:    []string{},
		StrengthAreas:       []string{},
		GrowthAreas:         []string{},
	}

	// Determine career level
	insights.CareerLevel = a.determineCareerLevel(profile)

	// Identify technical focus areas
	for _, lang := range profile.Languages {
		if lang.Percentage > 10 { // Focus on languages with >10% usage
			insights.TechnicalFocus = append(insights.TechnicalFocus, lang.Language)
		}
	}

	// Analyze leadership indicators
	insights.LeadershipIndicators = a.analyzeLeadershipIndicators(profile)

	// Calculate overall impact score
	insights.OverallImpactScore = a.calculateImpactScore(profile)

	// Generate role recommendations
	insights.RecommendedRoles = a.recommendRoles(profile)

	// Identify strengths and growth areas
	insights.StrengthAreas, insights.GrowthAreas = a.identifyStrengthsAndGrowthAreas(profile)

	profile.Insights = insights
}

// determineCareerLevel determines the user's career level based on various factors
func (a *Analyzer) determineCareerLevel(profile *UserProfile) string {
	yearsActive := float64(profile.Contributions.ContributionYears)
	totalStars := 0
	ownedRepos := 0

	for _, repo := range profile.Repositories {
		totalStars += repo.Stars
		if repo.IsOwner {
			ownedRepos++
		}
	}

	orgCount := len(profile.Organizations)

	// Scoring system
	score := 0.0

	// Years of experience
	if yearsActive >= 8 {
		score += 4
	} else if yearsActive >= 5 {
		score += 3
	} else if yearsActive >= 2 {
		score += 2
	} else {
		score += 1
	}

	// Repository ownership and quality
	if ownedRepos >= 20 && totalStars >= 100 {
		score += 3
	} else if ownedRepos >= 10 && totalStars >= 20 {
		score += 2
	} else if ownedRepos >= 5 {
		score += 1
	}

	// Organization involvement
	if orgCount >= 5 {
		score += 2
	} else if orgCount >= 2 {
		score += 1
	}

	// Contribution consistency
	if profile.Contributions.ConsistencyScore >= 0.7 {
		score += 1
	}

	// Determine level
	if score >= 8 {
		return "principal"
	} else if score >= 6 {
		return "senior"
	} else if score >= 4 {
		return "mid"
	} else {
		return "junior"
	}
}

// analyzeLeadershipIndicators identifies signs of technical leadership
func (a *Analyzer) analyzeLeadershipIndicators(profile *UserProfile) []LeadershipIndicator {
	var indicators []LeadershipIndicator

	// Repository ownership
	ownedRepos := 0
	starsReceived := 0
	for _, repo := range profile.Repositories {
		if repo.IsOwner {
			ownedRepos++
			starsReceived += repo.Stars
		}
	}

	if ownedRepos >= 5 {
		indicators = append(indicators, LeadershipIndicator{
			Type:        "project_ownership",
			Evidence:    []string{fmt.Sprintf("%d owned repositories", ownedRepos)},
			Strength:    float64(ownedRepos) / 20, // Normalize to 0-1
			Description: "Demonstrates ability to create and maintain projects",
		})
	}

	// Organization involvement
	if len(profile.Organizations) >= 2 {
		indicators = append(indicators, LeadershipIndicator{
			Type:        "organizational_involvement",
			Evidence:    []string{fmt.Sprintf("Active in %d organizations", len(profile.Organizations))},
			Strength:    float64(len(profile.Organizations)) / 10,
			Description: "Shows collaborative leadership across multiple teams",
		})
	}

	return indicators
}

// calculateImpactScore calculates an overall impact score for the user
func (a *Analyzer) calculateImpactScore(profile *UserProfile) float64 {
	score := 0.0

	// Contribution volume
	totalContributions := profile.Contributions.TotalCommits + profile.Contributions.TotalPullRequests + profile.Contributions.TotalIssues
	score += float64(totalContributions) / 1000 // Normalize

	// Repository impact (stars received)
	totalStars := 0
	for _, repo := range profile.Repositories {
		totalStars += repo.Stars
	}
	score += float64(totalStars) / 100

	// Language diversity
	score += float64(len(profile.Languages)) / 10

	// Organization involvement
	score += float64(len(profile.Organizations)) / 5

	// Consistency
	score += profile.Contributions.ConsistencyScore

	// Normalize to 0-1 scale
	if score > 1 {
		score = 1
	}

	return score
}

// recommendRoles suggests suitable roles based on the profile
func (a *Analyzer) recommendRoles(profile *UserProfile) []string {
	var roles []string

	careerLevel := profile.Insights.CareerLevel
	if careerLevel == "" {
		careerLevel = "mid" // Default fallback if career level is empty
	}
	primaryLangs := profile.Skills.PrimaryLanguages

	// Backend roles
	backendLangs := []string{"Go", "Python", "Java", "C#", "Node.js", "Rust"}
	if a.hasAnyLanguage(primaryLangs, backendLangs) {
		switch careerLevel {
		case "principal":
			roles = append(roles, "Principal Engineer", "Staff Engineer", "Engineering Manager")
		case "senior":
			roles = append(roles, "Senior Backend Engineer", "Backend Team Lead", "Staff Engineer")
		case "junior":
			roles = append(roles, "Junior Backend Developer", "Software Engineer")
		default:
			roles = append(roles, "Backend Developer", "Software Engineer")
		}
	}

	// Frontend roles
	frontendLangs := []string{"JavaScript", "TypeScript", "React", "Vue", "Angular"}
	if a.hasAnyLanguage(primaryLangs, frontendLangs) {
		switch careerLevel {
		case "principal":
			roles = append(roles, "Principal Frontend Engineer", "Frontend Architect")
		case "senior":
			roles = append(roles, "Senior Frontend Engineer", "Frontend Team Lead")
		case "junior":
			roles = append(roles, "Junior Frontend Developer", "UI Developer")
		default:
			roles = append(roles, "Frontend Developer", "UI Developer")
		}
	}

	// Full-stack roles
	if a.hasAnyLanguage(primaryLangs, backendLangs) && a.hasAnyLanguage(primaryLangs, frontendLangs) {
		switch careerLevel {
		case "principal":
			roles = append(roles, "Principal Engineer", "Full-Stack Architect", "Technical Lead")
		case "senior":
			roles = append(roles, "Senior Full-Stack Engineer", "Technical Lead")
		case "junior":
			roles = append(roles, "Junior Full-Stack Developer", "Software Engineer")
		default:
			roles = append(roles, "Full-Stack Developer", "Software Engineer")
		}
	}

	// DevOps roles with career level consideration
	if len(profile.Skills.DevOpsSkills) >= 3 {
		switch careerLevel {
		case "principal":
			roles = append(roles, "Principal SRE", "DevOps Architect", "Platform Lead")
		case "senior":
			roles = append(roles, "Senior DevOps Engineer", "Senior SRE", "Platform Engineer")
		case "junior":
			roles = append(roles, "Junior DevOps Engineer", "Platform Engineer")
		default:
			roles = append(roles, "DevOps Engineer", "Platform Engineer", "SRE")
		}
	}

	// Ensure we always have at least some role recommendations
	if len(roles) == 0 {
		switch careerLevel {
		case "principal":
			roles = append(roles, "Principal Engineer", "Staff Engineer", "Technical Lead")
		case "senior":
			roles = append(roles, "Senior Software Engineer", "Technical Lead")
		case "junior":
			roles = append(roles, "Junior Software Engineer", "Software Developer")
		default:
			roles = append(roles, "Software Engineer", "Software Developer")
		}
	}

	return roles
}

// identifyStrengthsAndGrowthAreas identifies user's strengths and areas for growth
func (a *Analyzer) identifyStrengthsAndGrowthAreas(profile *UserProfile) ([]string, []string) {
	var strengths, growthAreas []string

	// Strengths based on language proficiency
	for _, lang := range profile.Languages {
		if lang.Percentage > 25 {
			strengths = append(strengths, lang.Language+" development")
		}
	}

	// Strengths based on repository ownership
	ownedRepos := 0
	for _, repo := range profile.Repositories {
		if repo.IsOwner {
			ownedRepos++
		}
	}
	if ownedRepos >= 10 {
		strengths = append(strengths, "Project creation and ownership")
	}

	// Strengths based on collaboration
	if len(profile.Organizations) >= 3 {
		strengths = append(strengths, "Cross-team collaboration")
	}

	// Growth areas based on missing common skills
	commonSkills := []string{"testing", "documentation", "ci/cd", "monitoring"}
	hasSkill := make(map[string]bool)

	for _, repo := range profile.Repositories {
		for _, topic := range repo.Topics {
			hasSkill[strings.ToLower(topic)] = true
		}
	}

	for _, skill := range commonSkills {
		if !hasSkill[skill] && !hasSkill[skill+"-testing"] && !hasSkill[skill+"-automation"] {
			growthAreas = append(growthAreas, strings.Title(skill))
		}
	}

	return strengths, growthAreas
}

// hasAnyLanguage checks if any of the target languages are in the user's primary languages
func (a *Analyzer) hasAnyLanguage(userLangs, targetLangs []string) bool {
	for _, userLang := range userLangs {
		for _, targetLang := range targetLangs {
			if strings.EqualFold(userLang, targetLang) {
				return true
			}
		}
	}
	return false
}

// analyzeDockerHub analyzes the user's Docker Hub profile
func (a *Analyzer) analyzeDockerHub(ctx context.Context, username string, profile *UserProfile) error {
	log.Printf("Analyzing Docker Hub profile for user: %s", username)

	// Analyze Docker Hub profile
	dockerProfile, err := a.dockerClient.AnalyzeDockerProfile(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to analyze Docker Hub profile: %w", err)
	}

	// Convert to simplified profile structure for integration
	if dockerProfile != nil && len(dockerProfile.Repositories) > 0 {
		profile.DockerHubProfile = &DockerHubProfile{
			Username:            dockerProfile.Username,
			TotalDownloads:      dockerProfile.ImpactMetrics.TotalDownloads,
			TotalImages:         dockerProfile.TotalImages,
			TopRepositories:     dockerProfile.ImpactMetrics.TopRepositories,
			MostDownloadedImage: dockerProfile.ImpactMetrics.MostDownloadedImage,
			CommunityImpact:     dockerProfile.ImpactMetrics.CommunityImpact,
			ExperienceYears:     dockerProfile.ContainerExpertise.ExperienceYears,
			ProficiencyLevel:    dockerProfile.ContainerExpertise.ProficiencyLevel,
		}

		// Update last activity from most recent repository
		for _, repo := range dockerProfile.Repositories {
			if repo.LastUpdated.After(profile.DockerHubProfile.LastActivity) {
				profile.DockerHubProfile.LastActivity = repo.LastUpdated
			}
		}

		log.Printf("Docker Hub analysis complete: %d images, %d total downloads",
			dockerProfile.TotalImages, dockerProfile.ImpactMetrics.TotalDownloads)
	}

	return nil
}

// analyzeDiscourseProfile analyzes the user's Discourse community engagement
func (a *Analyzer) analyzeDiscourseProfile(ctx context.Context, username, discourseUsername string, profile *UserProfile) error {
	// Use custom Discourse username if provided, otherwise fall back to GitHub username
	targetUsername := username
	if discourseUsername != "" {
		targetUsername = discourseUsername
		log.Printf("Analyzing Discourse profile for user: %s (using custom Discourse username: %s)", username, discourseUsername)
	} else {
		log.Printf("Analyzing Discourse profile for user: %s", username)
	}

	// For Jenkins community, try common username variations
	usernamesToTry := []string{targetUsername}

	// Add common variations if not already present
	if targetUsername != strings.ToLower(targetUsername) {
		usernamesToTry = append(usernamesToTry, strings.ToLower(targetUsername))
	}

	// Try with underscores instead of hyphens and vice versa
	if strings.Contains(targetUsername, "-") {
		usernamesToTry = append(usernamesToTry, strings.ReplaceAll(targetUsername, "-", "_"))
	}
	if strings.Contains(targetUsername, "_") {
		usernamesToTry = append(usernamesToTry, strings.ReplaceAll(targetUsername, "_", "-"))
	}

	var discourseProfile *discourse.DiscourseProfile
	var err error

	// Try each username variation
	for _, tryUsername := range usernamesToTry {
		discourseProfile, err = a.discourseClient.AnalyzeDiscourseProfile(ctx, tryUsername)
		if err == nil && discourseProfile != nil {
			log.Printf("Found Discourse profile for username: %s", tryUsername)
			break
		}
		log.Printf("Discourse profile not found for username: %s (%v)", tryUsername, err)
	}

	if discourseProfile == nil {
		return fmt.Errorf("no Discourse profile found for any username variation")
	}

	// Convert to simplified profile structure for integration
	profile.DiscourseProfile = &DiscourseProfile{
		Username:            discourseProfile.Username,
		DisplayName:         discourseProfile.DisplayName,
		ProfileURL:          discourseProfile.ProfileURL,
		CommunityURL:        discourseProfile.CommunityURL,
		JoinedDate:          discourseProfile.JoinedDate,
		LastActivity:        discourseProfile.LastActivity,
		PostCount:           discourseProfile.PostCount,
		TopicCount:          discourseProfile.TopicCount,
		LikesReceived:       discourseProfile.LikesReceived,
		LikesGiven:          discourseProfile.LikesGiven,
		SolutionsCount:      discourseProfile.SolutionsCount,
		DaysActive:          discourseProfile.DaysActive,
		ReadingTime:         discourseProfile.ReadingTime,
		TrustLevel:          discourseProfile.TrustLevel,
		BadgeCount:          discourseProfile.BadgeCount,
		CommunityMetrics:    discourseProfile.CommunityMetrics,
		ExpertiseAreas:      discourseProfile.ExpertiseAreas,
		MentorshipSignals:   discourseProfile.MentorshipSignals,
		CategoryActivity:    discourseProfile.CategoryActivity,
	}

	log.Printf("Discourse analysis complete: %d posts, %d solutions, trust level %d",
		discourseProfile.PostCount, discourseProfile.SolutionsCount, discourseProfile.TrustLevel)

	return nil
}

// ProgressData represents saved analysis progress
type ProgressData struct {
	Username     string       `json:"username"`
	LastStep     int          `json:"lastStep"`
	SavedAt      time.Time    `json:"savedAt"`
	UserProfile  *UserProfile `json:"userProfile"`
}

// saveProgress saves the current analysis progress
func (a *Analyzer) saveProgress(username string, profile *UserProfile, step int) error {
	// Create progress directory if it doesn't exist
	if err := os.MkdirAll(a.saveProgressDir, 0755); err != nil {
		return fmt.Errorf("failed to create progress directory: %w", err)
	}

	progressData := ProgressData{
		Username:    username,
		LastStep:    step,
		SavedAt:     time.Now(),
		UserProfile: profile,
	}

	filename := filepath.Join(a.saveProgressDir, fmt.Sprintf("%s_progress.json", username))
	data, err := json.MarshalIndent(progressData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal progress data: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write progress file: %w", err)
	}

	log.Printf("Progress saved after step %d for user: %s", step, username)
	return nil
}

// tryResumeProgress attempts to resume from saved progress
func (a *Analyzer) tryResumeProgress(username string) (*UserProfile, int) {
	filename := filepath.Join(a.saveProgressDir, fmt.Sprintf("%s_progress.json", username))

	data, err := os.ReadFile(filename)
	if err != nil {
		// No progress file exists, start from beginning
		return nil, 1
	}

	var progressData ProgressData
	if err := json.Unmarshal(data, &progressData); err != nil {
		log.Printf("Warning: Failed to parse progress file, starting from beginning: %v", err)
		return nil, 1
	}

	// Check if progress file is too old (older than 1 day)
	if time.Since(progressData.SavedAt) > 24*time.Hour {
		log.Printf("Progress file is older than 24 hours, starting fresh")
		a.cleanupProgress(username)
		return nil, 1
	}

	log.Printf("Resuming analysis for user %s from step %d (saved at %s)",
		username, progressData.LastStep+1, progressData.SavedAt.Format("2006-01-02 15:04:05"))

	return progressData.UserProfile, progressData.LastStep + 1
}

// cleanupProgress removes the progress file after successful completion
func (a *Analyzer) cleanupProgress(username string) {
	filename := filepath.Join(a.saveProgressDir, fmt.Sprintf("%s_progress.json", username))
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: Failed to clean up progress file: %v", err)
	} else {
		log.Printf("Progress file cleaned up for user: %s", username)
	}
}

// saveToCache saves completed analysis to permanent cache for template reuse
func (a *Analyzer) saveToCache(username string, profile *UserProfile) error {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(a.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	filename := filepath.Join(a.cacheDir, fmt.Sprintf("%s_analysis.json", username))
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profile to JSON: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	log.Printf("Analysis cached for user: %s", username)
	return nil
}

// tryLoadFromCache attempts to load completed analysis from cache
func (a *Analyzer) tryLoadFromCache(username string) *UserProfile {
	filename := filepath.Join(a.cacheDir, fmt.Sprintf("%s_analysis.json", username))

	data, err := os.ReadFile(filename)
	if err != nil {
		// No cache file exists
		return nil
	}

	var profile UserProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		log.Printf("Warning: Failed to parse cache file, will re-analyze: %v", err)
		return nil
	}

	// Check if cache is too old (older than 7 days)
	if time.Since(profile.LastAnalyzed) > 7*24*time.Hour {
		log.Printf("Cache is older than 7 days, will re-analyze")
		return nil
	}

	return &profile
}

// GetGitHubRateLimitStatus returns current GitHub API rate limit status for monitoring
func (a *Analyzer) GetGitHubRateLimitStatus() github.RateLimitInfo {
	return a.client.GetRateLimitStatus()
}

// analyzeDockerConfig analyzes a repository for Docker configuration and expertise
func (a *Analyzer) analyzeDockerConfig(ctx context.Context, fullName string) *DockerConfig {
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 {
		return nil
	}
	owner, repo := parts[0], parts[1]

	// Fetch repository contents
	contents, err := a.client.FetchRepositoryContents(ctx, owner, repo)
	if err != nil {
		log.Printf("Failed to fetch contents for %s: %v", fullName, err)
		return nil
	}

	if len(contents) == 0 {
		return nil
	}

	config := &DockerConfig{
		DockerFiles:  []DockerFile{},
		ComposeFiles: []string{},
		BakeFiles:    []string{},
		DockerPatterns: []string{},
	}

	var hasDockerFiles bool

	// Scan repository contents for Docker-related files
	for _, item := range contents {
		fileName := strings.ToLower(item.Name)

		switch {
		case fileName == "dockerfile" || strings.HasSuffix(fileName, ".dockerfile"):
			config.HasDockerfile = true
			hasDockerFiles = true
			dockerFile := a.analyzeDockerFile(item)
			config.DockerFiles = append(config.DockerFiles, dockerFile)

		case fileName == "docker-compose.yml" || fileName == "docker-compose.yaml" ||
			 strings.Contains(fileName, "compose") && (strings.HasSuffix(fileName, ".yml") || strings.HasSuffix(fileName, ".yaml")):
			config.HasCompose = true
			hasDockerFiles = true
			config.ComposeFiles = append(config.ComposeFiles, item.Name)

		case fileName == "docker-bake.hcl" || fileName == "docker-bake.json" || strings.Contains(fileName, "bake"):
			config.HasBakeFile = true
			hasDockerFiles = true
			config.BakeFiles = append(config.BakeFiles, item.Name)

		case fileName == ".dockerignore":
			config.HasDockerIgnore = true
			hasDockerFiles = true
		}
	}

	// Only return config if we found Docker-related files
	if !hasDockerFiles {
		return nil
	}

	// Calculate complexity score and expertise
	config.ComplexityScore = a.calculateDockerComplexity(config)
	config.ContainerExpertise = a.assessDockerExpertise(config)
	config.DockerPatterns = a.identifyDockerPatterns(config)

	return config
}

// analyzeDockerFile analyzes a specific Dockerfile for complexity and patterns
func (a *Analyzer) analyzeDockerFile(item github.RepositoryContentResponse) DockerFile {
	dockerFile := DockerFile{
		Path:              item.Path,
		Instructions:      []string{},
		BestPractices:     []string{},
		SecurityPatterns:  []string{},
		OptimizationLevel: "basic",
	}

	// Note: We're not fetching file contents to avoid API rate limits
	// Instead, we'll infer complexity from file size and name

	if item.Size > 1000 {
		dockerFile.OptimizationLevel = "intermediate"
		dockerFile.BestPractices = append(dockerFile.BestPractices, "complex-dockerfile")
	}

	if item.Size > 3000 {
		dockerFile.OptimizationLevel = "advanced"
		dockerFile.IsMultiStage = true
		dockerFile.StageCount = 2 // Estimate based on size
	}

	// Infer from filename patterns
	if strings.Contains(strings.ToLower(item.Name), "prod") {
		dockerFile.SecurityPatterns = append(dockerFile.SecurityPatterns, "production-optimized")
	}

	return dockerFile
}

// calculateDockerComplexity calculates overall Docker expertise complexity score (0-10)
func (a *Analyzer) calculateDockerComplexity(config *DockerConfig) float64 {
	score := 0.0

	// Base points for different file types
	if config.HasDockerfile {
		score += 2.0
	}
	if config.HasCompose {
		score += 2.5
	}
	if config.HasBakeFile {
		score += 3.0 // More advanced
	}
	if config.HasDockerIgnore {
		score += 0.5
	}

	// Additional points for multiple files
	score += float64(len(config.DockerFiles)) * 0.5
	score += float64(len(config.ComposeFiles)) * 0.3
	score += float64(len(config.BakeFiles)) * 1.0

	// Advanced dockerfile patterns
	for _, dockerFile := range config.DockerFiles {
		switch dockerFile.OptimizationLevel {
		case "advanced":
			score += 2.0
		case "intermediate":
			score += 1.0
		}
		if dockerFile.IsMultiStage {
			score += 1.5
		}
	}

	// Cap at 10
	if score > 10.0 {
		score = 10.0
	}

	return score
}

// assessDockerExpertise determines the user's Docker expertise level based on configuration
func (a *Analyzer) assessDockerExpertise(config *DockerConfig) DockerExpertiseLevel {
	expertise := DockerExpertiseLevel{
		Evidence:         []string{},
		TechnologiesUsed: []string{},
		AdvancedPatterns: []string{},
	}

	score := config.ComplexityScore

	// Determine expertise level
	switch {
	case score >= 7.0:
		expertise.Level = "expert"
		expertise.ProductionReadiness = true
	case score >= 5.0:
		expertise.Level = "advanced"
		expertise.ProductionReadiness = true
	case score >= 3.0:
		expertise.Level = "intermediate"
	default:
		expertise.Level = "beginner"
	}

	// Collect evidence
	if config.HasDockerfile {
		expertise.Evidence = append(expertise.Evidence, "dockerfile-usage")
		expertise.TechnologiesUsed = append(expertise.TechnologiesUsed, "docker")
	}
	if config.HasCompose {
		expertise.Evidence = append(expertise.Evidence, "docker-compose-usage")
		expertise.TechnologiesUsed = append(expertise.TechnologiesUsed, "docker-compose")
	}
	if config.HasBakeFile {
		expertise.Evidence = append(expertise.Evidence, "docker-buildx-bake")
		expertise.TechnologiesUsed = append(expertise.TechnologiesUsed, "docker-buildx")
		expertise.AdvancedPatterns = append(expertise.AdvancedPatterns, "buildx-bake")
	}

	// Advanced patterns detection
	for _, dockerFile := range config.DockerFiles {
		if dockerFile.IsMultiStage {
			expertise.AdvancedPatterns = append(expertise.AdvancedPatterns, "multi-stage-builds")
		}
		if dockerFile.OptimizationLevel == "advanced" {
			expertise.AdvancedPatterns = append(expertise.AdvancedPatterns, "optimized-builds")
		}
	}

	return expertise
}

// identifyDockerPatterns identifies specific Docker usage patterns
func (a *Analyzer) identifyDockerPatterns(config *DockerConfig) []string {
	patterns := []string{}

	if config.HasDockerfile && config.HasCompose {
		patterns = append(patterns, "full-docker-stack")
	}

	if config.HasBakeFile {
		patterns = append(patterns, "advanced-build-system")
	}

	if len(config.DockerFiles) > 1 {
		patterns = append(patterns, "multi-dockerfile-project")
	}

	if config.HasDockerIgnore {
		patterns = append(patterns, "optimized-build-context")
	}

	// Multi-stage detection
	hasMultiStage := false
	for _, dockerFile := range config.DockerFiles {
		if dockerFile.IsMultiStage {
			hasMultiStage = true
			break
		}
	}
	if hasMultiStage {
		patterns = append(patterns, "multi-stage-optimization")
	}

	return patterns
}

// categorizeDockerSkills analyzes Docker configuration and adds appropriate skills
func (a *Analyzer) categorizeDockerSkills(repo RepositoryProfile, techMap map[string]*TechnologySkill, skills *SkillProfile) {
	config := repo.DockerConfig
	if config == nil {
		return
	}

	// Add Docker as a DevOps skill with confidence based on expertise level
	confidence := 0.7 // Base confidence
	switch config.ContainerExpertise.Level {
	case "expert":
		confidence = 0.95
	case "advanced":
		confidence = 0.9
	case "intermediate":
		confidence = 0.8
	case "beginner":
		confidence = 0.6
	}

	// Add Docker skill
	dockerSkill := TechnologySkill{
		Name:             "Docker",
		Confidence:       confidence,
		Evidence:         []string{repo.FullName},
		ProjectCount:     1,
		ProficiencyLevel: config.ContainerExpertise.Level,
	}

	// Add evidence from Docker configuration
	dockerSkill.Evidence = append(dockerSkill.Evidence, config.ContainerExpertise.Evidence...)
	skills.DevOpsSkills = append(skills.DevOpsSkills, dockerSkill)

	// Also add as cloud platform skill if expertise is high
	if confidence >= 0.8 {
		skills.CloudPlatforms = append(skills.CloudPlatforms, dockerSkill)
	}

	// Add specific Docker technologies based on files found
	if config.HasCompose {
		composeSkill := TechnologySkill{
			Name:             "Docker Compose",
			Confidence:       confidence,
			Evidence:         []string{repo.FullName},
			ProjectCount:     1,
			ProficiencyLevel: config.ContainerExpertise.Level,
		}
		skills.DevOpsSkills = append(skills.DevOpsSkills, composeSkill)
		skills.Tools = append(skills.Tools, composeSkill)
	}

	if config.HasBakeFile {
		bakeSkill := TechnologySkill{
			Name:             "Docker Buildx",
			Confidence:       0.9, // High confidence for advanced tool
			Evidence:         []string{repo.FullName},
			ProjectCount:     1,
			ProficiencyLevel: "advanced", // Using bake files indicates advanced usage
		}
		skills.DevOpsSkills = append(skills.DevOpsSkills, bakeSkill)
		skills.Tools = append(skills.Tools, bakeSkill)
	}

	// Add containerization as a technical area (check if it already exists to avoid duplicates)
	containerFound := false
	for i := range skills.TechnicalAreas {
		if skills.TechnicalAreas[i].Area == "Containerization" {
			// Update existing entry with higher competency if found
			if confidence > skills.TechnicalAreas[i].Competency {
				skills.TechnicalAreas[i].Competency = confidence
			}
			skills.TechnicalAreas[i].ProjectCount++
			// Merge technologies
			for _, tech := range config.ContainerExpertise.TechnologiesUsed {
				exists := false
				for _, existing := range skills.TechnicalAreas[i].Technologies {
					if existing == tech {
						exists = true
						break
					}
				}
				if !exists {
					skills.TechnicalAreas[i].Technologies = append(skills.TechnicalAreas[i].Technologies, tech)
				}
			}
			containerFound = true
			break
		}
	}

	if !containerFound {
		containerArea := TechnicalArea{
			Area:         "Containerization",
			Competency:   confidence,
			Technologies: config.ContainerExpertise.TechnologiesUsed,
			ProjectCount: 1,
			YearsActive:  1.0, // Estimate based on repository activity
		}
		skills.TechnicalAreas = append(skills.TechnicalAreas, containerArea)
	}
}

// logAnalysisSummary provides a summary of what data was successfully collected
func (a *Analyzer) logAnalysisSummary(profile *UserProfile) {
	log.Printf("=== Analysis Summary for %s ===", profile.Username)

	// Repository analysis
	repoCount := len(profile.Repositories)
	dockerRepoCount := 0
	for _, repo := range profile.Repositories {
		if repo.DockerConfig != nil {
			dockerRepoCount++
		}
	}

	log.Printf("📦 Repositories: %d total", repoCount)
	if dockerRepoCount > 0 {
		log.Printf("🐳 Docker repositories detected: %d", dockerRepoCount)
	}

	// Language analysis
	if len(profile.Languages) > 0 {
		log.Printf("💻 Programming languages: %d", len(profile.Languages))
		if len(profile.Languages) >= 3 {
			log.Printf("    Primary languages: %s, %s, %s",
				profile.Languages[0].Language,
				profile.Languages[1].Language,
				profile.Languages[2].Language)
		}
	}

	// Skills analysis
	skillCounts := map[string]int{
		"DevOps": len(profile.Skills.DevOpsSkills),
		"Cloud": len(profile.Skills.CloudPlatforms),
		"Tools": len(profile.Skills.Tools),
		"Frameworks": len(profile.Skills.Frameworks),
		"Technical Areas": len(profile.Skills.TechnicalAreas),
	}

	log.Printf("🛠️  Skills detected:")
	for category, count := range skillCounts {
		if count > 0 {
			log.Printf("    %s: %d", category, count)
		}
	}

	// Organization analysis
	if len(profile.Organizations) > 0 {
		log.Printf("🏢 Organizations: %d", len(profile.Organizations))
	}

	// Docker Hub analysis
	if profile.DockerHubProfile != nil {
		log.Printf("🐳 Docker Hub: %d images, %.1fM downloads, %s level",
			profile.DockerHubProfile.TotalImages,
			float64(profile.DockerHubProfile.TotalDownloads)/1000000,
			profile.DockerHubProfile.ProficiencyLevel)
	}

	// Overall metrics
	log.Printf("📊 Overall: %d commits, %d repos, %.1f impact score",
		profile.Contributions.TotalCommits,
		repoCount,
		profile.Insights.OverallImpactScore)

	log.Printf("✅ Profile analysis ready for template generation")
}