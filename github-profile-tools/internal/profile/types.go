package profile

import (
	"time"

	"github.com/jenkins/github-profile-tools/internal/discourse"
)

// UserProfile represents a comprehensive GitHub user profile analysis
type UserProfile struct {
	Username          string                 `json:"username"`
	Name              string                 `json:"name"`
	Bio               string                 `json:"bio"`
	Company           string                 `json:"company"`
	Location          string                 `json:"location"`
	Email             string                 `json:"email"`
	BlogURL           string                 `json:"blog_url"`
	TwitterUsername   string                 `json:"twitter_username"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	LastAnalyzed      time.Time              `json:"last_analyzed"`
	PublicRepos       int                    `json:"public_repos"`
	PublicGists       int                    `json:"public_gists"`
	Followers         int                    `json:"followers"`
	Following         int                    `json:"following"`

	Organizations     []OrganizationProfile  `json:"organizations"`
	Repositories      []RepositoryProfile    `json:"repositories"`
	Contributions     ContributionSummary    `json:"contributions"`
	Languages         []LanguageStats        `json:"languages"`
	Skills            SkillProfile           `json:"skills"`
	Collaborations    []CollaborationProfile `json:"collaborations"`
	Insights          UserInsights           `json:"insights"`
	DockerHubProfile  *DockerHubProfile      `json:"docker_hub_profile,omitempty"`
	DiscourseProfile  *DiscourseProfile      `json:"discourse_profile,omitempty"`
}

// OrganizationProfile represents user's involvement with organizations
type OrganizationProfile struct {
	Name              string    `json:"name"`
	Login             string    `json:"login"`
	Description       string    `json:"description"`
	URL               string    `json:"url"`
	AvatarURL         string    `json:"avatar_url"`
	FirstContribution time.Time `json:"first_contribution"`
	LastContribution  time.Time `json:"last_contribution"`
	ContributionCount int       `json:"contribution_count"`
	Repositories      []string  `json:"repositories"`
	Role              string    `json:"role"` // member, collaborator, contributor
	IsPublicMember    bool      `json:"is_public_member"`
}

// RepositoryProfile represents detailed repository analysis
type RepositoryProfile struct {
	Name              string            `json:"name"`
	FullName          string            `json:"full_name"`
	Description       string            `json:"description"`
	URL               string            `json:"url"`
	Language          string            `json:"primary_language"`
	Languages         map[string]int    `json:"languages"`
	IsPrivate         bool              `json:"is_private"`
	IsFork            bool              `json:"is_fork"`
	IsOwner           bool              `json:"is_owner"`
	IsArchived        bool              `json:"is_archived"`
	Stars             int               `json:"stargazers_count"`
	Forks             int               `json:"forks_count"`
	Watchers          int               `json:"watchers_count"`
	OpenIssues        int               `json:"open_issues_count"`
	Size              int               `json:"size"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	PushedAt          time.Time         `json:"pushed_at"`
	Topics            []string          `json:"topics"`
	License           string            `json:"license"`
	ContributionStats ContributionStats `json:"contribution_stats"`
	Organization      string            `json:"organization,omitempty"`
	CollaboratorCount int               `json:"collaborator_count"`
	DockerConfig      *DockerConfig     `json:"docker_config,omitempty"`
}

// ContributionStats represents user's contribution statistics to a repository
type ContributionStats struct {
	Commits           int       `json:"commits"`
	Additions         int       `json:"additions"`
	Deletions         int       `json:"deletions"`
	PullRequests      int       `json:"pull_requests"`
	Issues            int       `json:"issues"`
	CodeReviews       int       `json:"code_reviews"`
	FirstCommit       time.Time `json:"first_commit"`
	LastCommit        time.Time `json:"last_commit"`
	ImpactScore       float64   `json:"impact_score"`
}

// ContributionSummary represents overall contribution statistics
type ContributionSummary struct {
	TotalCommits            int                    `json:"total_commits"`
	TotalAdditions          int                    `json:"total_additions"`
	TotalDeletions          int                    `json:"total_deletions"`
	TotalPullRequests       int                    `json:"total_pull_requests"`
	TotalIssues             int                    `json:"total_issues"`
	TotalCodeReviews        int                    `json:"total_code_reviews"`
	ContributionYears       int                    `json:"contribution_years"`
	MostActiveYear          int                    `json:"most_active_year"`
	MostActiveMonth         string                 `json:"most_active_month"`
	ConsistencyScore        float64                `json:"consistency_score"`
	YearlyContributions     map[string]int         `json:"yearly_contributions"`
	MonthlyContributions    map[string]int         `json:"monthly_contributions"`
	WeeklyPattern           []int                  `json:"weekly_pattern"` // Sunday = 0
	ContributionStreak      int                    `json:"current_streak"`
	LongestStreak           int                    `json:"longest_streak"`
}

