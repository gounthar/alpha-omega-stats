package docker

import "time"

// DockerHubProfile represents a user's Docker Hub activity and impact
type DockerHubProfile struct {
	Username          string                `json:"username"`
	FullName          string                `json:"full_name"`
	ProfileURL        string                `json:"profile_url"`
	JoinedDate        time.Time             `json:"joined_date"`
	LastActivity      time.Time             `json:"last_activity"`

	Repositories      []DockerRepository    `json:"repositories"`
	TotalDownloads    int64                 `json:"total_downloads"`
	TotalImages       int                   `json:"total_images"`
	PublicRepos       int                   `json:"public_repos"`

	ImpactMetrics     DockerImpactMetrics   `json:"impact_metrics"`
	ContainerExpertise ContainerExpertise   `json:"container_expertise"`
}

// DockerRepository represents a Docker Hub repository
type DockerRepository struct {
	Name              string    `json:"name"`
	Namespace         string    `json:"namespace"`
	FullName          string    `json:"full_name"`
	Description       string    `json:"description"`
	ShortDescription  string    `json:"short_description"`
	IsOfficial        bool      `json:"is_official"`
	IsAutomated       bool      `json:"is_automated"`
	CanEdit           bool      `json:"can_edit"`
	StarCount         int       `json:"star_count"`
	PullCount         int64     `json:"pull_count"`
	LastUpdated       time.Time `json:"last_updated"`
	DateRegistered    time.Time `json:"date_registered"`

	// Additional metrics
	Tags              []DockerTag           `json:"tags"`
	Architecture      []string              `json:"architectures"`
	Categories        []string              `json:"categories"`

	// Calculated fields
	PopularityScore   float64               `json:"popularity_score"`
	MaintenanceScore  float64               `json:"maintenance_score"`
	ImpactLevel       string                `json:"impact_level"` // low, medium, high, massive
}

// DockerTag represents a specific tag/version of a Docker image
type DockerTag struct {
	Name              string    `json:"name"`
	FullSize          int64     `json:"full_size"`
	ImageID           string    `json:"image_id"`
	V2                bool      `json:"v2"`
	LastUpdated       time.Time `json:"last_updated"`
	LastPushed        time.Time `json:"last_pushed"`
	Architecture      string    `json:"architecture"`
}

// DockerImpactMetrics represents the professional impact of Docker work
type DockerImpactMetrics struct {
	TotalDownloads          int64   `json:"total_downloads"`
	MonthlyDownloads        int64   `json:"monthly_downloads"`
	DownloadGrowthRate      float64 `json:"download_growth_rate"`

	PopularityRank          int     `json:"popularity_rank"`
	CommunityImpact         float64 `json:"community_impact"`      // 0-10 scale
	InfrastructureInfluence float64 `json:"infrastructure_influence"` // 0-10 scale

	TopRepositories         []string `json:"top_repositories"`
	MostDownloadedImage     string   `json:"most_downloaded_image"`
	TotalImageVariants      int      `json:"total_image_variants"`

	// Professional indicators
	EnterpriseAdoption      bool     `json:"enterprise_adoption"`
	ContinuousDeployment    bool     `json:"continuous_deployment"`
	MultiArchSupport        bool     `json:"multi_arch_support"`
	SecurityScanning        bool     `json:"security_scanning"`
}

// ContainerExpertise represents inferred container technology expertise
type ContainerExpertise struct {
	ExperienceYears         float64              `json:"experience_years"`
	ProficiencyLevel        string               `json:"proficiency_level"` // beginner, intermediate, advanced, expert

	Technologies            []ContainerTechnology `json:"technologies"`
	Orchestration          []string              `json:"orchestration"`     // kubernetes, docker-swarm, etc.
	BaseImages             []string              `json:"base_images"`       // alpine, ubuntu, centos, etc.

	// Specialization areas
	Microservices          bool                  `json:"microservices"`
	DevOpsIntegration      bool                  `json:"devops_integration"`
	ProductionOptimization bool                  `json:"production_optimization"`
	SecurityHardening      bool                  `json:"security_hardening"`

	// Advanced patterns
	MultiStageBuilds       bool                  `json:"multi_stage_builds"`
	DistrolessImages       bool                  `json:"distroless_images"`
	InitContainers         bool                  `json:"init_containers"`
	SidecarPatterns        bool                  `json:"sidecar_patterns"`
}

// ContainerTechnology represents expertise in container-related technologies
type ContainerTechnology struct {
	Name               string    `json:"name"`
	Category           string    `json:"category"` // runtime, orchestration, registry, security, etc.
	ProficiencyScore   float64   `json:"proficiency_score"`
	Evidence           []string  `json:"evidence"`
	FirstUsed          time.Time `json:"first_used"`
	LastUsed           time.Time `json:"last_used"`
	ProjectCount       int       `json:"project_count"`
}

// DockerHubSearchResponse represents Docker Hub API search results
type DockerHubSearchResponse struct {
	Count    int                    `json:"count"`
	Next     string                 `json:"next"`
	Previous string                 `json:"previous"`
	Results  []DockerSearchResult   `json:"results"`
}

// DockerSearchResult represents a single search result from Docker Hub
type DockerSearchResult struct {
	RepoName         string    `json:"repo_name"`
	ShortDescription string    `json:"short_description"`
	StarCount        int       `json:"star_count"`
	PullCount        int64     `json:"pull_count"`
	RepoOwner        string    `json:"repo_owner"`
	IsAutomated      bool      `json:"is_automated"`
	IsOfficial       bool      `json:"is_official"`
	LastUpdated      time.Time `json:"last_updated"`
}

// DockerHubRepositoryResponse represents detailed repository info from Docker Hub API
type DockerHubRepositoryResponse struct {
	User              string    `json:"user"`
	Name              string    `json:"name"`
	Namespace         string    `json:"namespace"`
	RepositoryType    string    `json:"repository_type"`
	Status            int       `json:"status"`
	Description       string    `json:"description"`
	IsPrivate         bool      `json:"is_private"`
	IsAutomated       bool      `json:"is_automated"`
	CanEdit           bool      `json:"can_edit"`
	StarCount         int       `json:"star_count"`
	PullCount         int64     `json:"pull_count"`
	LastUpdated       time.Time `json:"last_updated"`
	DateRegistered    time.Time `json:"date_registered"`
	Collaborators     []string  `json:"collaborators,omitempty"`
	Affiliation       string    `json:"affiliation,omitempty"`
}