package discourse

import "time"

// DiscourseProfile represents a user's Discourse community engagement and impact
type DiscourseProfile struct {
	Username            string                    `json:"username"`
	DisplayName         string                    `json:"display_name"`
	ProfileURL          string                    `json:"profile_url"`
	CommunityURL        string                    `json:"community_url"`
	JoinedDate          time.Time                 `json:"joined_date"`
	LastActivity        time.Time                 `json:"last_activity"`

	// Core engagement metrics
	PostCount           int                       `json:"post_count"`
	TopicCount          int                       `json:"topic_count"`
	LikesReceived       int                       `json:"likes_received"`
	LikesGiven          int                       `json:"likes_given"`
	SolutionsCount      int                       `json:"solutions_count"`
	DaysActive          int                       `json:"days_active"`
	ReadingTime         int                       `json:"reading_time_minutes"`

	// Trust and reputation
	TrustLevel          int                       `json:"trust_level"`
	BadgeCount          int                       `json:"badge_count"`
	Badges              []DiscourseBadge          `json:"badges"`

	// Community impact
	TopPosts            []DiscoursePost           `json:"top_posts"`
	TopTopics           []DiscourseTopic          `json:"top_topics"`
	CategoryActivity    []CategoryEngagement      `json:"category_activity"`

	// Leadership indicators
	CommunityMetrics    CommunityLeadershipMetrics `json:"community_metrics"`
	ExpertiseAreas      []ExpertiseArea           `json:"expertise_areas"`
	MentorshipSignals   MentorshipIndicators      `json:"mentorship_signals"`
}