// LanguageStats represents programming language statistics
type LanguageStats struct {
	Language       string  `json:"language"`
	Bytes          int     `json:"bytes"`
	Percentage     float64 `json:"percentage"`
	RepositoryCount int    `json:"repository_count"`
	CommitCount    int     `json:"commit_count"`
	LinesOfCode    int     `json:"lines_of_code"`
	ProjectCount   int     `json:"project_count"`
	FirstUsed      time.Time `json:"first_used"`
	LastUsed       time.Time `json:"last_used"`
	ProficiencyScore float64 `json:"proficiency_score"`
}

// SkillProfile represents inferred technical skills
type SkillProfile struct {
	PrimaryLanguages   []string          `json:"primary_languages"`
	SecondaryLanguages []string          `json:"secondary_languages"`
	Frameworks         []TechnologySkill `json:"frameworks"`
	Databases          []TechnologySkill `json:"databases"`
	Tools              []TechnologySkill `json:"tools"`
	CloudPlatforms     []TechnologySkill `json:"cloud_platforms"`
	DevOpsSkills       []TechnologySkill `json:"devops_skills"`
	TechnicalAreas     []TechnicalArea   `json:"technical_areas"`
}

// TechnologySkill represents proficiency in a specific technology
type TechnologySkill struct {
	Name           string    `json:"name"`
	Confidence     float64   `json:"confidence"`     // 0-1 confidence score
	Evidence       []string  `json:"evidence"`       // repos, commits, etc.
	FirstUsed      time.Time `json:"first_used"`
	LastUsed       time.Time `json:"last_used"`
	ProjectCount   int       `json:"project_count"`
	ProficiencyLevel string  `json:"proficiency_level"` // beginner, intermediate, advanced, expert
}

// TechnicalArea represents broader technical competencies
type TechnicalArea struct {
	Area           string   `json:"area"`           // e.g., "Web Development", "Machine Learning"
	Competency     float64  `json:"competency"`     // 0-1 score
	Technologies   []string `json:"technologies"`   // supporting technologies
	ProjectCount   int      `json:"project_count"`
	YearsActive    float64  `json:"years_active"`
}

// CollaborationProfile represents collaboration patterns
type CollaborationProfile struct {
	Repository        string    `json:"repository"`
	Collaborators     []string  `json:"collaborators"`
	CollaborationType string    `json:"collaboration_type"` // contributor, maintainer, reviewer
	Duration          string    `json:"duration"`
	ImpactLevel       string    `json:"impact_level"` // low, medium, high
	StartDate         time.Time `json:"start_date"`
	EndDate           time.Time `json:"end_date"`
}

// UserInsights represents AI-generated insights about the user
type UserInsights struct {
	CareerLevel         string                 `json:"career_level"` // junior, mid, senior, lead, principal
	TechnicalFocus      []string               `json:"technical_focus"`
	LeadershipIndicators []LeadershipIndicator `json:"leadership_indicators"`
	MentorshipSigns     []string               `json:"mentorship_signs"`
	InnovationMetrics   InnovationMetrics      `json:"innovation_metrics"`
	CommunityImpact     CommunityMetrics       `json:"community_impact"`
	ArchitecturalThinking ArchitectureSignals  `json:"architectural_thinking"`
	OverallImpactScore  float64                `json:"overall_impact_score"`
	CareerTrajectory    string                 `json:"career_trajectory"` // ascending, stable, diverse
	RecommendedRoles    []string               `json:"recommended_roles"`
	StrengthAreas       []string               `json:"strength_areas"`
	GrowthAreas         []string               `json:"growth_areas"`
}

// LeadershipIndicator represents signs of technical leadership
type LeadershipIndicator struct {
	Type        string   `json:"type"`        // e.g., "project_ownership", "mentoring", "decision_making"
	Evidence    []string `json:"evidence"`    // specific examples
	Strength    float64  `json:"strength"`    // 0-1 confidence score
	Description string   `json:"description"`
}

// InnovationMetrics represents innovation and creativity indicators
type InnovationMetrics struct {
	OriginalProjects    int     `json:"original_projects"`
	ExperimentalRepos   int     `json:"experimental_repos"`
	TechnologyAdoption  float64 `json:"technology_adoption"`   // early vs late adopter score
	CreativityScore     float64 `json:"creativity_score"`
	ProblemSolvingScore float64 `json:"problem_solving_score"`
}

// CommunityMetrics represents open source and community engagement
type CommunityMetrics struct {
	OpenSourceProjects     int     `json:"open_source_projects"`
	CommunityContributions int     `json:"community_contributions"`
	IssueResolutionRate    float64 `json:"issue_resolution_rate"`
	HelpfulnessScore       float64 `json:"helpfulness_score"`
	DocumentationContrib   int     `json:"documentation_contributions"`
}

// ArchitectureSignals represents architectural thinking patterns
type ArchitectureSignals struct {
	SystemDesignProjects []string `json:"system_design_projects"`
	ArchitecturalPatterns []string `json:"architectural_patterns"`
	ScalabilityFocus     bool     `json:"scalability_focus"`
	PerformanceOptimization bool  `json:"performance_optimization"`
	SecurityMindedness   bool     `json:"security_mindedness"`
	ComplexityScore      float64  `json:"complexity_score"`
}

