package github

import "time"

// Response types for GitHub GraphQL API

// UserProfileResponse represents the response structure for user profile query
type UserProfileResponse struct {
	User struct {
		ID              string    `json:"id"`
		Login           string    `json:"login"`
		Name            string    `json:"name"`
		Bio             string    `json:"bio"`
		Company         string    `json:"company"`
		Location        string    `json:"location"`
		Email           string    `json:"email"`
		WebsiteUrl      string    `json:"websiteUrl"`
		TwitterUsername string    `json:"twitterUsername"`
		CreatedAt       time.Time `json:"createdAt"`
		UpdatedAt       time.Time `json:"updatedAt"`
		AvatarUrl       string    `json:"avatarUrl"`
		Followers       struct {
			TotalCount int `json:"totalCount"`
		} `json:"followers"`
		Following struct {
			TotalCount int `json:"totalCount"`
		} `json:"following"`
		Repositories struct {
			TotalCount int `json:"totalCount"`
		} `json:"repositories"`
		ContributionsCollection struct {
			TotalCommitContributions              int   `json:"totalCommitContributions"`
			TotalIssueContributions              int   `json:"totalIssueContributions"`
			TotalPullRequestContributions        int   `json:"totalPullRequestContributions"`
			TotalPullRequestReviewContributions  int   `json:"totalPullRequestReviewContributions"`
			ContributionYears                    []int `json:"contributionYears"`
		} `json:"contributionsCollection"`
		Organizations struct {
			Nodes []OrganizationNode `json:"nodes"`
		} `json:"organizations"`
	} `json:"user"`
}

// UserRepositoriesResponse represents the response for user repositories query
type UserRepositoriesResponse struct {
	User struct {
		Repositories struct {
			PageInfo PageInfo         `json:"pageInfo"`
			Nodes    []RepositoryNode `json:"nodes"`
		} `json:"repositories"`
	} `json:"user"`
}

// UserContributionsResponse represents the response for user contributions query
type UserContributionsResponse struct {
	User struct {
		ContributionsCollection struct {
			TotalCommitContributions             int `json:"totalCommitContributions"`
			TotalIssueContributions             int `json:"totalIssueContributions"`
			TotalPullRequestContributions       int `json:"totalPullRequestContributions"`
			TotalPullRequestReviewContributions int `json:"totalPullRequestReviewContributions"`
			ContributionCalendar                struct {
				TotalContributions int `json:"totalContributions"`
				Weeks              []struct {
					ContributionDays []struct {
						ContributionCount int    `json:"contributionCount"`
						Date              string `json:"date"`
					} `json:"contributionDays"`
				} `json:"weeks"`
			} `json:"contributionCalendar"`
			CommitContributionsByRepository []struct {
				Repository struct {
					NameWithOwner   string `json:"nameWithOwner"`
					PrimaryLanguage *struct {
						Name string `json:"name"`
					} `json:"primaryLanguage"`
					Owner struct {
						Login string `json:"login"`
					} `json:"owner"`
				} `json:"repository"`
				Contributions struct {
					Nodes []struct {
						CommitCount int `json:"commitCount"`
						User        struct {
							Login string `json:"login"`
						} `json:"user"`
					} `json:"nodes"`
				} `json:"contributions"`
			} `json:"commitContributionsByRepository"`
			IssueContributionsByRepository []struct {
				Repository struct {
					NameWithOwner string `json:"nameWithOwner"`
				} `json:"repository"`
				Contributions struct {
					Nodes []struct {
						IssueCount int `json:"issueCount"`
					} `json:"nodes"`
				} `json:"contributions"`
			} `json:"issueContributionsByRepository"`
			PullRequestContributionsByRepository []struct {
				Repository struct {
					NameWithOwner string `json:"nameWithOwner"`
					Owner         struct {
						Login string `json:"login"`
					} `json:"owner"`
				} `json:"repository"`
				Contributions struct {
					Nodes []struct {
						PullRequestCount int `json:"pullRequestCount"`
					} `json:"nodes"`
				} `json:"contributions"`
			} `json:"pullRequestContributionsByRepository"`
		} `json:"contributionsCollection"`
	} `json:"user"`
}

// UserOrganizationsResponse represents the response for user organizations query
type UserOrganizationsResponse struct {
	User struct {
		Organizations struct {
			Nodes []OrganizationNode `json:"nodes"`
		} `json:"organizations"`
		RepositoriesContributedTo struct {
			Nodes []RepositoryNode `json:"nodes"`
		} `json:"repositoriesContributedTo"`
	} `json:"user"`
}

// UserPullRequestsResponse represents the response for user pull requests query
type UserPullRequestsResponse struct {
	User struct {
		PullRequests struct {
			PageInfo PageInfo          `json:"pageInfo"`
			Nodes    []PullRequestNode `json:"nodes"`
		} `json:"pullRequests"`
	} `json:"user"`
}