// DiscoursePost represents a forum post
type DiscoursePost struct {
	ID              int       `json:"id"`
	TopicID         int       `json:"topic_id"`
	TopicTitle      string    `json:"topic_title"`
	PostNumber      int       `json:"post_number"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	LikeCount       int       `json:"like_count"`
	ReplyCount      int       `json:"reply_count"`
	IsSolution      bool      `json:"is_solution"`
	CategoryName    string    `json:"category_name"`
	Tags            []string  `json:"tags"`

	// Analysis metrics
	HelpfulnessScore float64   `json:"helpfulness_score"`
	TechnicalDepth   float64   `json:"technical_depth"`
	ClarityScore     float64   `json:"clarity_score"`
}

// DiscourseTopic represents a forum topic/thread
type DiscourseTopic struct {
	ID              int       `json:"id"`
	Title           string    `json:"title"`
	FancyTitle      string    `json:"fancy_title"`
	PostsCount      int       `json:"posts_count"`
	ReplyCount      int       `json:"reply_count"`
	CreatedAt       time.Time `json:"created_at"`
	LastPostedAt    time.Time `json:"last_posted_at"`
	Views           int       `json:"views"`
	LikeCount       int       `json:"like_count"`
	CategoryName    string    `json:"category_name"`
	Tags            []string  `json:"tags"`
	IsPinned        bool      `json:"is_pinned"`
	IsClosed        bool      `json:"is_closed"`

	// Impact metrics
	EngagementScore float64   `json:"engagement_score"`
	ImpactLevel     string    `json:"impact_level"` // low, medium, high, exceptional
}

// DiscourseBadge represents achievement badges
type DiscourseBadge struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	BadgeType   string    `json:"badge_type"` // bronze, silver, gold
	GrantedAt   time.Time `json:"granted_at"`
	Count       int       `json:"count"`

	// Categorization
	Category    string    `json:"category"` // participation, expertise, leadership, special
}

// CategoryEngagement represents activity within specific forum categories
type CategoryEngagement struct {
	CategoryID      int     `json:"category_id"`
	CategoryName    string  `json:"category_name"`
	PostCount       int     `json:"post_count"`
	TopicCount      int     `json:"topic_count"`
	LikesReceived   int     `json:"likes_received"`
	SolutionsCount  int     `json:"solutions_count"`

	// Engagement metrics
	ExpertiseLevel  float64 `json:"expertise_level"`  // 0-10 scale
	ActivityRank    int     `json:"activity_rank"`    // rank within category
	InfluenceScore  float64 `json:"influence_score"`  // impact within this area
}

// CommunityLeadershipMetrics represents leadership and influence indicators
type CommunityLeadershipMetrics struct {
	OverallRank           int     `json:"overall_rank"`           // site-wide ranking
	HelpfulnessRatio      float64 `json:"helpfulness_ratio"`      // solutions/posts ratio
	EngagementConsistency float64 `json:"engagement_consistency"` // regular activity score
	MentorshipScore       float64 `json:"mentorship_score"`       // helping others indicator
	ThoughtLeadership     float64 `json:"thought_leadership"`     // creating valuable discussions

	// Community impact
	PeopleHelped          int     `json:"people_helped"`          // estimated from solutions
	KnowledgeSharing      float64 `json:"knowledge_sharing"`      // teaching vs asking ratio
	CommunityBuilding     float64 `json:"community_building"`     // fostering discussions

	// Professional indicators
	TechnicalAuthority    float64 `json:"technical_authority"`    // expertise demonstration
	ProblemSolvingSkill   float64 `json:"problem_solving_skill"`  // solution quality
	CommunicationSkill    float64 `json:"communication_skill"`    // clarity and helpfulness
}

// ExpertiseArea represents areas of technical expertise demonstrated
type ExpertiseArea struct {
	Area            string   `json:"area"`            // e.g., "CI/CD", "Docker", "Jenkins Pipelines"
	PostCount       int      `json:"post_count"`      // posts in this area
	SolutionsCount  int      `json:"solutions_count"` // solutions provided
	ExpertiseScore  float64  `json:"expertise_score"` // 0-10 confidence level
	KeyTopics       []string `json:"key_topics"`      // specific topics discussed
	FirstActivity   time.Time `json:"first_activity"`  // when expertise first shown
	LastActivity    time.Time `json:"last_activity"`   // most recent activity

	// Evidence of expertise
	HighImpactPosts []int    `json:"high_impact_posts"` // post IDs with significant engagement
	RecognitionLevel string  `json:"recognition_level"` // beginner, intermediate, advanced, expert
}

// MentorshipIndicators represents signs of mentoring and helping others
type MentorshipIndicators struct {
	NewUserHelp         int     `json:"new_user_help"`         // responses to new users
	DetailedExplanations int     `json:"detailed_explanations"` // long, helpful posts
	FollowUpEngagement  int     `json:"follow_up_engagement"`  // continuing to help
	PatienceIndicators  int     `json:"patience_indicators"`   // kind responses to basic questions

	// Mentorship quality
	MentorshipStyle     string  `json:"mentorship_style"`      // encouraging, technical, comprehensive
	TeachingEffectiveness float64 `json:"teaching_effectiveness"` // success rate of help provided
	CommunityWelcoming  float64 `json:"community_welcoming"`   // making newcomers feel welcome
}

// API Response Types

// DiscourseUserResponse represents the API response for user information
type DiscourseUserResponse struct {
	User struct {
		ID                int       `json:"id"`
		Username          string    `json:"username"`
		Name              string    `json:"name"`
		AvatarTemplate    string    `json:"avatar_template"`
		CreatedAt         time.Time `json:"created_at"`
		LastPostedAt      time.Time `json:"last_posted_at"`
		LastSeenAt        time.Time `json:"last_seen_at"`
		PostCount         int       `json:"post_count"`
		TopicCount        int       `json:"topic_count"`
		LikesReceived     int       `json:"likes_received"`
		LikesGiven        int       `json:"likes_given"`
		SolutionCount     int       `json:"solution_count"`
		DaysVisited       int       `json:"days_visited"`
		ReadingTimeMinutes int      `json:"reading_time_minutes"`
		TrustLevel        int       `json:"trust_level"`
		Moderator         bool      `json:"moderator"`
		Admin             bool      `json:"admin"`
		BadgeCount        int       `json:"badge_count"`
	} `json:"user"`

	UserBadges []struct {
		ID          int    `json:"id"`
		GrantedAt   time.Time `json:"granted_at"`
		Count       int    `json:"count"`
		Badge       struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			BadgeTypeID int    `json:"badge_type_id"`
		} `json:"badge"`
	} `json:"user_badges"`
}

// DiscoursePostsResponse represents the API response for user posts
type DiscoursePostsResponse struct {
	LatestPosts []struct {
		ID            int       `json:"id"`
		Username      string    `json:"username"`
		CreatedAt     time.Time `json:"created_at"`
		Cooked        string    `json:"cooked"`
		PostNumber    int       `json:"post_number"`
		PostType      int       `json:"post_type"`
		TopicID       int       `json:"topic_id"`
		TopicTitle    string    `json:"topic_title"`
		CategoryID    int       `json:"category_id"`
		LikeCount     int       `json:"like_count"`
		ReplyCount    int       `json:"reply_count"`
		TopicViews    int       `json:"topic_views"`
		AcceptedAnswer bool     `json:"accepted_answer"`
	} `json:"latest_posts"`
}

// DiscourseTopicsResponse represents the API response for user topics
type DiscourseTopicsResponse struct {
	TopicList struct {
		Topics []struct {
			ID           int       `json:"id"`
			Title        string    `json:"title"`
			FancyTitle   string    `json:"fancy_title"`
			PostsCount   int       `json:"posts_count"`
			ReplyCount   int       `json:"reply_count"`
			CreatedAt    time.Time `json:"created_at"`
			LastPostedAt time.Time `json:"last_posted_at"`
			Views        int       `json:"views"`
			LikeCount    int       `json:"like_count"`
			CategoryID   int       `json:"category_id"`
			Tags         []string  `json:"tags"`
			Pinned       bool      `json:"pinned"`
			Closed       bool      `json:"closed"`
		} `json:"topics"`
	} `json:"topic_list"`
}

// DiscourseCategoriesResponse represents the API response for categories
type DiscourseCategoriesResponse struct {
	CategoryList struct {
		Categories []struct {
			ID                int    `json:"id"`
			Name              string `json:"name"`
			Color             string `json:"color"`
			TextColor         string `json:"text_color"`
			Slug              string `json:"slug"`
			TopicCount        int    `json:"topic_count"`
			PostCount         int    `json:"post_count"`
			Description       string `json:"description"`
			DescriptionText   string `json:"description_text"`
			TopicsYear        int    `json:"topics_year"`
			TopicsMonth       int    `json:"topics_month"`
			TopicsWeek        int    `json:"topics_week"`
			TopicsDay         int    `json:"topics_day"`
		} `json:"categories"`
	} `json:"category_list"`
}