// ProfileSnapshot represents versioned profile data for incremental updates
type ProfileSnapshot struct {
	Username      string    `json:"username"`
	Timestamp     time.Time `json:"timestamp"`
	Version       string    `json:"version"`
	LastEventID   string    `json:"last_event_id"`
	LastCommitSHA string    `json:"last_commit_sha"`
	Checkpoints   []Checkpoint `json:"checkpoints"`
}

// Checkpoint represents incremental update tracking
type Checkpoint struct {
	DataType      string    `json:"data_type"` // repos, orgs, events, etc.
	LastUpdated   time.Time `json:"last_updated"`
	LastCursor    string    `json:"last_cursor,omitempty"`
	RecordCount   int       `json:"record_count"`
	Hash          string    `json:"hash"` // data integrity check
}

// DockerHubProfile represents Docker Hub activity and impact (simplified for profile integration)
type DockerHubProfile struct {
	Username          string    `json:"username"`
	TotalDownloads    int64     `json:"total_downloads"`
	TotalImages       int       `json:"total_images"`
	TopRepositories   []string  `json:"top_repositories"`
	MostDownloadedImage string  `json:"most_downloaded_image"`
	CommunityImpact   float64   `json:"community_impact"`      // 0-10 scale
	ExperienceYears   float64   `json:"experience_years"`
	ProficiencyLevel  string    `json:"proficiency_level"`     // beginner, intermediate, advanced, expert
	LastActivity      time.Time `json:"last_activity"`
}

// DiscourseProfile represents Discourse community engagement (simplified for profile integration)
type DiscourseProfile struct {
	Username         string                                       `json:"username"`
	DisplayName      string                                       `json:"display_name"`
	ProfileURL       string                                       `json:"profile_url"`
	CommunityURL     string                                       `json:"community_url"`
	JoinedDate       time.Time                                    `json:"joined_date"`
	LastActivity     time.Time                                    `json:"last_activity"`
	PostCount        int                                          `json:"post_count"`
	TopicCount       int                                          `json:"topic_count"`
	LikesReceived    int                                          `json:"likes_received"`
	LikesGiven       int                                          `json:"likes_given"`
	SolutionsCount   int                                          `json:"solutions_count"`
	DaysActive       int                                          `json:"days_active"`
	ReadingTime      int                                          `json:"reading_time"`
	TrustLevel       int                                          `json:"trust_level"`
	BadgeCount       int                                          `json:"badge_count"`
	CommunityMetrics discourse.CommunityLeadershipMetrics        `json:"community_metrics"`
	ExpertiseAreas   []discourse.ExpertiseArea                   `json:"expertise_areas"`
	MentorshipSignals discourse.MentorshipIndicators             `json:"mentorship_signals"`
	CategoryActivity []discourse.CategoryEngagement              `json:"category_activity"`
}

// DockerConfig represents Docker-related configuration and complexity in a repository
type DockerConfig struct {
	HasDockerfile       bool                `json:"has_dockerfile"`
	HasCompose          bool                `json:"has_compose"`
	HasBakeFile         bool                `json:"has_bake_file"`
	HasDockerIgnore     bool                `json:"has_docker_ignore"`
	DockerFiles         []DockerFile        `json:"docker_files"`
	ComposeFiles        []string            `json:"compose_files"`
	BakeFiles           []string            `json:"bake_files"`
	ComplexityScore     float64             `json:"complexity_score"`     // 0-10 scale
	DockerPatterns      []string            `json:"docker_patterns"`      // detected patterns
	ContainerExpertise  DockerExpertiseLevel `json:"container_expertise"`
}

// DockerFile represents information about a specific Dockerfile
type DockerFile struct {
	Path                string   `json:"path"`
	BaseImage           string   `json:"base_image"`
	IsMultiStage        bool     `json:"is_multi_stage"`
	StageCount          int      `json:"stage_count"`
	Instructions        []string `json:"instructions"`          // RUN, COPY, etc.
	BestPractices       []string `json:"best_practices"`        // detected best practices
	SecurityPatterns    []string `json:"security_patterns"`     // security-related patterns
	OptimizationLevel   string   `json:"optimization_level"`    // basic, intermediate, advanced
}

// DockerExpertiseLevel represents the level of Docker expertise demonstrated in a repository
type DockerExpertiseLevel struct {
	Level               string   `json:"level"`                // beginner, intermediate, advanced, expert
	Evidence            []string `json:"evidence"`             // specific evidence of expertise
	TechnologiesUsed    []string `json:"technologies_used"`    // docker, compose, swarm, kubernetes, etc.
	AdvancedPatterns    []string `json:"advanced_patterns"`    // multi-stage, distroless, init containers, etc.
	ProductionReadiness bool     `json:"production_readiness"` // indicates production-level Docker usage
}