// UserIssuesResponse represents the response for user issues query
type UserIssuesResponse struct {
	User struct {
		Issues struct {
			PageInfo PageInfo    `json:"pageInfo"`
			Nodes    []IssueNode `json:"nodes"`
		} `json:"issues"`
	} `json:"user"`
}

// SearchRepositoriesResponse represents the response for repository search query
type SearchRepositoriesResponse struct {
	Search struct {
		PageInfo PageInfo         `json:"pageInfo"`
		Nodes    []RepositoryNode `json:"nodes"`
	} `json:"search"`
}

// Common node types

// PageInfo represents pagination information
type PageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// OrganizationNode represents an organization in GraphQL responses
type OrganizationNode struct {
	Login       string    `json:"login"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Url         string    `json:"url"`
	AvatarUrl   string    `json:"avatarUrl"`
	CreatedAt   time.Time `json:"createdAt"`
	Repositories *struct {
		Nodes []RepositoryNode `json:"nodes"`
	} `json:"repositories,omitempty"`
}

// RepositoryNode represents a repository in GraphQL responses
type RepositoryNode struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	NameWithOwner string    `json:"nameWithOwner"`
	Description   string    `json:"description"`
	Url           string    `json:"url"`
	IsPrivate     bool      `json:"isPrivate"`
	IsFork        bool      `json:"isFork"`
	IsArchived    bool      `json:"isArchived"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	PushedAt      time.Time `json:"pushedAt"`
	StargazerCount int      `json:"stargazerCount"`
	ForkCount     int      `json:"forkCount"`
	Watchers      *struct {
		TotalCount int `json:"totalCount"`
	} `json:"watchers,omitempty"`
	Issues *struct {
		TotalCount int `json:"totalCount"`
	} `json:"issues,omitempty"`
	PullRequests *struct {
		TotalCount int `json:"totalCount"`
	} `json:"pullRequests,omitempty"`
	PrimaryLanguage *struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"primaryLanguage"`
	Languages *struct {
		TotalSize int `json:"totalSize,omitempty"`
		Nodes     []struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"nodes"`
		Edges []struct {
			Size int `json:"size"`
		} `json:"edges"`
	} `json:"languages,omitempty"`
	RepositoryTopics *struct {
		Nodes []struct {
			Topic struct {
				Name string `json:"name"`
			} `json:"topic"`
		} `json:"nodes"`
	} `json:"repositoryTopics,omitempty"`
	LicenseInfo *struct {
		Name   string `json:"name"`
		SpdxId string `json:"spdxId"`
	} `json:"licenseInfo"`
	DiskUsage int `json:"diskUsage"`
	Collaborators *struct {
		TotalCount int `json:"totalCount"`
		Nodes      []struct {
			Login string `json:"login"`
			Name  string `json:"name"`
		} `json:"nodes,omitempty"`
	} `json:"collaborators,omitempty"`
	Owner struct {
		Login       string `json:"login"`
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
	} `json:"owner"`
	Releases *struct {
		Nodes []struct {
			Name         string    `json:"name"`
			TagName      string    `json:"tagName"`
			CreatedAt    time.Time `json:"createdAt"`
			IsPrerelease bool      `json:"isPrerelease"`
		} `json:"nodes"`
	} `json:"releases,omitempty"`
}

// PullRequestNode represents a pull request in GraphQL responses
type PullRequestNode struct {
	ID           string    `json:"id"`
	Number       int       `json:"number"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	State        string    `json:"state"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	ClosedAt     *time.Time `json:"closedAt"`
	MergedAt     *time.Time `json:"mergedAt"`
	Additions    int       `json:"additions"`
	Deletions    int       `json:"deletions"`
	ChangedFiles int       `json:"changedFiles"`
	Repository   struct {
		NameWithOwner   string `json:"nameWithOwner"`
		Owner           struct {
			Login string `json:"login"`
		} `json:"owner"`
		PrimaryLanguage *struct {
			Name string `json:"name"`
		} `json:"primaryLanguage"`
	} `json:"repository"`
	Reviews *struct {
		TotalCount int `json:"totalCount"`
		Nodes      []struct {
			State       string    `json:"state"`
			SubmittedAt time.Time `json:"submittedAt"`
		} `json:"nodes"`
	} `json:"reviews,omitempty"`
	Comments *struct {
		TotalCount int `json:"totalCount"`
	} `json:"comments,omitempty"`
	Labels *struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels,omitempty"`
}

// IssueNode represents an issue in GraphQL responses
type IssueNode struct {
	ID        string     `json:"id"`
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	ClosedAt  *time.Time `json:"closedAt"`
	Repository struct {
		NameWithOwner   string `json:"nameWithOwner"`
		Owner           struct {
			Login string `json:"login"`
		} `json:"owner"`
		PrimaryLanguage *struct {
			Name string `json:"name"`
		} `json:"primaryLanguage"`
	} `json:"repository"`
	Comments *struct {
		TotalCount int `json:"totalCount"`
	} `json:"comments,omitempty"`
	Labels *struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels,omitempty"`
}