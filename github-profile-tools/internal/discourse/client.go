package discourse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Client handles Discourse API interactions
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Discourse client
func NewClient() *Client {
	return &Client{
		baseURL: "https://community.jenkins.io",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AnalyzeDiscourseProfile performs comprehensive analysis of a user's Discourse engagement
func (c *Client) AnalyzeDiscourseProfile(ctx context.Context, username string) (*DiscourseProfile, error) {
	// Step 1: Get user information
	userResp, err := c.fetchUser(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	user := userResp.User

	profile := &DiscourseProfile{
		Username:      user.Username,
		DisplayName:   user.Name,
		ProfileURL:    fmt.Sprintf("%s/u/%s", c.baseURL, username),
		CommunityURL:  c.baseURL,
		JoinedDate:    user.CreatedAt,
		LastActivity:  user.LastSeenAt,
		PostCount:     user.PostCount,
		TopicCount:    user.TopicCount,
		LikesReceived: user.LikesReceived,
		LikesGiven:    user.LikesGiven,
		DaysActive:    user.DaysVisited,
		TrustLevel:    user.TrustLevel,
		BadgeCount:    user.BadgeCount,
	}

	// Step 2: Fetch user badges
	if err := c.fetchUserBadges(ctx, username, profile); err != nil {
		// Continue even if badges fail - not critical
		fmt.Printf("Warning: Failed to fetch badges for %s: %v\n", username, err)
	}

	// Step 3: Fetch user posts for analysis
	if err := c.fetchUserPosts(ctx, username, profile); err != nil {
		// Continue even if posts fail - not critical
		fmt.Printf("Warning: Failed to fetch posts for %s: %v\n", username, err)
	}

	// Step 4: Fetch user topics
	if err := c.fetchUserTopics(ctx, username, profile); err != nil {
		// Continue even if topics fail - not critical
		fmt.Printf("Warning: Failed to fetch topics for %s: %v\n", username, err)
	}

	// Step 5: Fetch categories for context
	categories, err := c.fetchCategories(ctx)
	if err != nil {
		// Continue without categories - analysis will be less detailed
		fmt.Printf("Warning: Failed to fetch categories: %v\n", err)
	}

	// Step 6: Analyze engagement and generate insights
	c.analyzeEngagement(profile, categories)

	return profile, nil
}

// fetchUser retrieves basic user information
func (c *Client) fetchUser(ctx context.Context, username string) (*DiscourseUserResponse, error) {
	url := fmt.Sprintf("%s/users/%s.json", c.baseURL, username)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var userResp DiscourseUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &userResp, nil
}

// fetchUserBadges retrieves user's badges
func (c *Client) fetchUserBadges(ctx context.Context, username string, profile *DiscourseProfile) error {
	url := fmt.Sprintf("%s/user-badges/%s.json", c.baseURL, username)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("badges request failed with status %d", resp.StatusCode)
	}

	var badgesResp struct {
		UserBadges []struct {
			ID          int       `json:"id"`
			GrantedAt   time.Time `json:"granted_at"`
			Count       int       `json:"count"`
			Badge       struct {
				ID          int    `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
				BadgeTypeID int    `json:"badge_type_id"`
			} `json:"badge"`
		} `json:"user_badges"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&badgesResp); err != nil {
		return fmt.Errorf("failed to decode badges response: %w", err)
	}

	// Convert to our badge structure
	for _, ub := range badgesResp.UserBadges {
		badgeType := "bronze"
		category := "participation"

		switch ub.Badge.BadgeTypeID {
		case 1:
			badgeType = "gold"
			category = "leadership"
		case 2:
			badgeType = "silver"
			category = "expertise"
		case 3:
			badgeType = "bronze"
		}

		// Categorize badges based on name
		badgeName := strings.ToLower(ub.Badge.Name)
		if strings.Contains(badgeName, "leader") || strings.Contains(badgeName, "moderator") {
			category = "leadership"
		} else if strings.Contains(badgeName, "expert") || strings.Contains(badgeName, "guru") {
			category = "expertise"
		} else if strings.Contains(badgeName, "special") || strings.Contains(badgeName, "anniversary") {
			category = "special"
		}

		badge := DiscourseBadge{
			ID:          ub.Badge.ID,
			Name:        ub.Badge.Name,
			Description: ub.Badge.Description,
			BadgeType:   badgeType,
			GrantedAt:   ub.GrantedAt,
			Count:       ub.Count,
			Category:    category,
		}

		profile.Badges = append(profile.Badges, badge)
	}

	return nil
}

// fetchUserPosts retrieves user's recent posts for analysis
func (c *Client) fetchUserPosts(ctx context.Context, username string, profile *DiscourseProfile) error {
	url := fmt.Sprintf("%s/user_actions.json?username=%s&filter=5&limit=50", c.baseURL, username) // filter=5 is posts

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("posts request failed with status %d", resp.StatusCode)
	}

	var postsResp DiscoursePostsResponse
	if err := json.NewDecoder(resp.Body).Decode(&postsResp); err != nil {
		return fmt.Errorf("failed to decode posts response: %w", err)
	}

	// Convert to our post structure and analyze
	for _, latestPost := range postsResp.LatestPosts {
		post := DiscoursePost{
			ID:           latestPost.ID,
			TopicID:      latestPost.TopicID,
			TopicTitle:   latestPost.TopicTitle,
			PostNumber:   latestPost.PostNumber,
			Content:      latestPost.Cooked,
			CreatedAt:    latestPost.CreatedAt,
			LikeCount:    latestPost.LikeCount,
			ReplyCount:   latestPost.ReplyCount,
			IsSolution:   latestPost.AcceptedAnswer,
			CategoryID:   latestPost.CategoryID,
			CategoryName: "", // Will be filled later when categories map is available
		}

		// Analyze post for helpfulness and technical depth
		post.HelpfulnessScore = c.analyzeHelpfulness(post.Content, post.LikeCount, post.IsSolution)
		post.TechnicalDepth = c.analyzeTechnicalDepth(post.Content)
		post.ClarityScore = c.analyzeClarityScore(post.Content)

		profile.TopPosts = append(profile.TopPosts, post)

		// Update solution count
		if post.IsSolution {
			profile.SolutionsCount++
		}
	}

	// Sort posts by helpfulness score
	sort.Slice(profile.TopPosts, func(i, j int) bool {
		return profile.TopPosts[i].HelpfulnessScore > profile.TopPosts[j].HelpfulnessScore
	})

	// Keep only top 10 posts
	if len(profile.TopPosts) > 10 {
		profile.TopPosts = profile.TopPosts[:10]
	}

	return nil
}

// fetchUserTopics retrieves user's topics
func (c *Client) fetchUserTopics(ctx context.Context, username string, profile *DiscourseProfile) error {
	url := fmt.Sprintf("%s/user_actions.json?username=%s&filter=4&limit=20", c.baseURL, username) // filter=4 is topics

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("topics request failed with status %d", resp.StatusCode)
	}

	var topicsResp DiscourseTopicsResponse
	if err := json.NewDecoder(resp.Body).Decode(&topicsResp); err != nil {
		return fmt.Errorf("failed to decode topics response: %w", err)
	}

	// Convert to our topic structure
	for _, topic := range topicsResp.TopicList.Topics {
		discourseTopic := DiscourseTopic{
			ID:            topic.ID,
			Title:         topic.Title,
			FancyTitle:    topic.FancyTitle,
			PostsCount:    topic.PostsCount,
			ReplyCount:    topic.ReplyCount,
			CreatedAt:     topic.CreatedAt,
			LastPostedAt:  topic.LastPostedAt,
			Views:         topic.Views,
			LikeCount:     topic.LikeCount,
			CategoryID:    topic.CategoryID,
			CategoryName:  "", // Will be filled later when categories map is available
			Tags:          topic.Tags,
			IsPinned:      topic.Pinned,
			IsClosed:      topic.Closed,
		}

		// Calculate engagement score
		discourseTopic.EngagementScore = c.calculateEngagementScore(discourseTopic)

		// Determine impact level
		if discourseTopic.Views > 1000 && discourseTopic.LikeCount > 10 {
			discourseTopic.ImpactLevel = "high"
		} else if discourseTopic.Views > 100 && discourseTopic.LikeCount > 3 {
			discourseTopic.ImpactLevel = "medium"
		} else {
			discourseTopic.ImpactLevel = "low"
		}

		profile.TopTopics = append(profile.TopTopics, discourseTopic)
	}

	// Sort topics by engagement score
	sort.Slice(profile.TopTopics, func(i, j int) bool {
		return profile.TopTopics[i].EngagementScore > profile.TopTopics[j].EngagementScore
	})

	return nil
}

// fetchCategories retrieves forum categories for context
func (c *Client) fetchCategories(ctx context.Context) (map[int]string, error) {
	url := fmt.Sprintf("%s/categories.json", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("categories request failed with status %d", resp.StatusCode)
	}

	var categoriesResp DiscourseCategoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&categoriesResp); err != nil {
		return nil, fmt.Errorf("failed to decode categories response: %w", err)
	}

	categories := make(map[int]string)
	for _, cat := range categoriesResp.CategoryList.Categories {
		categories[cat.ID] = cat.Name
	}

	return categories, nil
}

// analyzeEngagement performs comprehensive analysis of user engagement
func (c *Client) analyzeEngagement(profile *DiscourseProfile, categories map[int]string) {
	// Calculate reading time (estimated from days active and posts)
	if profile.DaysActive > 0 {
		profile.ReadingTime = profile.DaysActive * 30 // Rough estimate: 30 min per active day
	}

	// Update category names in posts and topics now that we have the categories map
	c.updateCategoryNames(profile, categories)

	// Analyze category activity
	c.analyzeCategoryActivity(profile, categories)

	// Generate community metrics
	profile.CommunityMetrics = c.generateCommunityMetrics(profile)

	// Identify expertise areas
	profile.ExpertiseAreas = c.identifyExpertiseAreas(profile)

	// Analyze mentorship signals
	profile.MentorshipSignals = c.analyzeMentorshipSignals(profile)
}

// updateCategoryNames updates category names in posts and topics using the categories map
func (c *Client) updateCategoryNames(profile *DiscourseProfile, categories map[int]string) {
	// Update category names in posts
	for i := range profile.TopPosts {
		if profile.TopPosts[i].CategoryName == "" && profile.TopPosts[i].CategoryID > 0 {
			if categoryName, exists := categories[profile.TopPosts[i].CategoryID]; exists {
				profile.TopPosts[i].CategoryName = categoryName
			}
		}
	}

	// Update category names in topics
	for i := range profile.TopTopics {
		if profile.TopTopics[i].CategoryName == "" && profile.TopTopics[i].CategoryID > 0 {
			if categoryName, exists := categories[profile.TopTopics[i].CategoryID]; exists {
				profile.TopTopics[i].CategoryName = categoryName
			}
		}
	}
}

// analyzeCategoryActivity analyzes engagement within specific categories
func (c *Client) analyzeCategoryActivity(profile *DiscourseProfile, categories map[int]string) {
	categoryMap := make(map[string]*CategoryEngagement)

	// Analyze posts by category
	for _, post := range profile.TopPosts {
		if post.CategoryName == "" {
			continue
		}

		if categoryMap[post.CategoryName] == nil {
			categoryMap[post.CategoryName] = &CategoryEngagement{
				CategoryName: post.CategoryName,
			}
		}

		engagement := categoryMap[post.CategoryName]
		engagement.PostCount++
		engagement.LikesReceived += post.LikeCount

		if post.IsSolution {
			engagement.SolutionsCount++
		}
	}

	// Analyze topics by category
	for _, topic := range profile.TopTopics {
		if topic.CategoryName == "" {
			continue
		}

		if categoryMap[topic.CategoryName] == nil {
			categoryMap[topic.CategoryName] = &CategoryEngagement{
				CategoryName: topic.CategoryName,
			}
		}

		engagement := categoryMap[topic.CategoryName]
		engagement.TopicCount++
		engagement.LikesReceived += topic.LikeCount
	}

	// Convert map to slice and calculate metrics
	for _, engagement := range categoryMap {
		// Calculate expertise level (0-10 scale)
		expertiseScore := float64(engagement.PostCount)*0.5 + float64(engagement.SolutionsCount)*2.0 + float64(engagement.LikesReceived)*0.1
		if expertiseScore > 10 {
			expertiseScore = 10
		}
		engagement.ExpertiseLevel = expertiseScore

		// Calculate influence score
		totalActivity := engagement.PostCount + engagement.TopicCount
		if totalActivity > 0 {
			engagement.InfluenceScore = (float64(engagement.LikesReceived) / float64(totalActivity)) * (expertiseScore / 10)
		}

		profile.CategoryActivity = append(profile.CategoryActivity, *engagement)
	}

	// Sort categories by expertise level
	sort.Slice(profile.CategoryActivity, func(i, j int) bool {
		return profile.CategoryActivity[i].ExpertiseLevel > profile.CategoryActivity[j].ExpertiseLevel
	})
}

// generateCommunityMetrics generates leadership and impact metrics
func (c *Client) generateCommunityMetrics(profile *DiscourseProfile) CommunityLeadershipMetrics {
	metrics := CommunityLeadershipMetrics{}

	// Calculate helpfulness ratio
	if profile.PostCount > 0 {
		metrics.HelpfulnessRatio = float64(profile.SolutionsCount) / float64(profile.PostCount)
	}

	// Calculate engagement consistency (based on days active vs account age)
	accountAge := time.Since(profile.JoinedDate).Hours() / (24 * 365.25)
	if accountAge > 0 {
		metrics.EngagementConsistency = float64(profile.DaysActive) / (accountAge * 365.25)
		if metrics.EngagementConsistency > 1 {
			metrics.EngagementConsistency = 1
		}
	}

	// Calculate mentorship score
	metrics.MentorshipScore = c.calculateMentorshipScore(profile)

	// Calculate thought leadership
	metrics.ThoughtLeadership = c.calculateThoughtLeadership(profile)

	// Estimate people helped
	metrics.PeopleHelped = profile.SolutionsCount * 3 // Rough estimate: each solution helps 3 people on average

	// Calculate knowledge sharing ratio
	if profile.PostCount > 0 {
		// Ratio of providing answers vs asking questions (simplified)
		metrics.KnowledgeSharing = float64(profile.SolutionsCount) / float64(profile.PostCount) * 2
		if metrics.KnowledgeSharing > 1 {
			metrics.KnowledgeSharing = 1
		}
	}

	// Calculate community building score
	metrics.CommunityBuilding = c.calculateCommunityBuilding(profile)

	// Professional indicators
	metrics.TechnicalAuthority = c.calculateTechnicalAuthority(profile)
	metrics.ProblemSolvingSkill = float64(profile.SolutionsCount) / 10 // Normalize solutions
	if metrics.ProblemSolvingSkill > 1 {
		metrics.ProblemSolvingSkill = 1
	}

	metrics.CommunicationSkill = c.calculateCommunicationSkill(profile)

	return metrics
}

// Helper functions for metric calculations

func (c *Client) getCategoryName(categoryID int) string {
	// This function is deprecated - category names are now updated directly
	// in updateCategoryNames() function using the categories map
	return ""
}

func (c *Client) analyzeHelpfulness(content string, likes int, isSolution bool) float64 {
	score := 0.0

	// Base score from likes
	score += float64(likes) * 0.1

	// Solution bonus
	if isSolution {
		score += 2.0
	}

	// Content analysis (simplified)
	contentLower := strings.ToLower(content)

	// Helpful indicators
	helpfulWords := []string{"try", "solution", "fix", "install", "configure", "step", "guide", "tutorial"}
	for _, word := range helpfulWords {
		if strings.Contains(contentLower, word) {
			score += 0.2
		}
	}

	// Length bonus for detailed explanations
	if len(content) > 500 {
		score += 0.5
	}

	// Cap at 10
	if score > 10 {
		score = 10
	}

	return score
}

func (c *Client) analyzeTechnicalDepth(content string) float64 {
	score := 0.0
	contentLower := strings.ToLower(content)

	// Technical terms
	technicalTerms := []string{"jenkins", "pipeline", "plugin", "groovy", "dockerfile", "kubernetes", "yaml", "json", "api", "configuration", "deployment"}
	for _, term := range technicalTerms {
		if strings.Contains(contentLower, term) {
			score += 0.3
		}
	}

	// Code indicators
	if strings.Contains(content, "```") || strings.Contains(content, "`") {
		score += 1.0
	}

	// Cap at 10
	if score > 10 {
		score = 10
	}

	return score
}

func (c *Client) analyzeClarityScore(content string) float64 {
	score := 5.0 // Base score

	// Penalties for unclear writing
	if strings.Contains(strings.ToLower(content), "i think maybe") ||
	   strings.Contains(strings.ToLower(content), "not sure") {
		score -= 1.0
	}

	// Bonuses for clear writing
	if strings.Contains(content, "1.") || strings.Contains(content, "- ") {
		score += 1.0 // Lists are clear
	}

	// Length consideration
	if len(content) > 1000 {
		score -= 0.5 // Very long posts might be less clear
	}

	if score < 0 {
		score = 0
	}
	if score > 10 {
		score = 10
	}

	return score
}

func (c *Client) calculateEngagementScore(topic DiscourseTopic) float64 {
	score := 0.0

	// Views contribution
	score += float64(topic.Views) * 0.001

	// Likes contribution
	score += float64(topic.LikeCount) * 0.5

	// Reply count contribution
	score += float64(topic.ReplyCount) * 0.2

	// Recency bonus
	daysSinceCreated := time.Since(topic.CreatedAt).Hours() / 24
	if daysSinceCreated < 30 {
		score *= 1.2 // Recent topics get a bonus
	}

	return score
}

func (c *Client) calculateMentorshipScore(profile *DiscourseProfile) float64 {
	score := 0.0

	// Solution providing indicates mentorship
	score += float64(profile.SolutionsCount) * 0.3

	// High like ratio indicates helpful responses
	if profile.PostCount > 0 {
		likeRatio := float64(profile.LikesReceived) / float64(profile.PostCount)
		score += likeRatio * 0.2
	}

	// Trust level indicates community recognition
	score += float64(profile.TrustLevel) * 0.1

	if score > 1 {
		score = 1
	}

	return score
}

func (c *Client) calculateThoughtLeadership(profile *DiscourseProfile) float64 {
	score := 0.0

	// Topic creation indicates thought leadership
	score += float64(profile.TopicCount) * 0.1

	// High engagement topics
	for _, topic := range profile.TopTopics {
		if topic.EngagementScore > 5 {
			score += 0.2
		}
	}

	if score > 1 {
		score = 1
	}

	return score
}

func (c *Client) calculateCommunityBuilding(profile *DiscourseProfile) float64 {
	score := 0.0

	// Active participation
	if profile.DaysActive > 100 {
		score += 0.3
	}

	// Balanced giving and receiving likes
	if profile.LikesGiven > 0 && profile.LikesReceived > 0 {
		ratio := float64(profile.LikesGiven) / float64(profile.LikesReceived)
		if ratio > 0.5 && ratio < 2.0 {
			score += 0.3 // Balanced interaction
		}
	}

	// Trust level
	if profile.TrustLevel >= 3 {
		score += 0.4
	}

	if score > 1 {
		score = 1
	}

	return score
}

func (c *Client) calculateTechnicalAuthority(profile *DiscourseProfile) float64 {
	score := 0.0

	// Solutions provided
	score += float64(profile.SolutionsCount) * 0.2

	// Technical depth in posts
	for _, post := range profile.TopPosts {
		score += post.TechnicalDepth * 0.05
	}

	// Trust level and badges
	if profile.TrustLevel >= 4 {
		score += 0.3
	}

	if score > 1 {
		score = 1
	}

	return score
}

func (c *Client) calculateCommunicationSkill(profile *DiscourseProfile) float64 {
	score := 0.0

	// Average clarity score from posts
	if len(profile.TopPosts) > 0 {
		totalClarity := 0.0
		for _, post := range profile.TopPosts {
			totalClarity += post.ClarityScore
		}
		avgClarity := totalClarity / float64(len(profile.TopPosts))
		score += avgClarity / 10 // Normalize to 0-1
	}

	// Like ratio indicates good communication
	if profile.PostCount > 0 {
		likeRatio := float64(profile.LikesReceived) / float64(profile.PostCount)
		score += likeRatio * 0.2
	}

	if score > 1 {
		score = 1
	}

	return score
}

func (c *Client) identifyExpertiseAreas(profile *DiscourseProfile) []ExpertiseArea {
	var expertiseAreas []ExpertiseArea

	// Analyze categories for expertise
	for _, category := range profile.CategoryActivity {
		if category.ExpertiseLevel >= 3.0 { // Minimum threshold for expertise
			area := ExpertiseArea{
				Area:            category.CategoryName,
				PostCount:       category.PostCount,
				SolutionsCount:  category.SolutionsCount,
				ExpertiseScore:  category.ExpertiseLevel,
				RecognitionLevel: c.determineRecognitionLevel(category.ExpertiseLevel),
			}

			// Find first and last activity in this area
			var firstActivity, lastActivity time.Time
			var keyTopics []string
			var highImpactPosts []int

			for _, post := range profile.TopPosts {
				if post.CategoryName == category.CategoryName {
					if firstActivity.IsZero() || post.CreatedAt.Before(firstActivity) {
						firstActivity = post.CreatedAt
					}
					if lastActivity.IsZero() || post.CreatedAt.After(lastActivity) {
						lastActivity = post.CreatedAt
					}

					// Collect high-impact posts
					if post.HelpfulnessScore >= 5.0 {
						highImpactPosts = append(highImpactPosts, post.ID)
					}

					// Extract key topics from technical posts
					if post.TechnicalDepth >= 3.0 {
						keyTopics = append(keyTopics, post.TopicTitle)
					}
				}
			}

			area.FirstActivity = firstActivity
			area.LastActivity = lastActivity
			area.KeyTopics = keyTopics
			area.HighImpactPosts = highImpactPosts

			expertiseAreas = append(expertiseAreas, area)
		}
	}

	return expertiseAreas
}

func (c *Client) analyzeMentorshipSignals(profile *DiscourseProfile) MentorshipIndicators {
	indicators := MentorshipIndicators{}

	// Analyze posts for mentorship signals
	for _, post := range profile.TopPosts {
		content := strings.ToLower(post.Content)

		// New user help indicators
		if strings.Contains(content, "welcome") || strings.Contains(content, "new to jenkins") {
			indicators.NewUserHelp++
		}

		// Detailed explanations
		if len(post.Content) > 500 && post.TechnicalDepth > 3 {
			indicators.DetailedExplanations++
		}

		// Patience indicators
		if strings.Contains(content, "no problem") || strings.Contains(content, "happy to help") {
			indicators.PatienceIndicators++
		}
	}

	// Determine mentorship style based on post characteristics
	if indicators.DetailedExplanations > indicators.NewUserHelp {
		indicators.MentorshipStyle = "technical"
	} else if indicators.PatienceIndicators > 2 {
		indicators.MentorshipStyle = "encouraging"
	} else {
		indicators.MentorshipStyle = "comprehensive"
	}

	// Calculate teaching effectiveness
	if profile.PostCount > 0 {
		indicators.TeachingEffectiveness = float64(profile.SolutionsCount) / float64(profile.PostCount) * 2
		if indicators.TeachingEffectiveness > 1 {
			indicators.TeachingEffectiveness = 1
		}
	}

	// Community welcoming score
	indicators.CommunityWelcoming = float64(indicators.NewUserHelp) / 10
	if indicators.CommunityWelcoming > 1 {
		indicators.CommunityWelcoming = 1
	}

	return indicators
}

func (c *Client) determineRecognitionLevel(expertiseScore float64) string {
	if expertiseScore >= 8 {
		return "expert"
	} else if expertiseScore >= 6 {
		return "advanced"
	} else if expertiseScore >= 4 {
		return "intermediate"
	} else {
		return "beginner"
	}
}