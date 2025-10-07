package markdown

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jenkins/github-profile-tools/internal/profile"
)

// Generator handles markdown profile generation
type Generator struct{}

// NewGenerator creates a new markdown generator
func NewGenerator() *Generator {
	return &Generator{}
}

// TemplateType represents different markdown template types
type TemplateType string

const (
	ResumeTemplate    TemplateType = "resume"
	TechnicalTemplate TemplateType = "technical"
	ExecutiveTemplate TemplateType = "executive"
	ATSTemplate       TemplateType = "ats"
)

// GenerateMarkdown generates markdown profile based on template type
func (g *Generator) GenerateMarkdown(prof *profile.UserProfile, templateType TemplateType) (string, error) {
	switch templateType {
	case ResumeTemplate:
		return g.generateResumeTemplate(prof), nil
	case TechnicalTemplate:
		return g.generateTechnicalTemplate(prof), nil
	case ExecutiveTemplate:
		return g.generateExecutiveTemplate(prof), nil
	case ATSTemplate:
		return g.generateATSTemplate(prof), nil
	default:
		return "", fmt.Errorf("unknown template type: %s", templateType)
	}
}

// generateResumeTemplate creates a resume-focused markdown profile
func (g *Generator) generateResumeTemplate(prof *profile.UserProfile) string {
	var md strings.Builder

	// Header
	md.WriteString(fmt.Sprintf("# GitHub Professional Profile - %s\n\n", prof.Username))

	if prof.Name != "" {
		md.WriteString(fmt.Sprintf("**Name:** %s\n", prof.Name))
	}
	if prof.Location != "" {
		md.WriteString(fmt.Sprintf("**Location:** %s\n", prof.Location))
	}
	if prof.Company != "" {
		md.WriteString(fmt.Sprintf("**Company:** %s\n", prof.Company))
	}
	if prof.BlogURL != "" {
		md.WriteString(fmt.Sprintf("**Website:** %s\n", prof.BlogURL))
	}
	md.WriteString("\n")

	if prof.Bio != "" {
		md.WriteString(fmt.Sprintf("*%s*\n\n", prof.Bio))
	}

	// Contribution Overview
	md.WriteString("## 📊 Contribution Overview\n\n")
	md.WriteString(fmt.Sprintf("- **%d** total contributions across **%.0f** years of active development\n",
		prof.Contributions.TotalCommits+prof.Contributions.TotalPullRequests+prof.Contributions.TotalIssues,
		float64(prof.Contributions.ContributionYears)))
	md.WriteString(fmt.Sprintf("- **%d** repositories with **%d** stars received\n",
		len(prof.Repositories), g.getTotalStars(prof)))

	// Add Docker Hub metrics if available
	if prof.DockerHubProfile != nil {
		md.WriteString(fmt.Sprintf("- **%s** Docker Hub downloads across **%d** container images 🐳\n",
			g.formatLargeNumber(prof.DockerHubProfile.TotalDownloads), prof.DockerHubProfile.TotalImages))
	}

	// Add Discourse community engagement if available
	if prof.DiscourseProfile != nil {
		md.WriteString(fmt.Sprintf("- **%d** community posts with **%d** solutions provided in Jenkins forums 💬\n",
			prof.DiscourseProfile.PostCount, prof.DiscourseProfile.SolutionsCount))
	}

	md.WriteString(fmt.Sprintf("- Active contributor in **%d** organizations\n", len(prof.Organizations)))
	md.WriteString(fmt.Sprintf("- Proficient in **%d** programming languages\n", len(prof.Languages)))
	md.WriteString(fmt.Sprintf("- Career Level: **%s**\n", strings.Title(prof.Insights.CareerLevel)))
	md.WriteString("\n")

	// Organization Contributions
	if len(prof.Organizations) > 0 {
		md.WriteString("## 🏢 Organization Contributions\n\n")

		// Sort organizations by contribution count
		orgs := make([]profile.OrganizationProfile, len(prof.Organizations))
		copy(orgs, prof.Organizations)
		sort.Slice(orgs, func(i, j int) bool {
			return orgs[i].ContributionCount > orgs[j].ContributionCount
		})

		for _, org := range orgs {
			if org.ContributionCount == 0 {
				continue
			}

			md.WriteString(fmt.Sprintf("### %s\n", org.Name))
			if org.Description != "" {
				md.WriteString(fmt.Sprintf("*%s*\n\n", org.Description))
			}
			md.WriteString(fmt.Sprintf("- **Role:** %s\n", strings.Title(org.Role)))
			md.WriteString(fmt.Sprintf("- **Contributions:** %d repositories\n", org.ContributionCount))

			if len(org.Repositories) > 0 {
				md.WriteString("- **Key Projects:** ")
				topRepos := org.Repositories
				if len(topRepos) > 3 {
					topRepos = topRepos[:3]
				}
				md.WriteString(strings.Join(topRepos, ", "))
				if len(org.Repositories) > 3 {
					md.WriteString(fmt.Sprintf(" and %d more", len(org.Repositories)-3))
				}
				md.WriteString("\n")
			}
			md.WriteString("\n")
		}
	}

	// Docker Hub Impact Section (if significant)
	if prof.DockerHubProfile != nil && prof.DockerHubProfile.TotalDownloads > 100000 {
		md.WriteString("## 🐳 Container Infrastructure Impact\n\n")

		md.WriteString(fmt.Sprintf("### Docker Hub Profile: [@%s](https://hub.docker.com/u/%s)\n\n",
			prof.DockerHubProfile.Username, prof.DockerHubProfile.Username))

		md.WriteString(fmt.Sprintf("- **Total Downloads**: %s across all images\n",
			g.formatLargeNumber(prof.DockerHubProfile.TotalDownloads)))
		md.WriteString(fmt.Sprintf("- **Container Images**: %d published images\n", prof.DockerHubProfile.TotalImages))
		md.WriteString(fmt.Sprintf("- **Community Impact**: %.1f/10 (Infrastructure influence)\n", prof.DockerHubProfile.CommunityImpact))
		md.WriteString(fmt.Sprintf("- **Container Expertise**: %s level (%.1f years experience)\n",
			strings.Title(prof.DockerHubProfile.ProficiencyLevel), prof.DockerHubProfile.ExperienceYears))

		if prof.DockerHubProfile.MostDownloadedImage != "" {
			md.WriteString(fmt.Sprintf("- **Most Popular Image**: `%s`\n", prof.DockerHubProfile.MostDownloadedImage))
		}

		if len(prof.DockerHubProfile.TopRepositories) > 0 {
			md.WriteString("- **Key Container Projects**: ")
			topRepos := prof.DockerHubProfile.TopRepositories
			if len(topRepos) > 3 {
				topRepos = topRepos[:3]
			}
			for i, repo := range topRepos {
				if i > 0 {
					md.WriteString(", ")
				}
				md.WriteString(fmt.Sprintf("`%s`", repo))
			}
			md.WriteString("\n")
		}

		md.WriteString("\n**Infrastructure Impact**: This level of container adoption demonstrates significant influence on development workflows and production deployments across the software community.\n\n")
	}

	// Discourse Community Engagement Section (if active)
	if prof.DiscourseProfile != nil && prof.DiscourseProfile.PostCount > 50 {
		md.WriteString("## 💬 Jenkins Community Leadership\n\n")

		md.WriteString(fmt.Sprintf("### Community Profile: [@%s](%s)\n\n",
			prof.DiscourseProfile.Username, prof.DiscourseProfile.ProfileURL))

		md.WriteString(fmt.Sprintf("- **Community Tenure**: %.1f years active (joined %s)\n",
			time.Since(prof.DiscourseProfile.JoinedDate).Hours()/(24*365.25), prof.DiscourseProfile.JoinedDate.Format("Jan 2006")))
		md.WriteString(fmt.Sprintf("- **Engagement**: %d posts, %d topics created\n",
			prof.DiscourseProfile.PostCount, prof.DiscourseProfile.TopicCount))
		md.WriteString(fmt.Sprintf("- **Community Impact**: %d solutions provided, %d likes received\n",
			prof.DiscourseProfile.SolutionsCount, prof.DiscourseProfile.LikesReceived))
		md.WriteString(fmt.Sprintf("- **Trust Level**: %d/4 (Community recognition)\n", prof.DiscourseProfile.TrustLevel))

		if prof.DiscourseProfile.BadgeCount > 0 {
			md.WriteString(fmt.Sprintf("- **Achievements**: %d community badges earned\n", prof.DiscourseProfile.BadgeCount))
		}

		// Community metrics
		if prof.DiscourseProfile.CommunityMetrics.HelpfulnessRatio > 0 {
			md.WriteString(fmt.Sprintf("- **Mentorship Score**: %.1f/10 (Helping others indicator)\n",
				prof.DiscourseProfile.CommunityMetrics.MentorshipScore*10))
		}

		if prof.DiscourseProfile.CommunityMetrics.PeopleHelped > 0 {
			md.WriteString(fmt.Sprintf("- **Estimated People Helped**: %d+ community members\n",
				prof.DiscourseProfile.CommunityMetrics.PeopleHelped))
		}

		// Expertise areas
		if len(prof.DiscourseProfile.ExpertiseAreas) > 0 {
			md.WriteString("\n**Areas of Expertise in Jenkins Community**:\n")
			for i, area := range prof.DiscourseProfile.ExpertiseAreas {
				if i >= 3 { // Limit to top 3
					break
				}
				md.WriteString(fmt.Sprintf("- **%s**: %s level (%.1f/10 expertise score)\n",
					area.Area, strings.Title(area.RecognitionLevel), area.ExpertiseScore))
			}
		}

		md.WriteString("\n**Community Leadership**: Active Jenkins community member providing technical guidance and solutions to fellow developers and DevOps practitioners.\n\n")
	}

	// Notable Projects
	md.WriteString("## 💼 Notable Projects\n\n")
	notableRepos := g.getNotableRepositories(prof)

	for _, repo := range notableRepos {
		md.WriteString(fmt.Sprintf("### [%s](%s)", repo.Name, repo.URL))
		if repo.Stars > 0 {
			md.WriteString(fmt.Sprintf(" ⭐ %d", repo.Stars))
		}
		md.WriteString("\n")

		if repo.Description != "" {
			md.WriteString(fmt.Sprintf("**Description:** %s\n\n", repo.Description))
		}

		md.WriteString(fmt.Sprintf("- **Language:** %s", repo.Language))
		if repo.Size > 0 {
			md.WriteString(fmt.Sprintf(" | **Size:** %.1f MB", float64(repo.Size)/1024))
		}
		md.WriteString("\n")

		if len(repo.Topics) > 0 {
			md.WriteString(fmt.Sprintf("- **Technologies:** %s\n", strings.Join(repo.Topics, ", ")))
		}

		if repo.ContributionStats.Commits > 0 {
			md.WriteString(fmt.Sprintf("- **Contributions:** %d commits", repo.ContributionStats.Commits))
			if repo.ContributionStats.Additions > 0 {
				md.WriteString(fmt.Sprintf(" (+%d/-%d lines)",
					repo.ContributionStats.Additions, repo.ContributionStats.Deletions))
			}
			md.WriteString("\n")
		}
		md.WriteString("\n")
	}

	// Technical Skills
	md.WriteString("## 🛠 Technical Skills\n\n")

	if len(prof.Skills.PrimaryLanguages) > 0 {
		md.WriteString("### Programming Languages\n")
		for i, lang := range prof.Languages {
			if i >= 8 { // Limit to top 8 languages
				break
			}

			profLevel := "Intermediate"
			if lang.Percentage > 25 {
				profLevel = "Advanced"
			} else if lang.Percentage > 40 {
				profLevel = "Expert"
			} else if lang.Percentage < 5 {
				profLevel = "Beginner"
			}

			md.WriteString(fmt.Sprintf("- **%s:** %s (%.1f%% of codebase, %d projects)\n",
				lang.Language, profLevel, lang.Percentage, lang.RepositoryCount))
		}
		md.WriteString("\n")
	}

	// Technology Stack
	if len(prof.Skills.Frameworks) > 0 || len(prof.Skills.Databases) > 0 || len(prof.Skills.CloudPlatforms) > 0 {
		md.WriteString("### Technology Stack\n")

		if len(prof.Skills.Frameworks) > 0 {
			frameworks := g.getTechnologyNames(prof.Skills.Frameworks, 5)
			md.WriteString(fmt.Sprintf("- **Frameworks:** %s\n", strings.Join(frameworks, ", ")))
		}

		if len(prof.Skills.Databases) > 0 {
			databases := g.getTechnologyNames(prof.Skills.Databases, 5)
			md.WriteString(fmt.Sprintf("- **Databases:** %s\n", strings.Join(databases, ", ")))
		}

		if len(prof.Skills.CloudPlatforms) > 0 {
			cloud := g.getTechnologyNames(prof.Skills.CloudPlatforms, 5)
			md.WriteString(fmt.Sprintf("- **Cloud Platforms:** %s\n", strings.Join(cloud, ", ")))
		}

		if len(prof.Skills.DevOpsSkills) > 0 {
			devops := g.getTechnologyNames(prof.Skills.DevOpsSkills, 5)
			md.WriteString(fmt.Sprintf("- **DevOps & Tools:** %s\n", strings.Join(devops, ", ")))
		}
		md.WriteString("\n")
	}

	// Professional Insights
	md.WriteString("## 🤝 Professional Insights\n\n")

	totalOSContributions := g.countOpenSourceContributions(prof)
	md.WriteString(fmt.Sprintf("- **Open Source Contributions:** %d repositories\n", totalOSContributions))

	if len(prof.Organizations) > 1 {
		md.WriteString(fmt.Sprintf("- **Cross-Organization Work:** Contributed to %d different organizations\n", len(prof.Organizations)))
	}

	if len(prof.Insights.LeadershipIndicators) > 0 {
		md.WriteString("- **Leadership Experience:** ")
		var indicators []string
		for _, indicator := range prof.Insights.LeadershipIndicators {
			indicators = append(indicators, indicator.Description)
		}
		md.WriteString(strings.Join(indicators, "; "))
		md.WriteString("\n")
	}

	md.WriteString(fmt.Sprintf("- **Overall Impact Score:** %.1f/10\n", prof.Insights.OverallImpactScore*10))
	md.WriteString("\n")

	// Activity Timeline
	md.WriteString("## 📈 Activity Timeline\n\n")

	if prof.Contributions.MostActiveYear > 0 {
		md.WriteString(fmt.Sprintf("- **Most Active Period:** %d\n", prof.Contributions.MostActiveYear))
	}

	md.WriteString(fmt.Sprintf("- **Consistency Score:** %.1f/10\n", prof.Contributions.ConsistencyScore*10))

	// Recent activity
	recentRepos := g.getRecentRepositories(prof, 30) // Last 30 days
	if len(recentRepos) > 0 {
		md.WriteString(fmt.Sprintf("- **Recent Activity:** Active in %d repositories in the last 30 days\n", len(recentRepos)))
	}

	if len(prof.Insights.RecommendedRoles) > 0 {
		md.WriteString(fmt.Sprintf("- **Recommended Roles:** %s\n", strings.Join(prof.Insights.RecommendedRoles, ", ")))
	}
	md.WriteString("\n")

	// Footer
	md.WriteString("---\n")
	md.WriteString(fmt.Sprintf("*Profile generated on %s | GitHub: [@%s](https://github.com/%s)*\n",
		time.Now().Format("January 2, 2006"), prof.Username, prof.Username))

	return md.String()
}

// generateTechnicalTemplate creates a detailed technical profile
func (g *Generator) generateTechnicalTemplate(prof *profile.UserProfile) string {
	var md strings.Builder

	md.WriteString(fmt.Sprintf("# Technical Profile - %s\n\n", prof.Username))

	// Technical Overview
	md.WriteString("## 🔧 Technical Overview\n\n")
	md.WriteString("### Language Proficiency Analysis\n\n")

	for _, lang := range prof.Languages {
		if lang.Percentage < 1 { // Skip languages with very low usage
			continue
		}

		yearsUsed := time.Since(lang.FirstUsed).Hours() / (24 * 365.25)
		md.WriteString(fmt.Sprintf("#### %s\n", lang.Language))
		md.WriteString(fmt.Sprintf("- **Usage:** %.1f%% of total codebase (%s lines)\n",
			lang.Percentage, g.formatNumber(lang.LinesOfCode)))
		md.WriteString(fmt.Sprintf("- **Experience:** %.1f years (%d projects)\n",
			yearsUsed, lang.ProjectCount))
		md.WriteString(fmt.Sprintf("- **Proficiency Score:** %.1f/10\n", lang.ProficiencyScore*10))
		md.WriteString("\n")
	}

	// Add Docker/Dockerfile if containerization expertise exists but wasn't detected as a language
	// (can happen when REST API rate limits prevent file content analysis)
	hasDockerLanguage := false
	for _, lang := range prof.Languages {
		if strings.ToLower(lang.Language) == "dockerfile" {
			hasDockerLanguage = true
			break
		}
	}

	log.Printf("Checking for Docker language: hasDockerLanguage=%v, technical_areas_count=%d", hasDockerLanguage, len(prof.Skills.TechnicalAreas))

	if !hasDockerLanguage {
		for _, area := range prof.Skills.TechnicalAreas {
			log.Printf("Checking technical area: %s, project_count=%d", area.Area, area.ProjectCount)
			if area.Area == "Containerization" && area.ProjectCount > 0 {
				log.Printf("Adding Dockerfile language section from containerization data")
				md.WriteString("#### Dockerfile\n")
				md.WriteString(fmt.Sprintf("- **Usage:** Docker/containerization expertise across %d projects\n", area.ProjectCount))
				md.WriteString(fmt.Sprintf("- **Experience:** %.1f years (%d projects)\n", area.YearsActive, area.ProjectCount))
				md.WriteString(fmt.Sprintf("- **Proficiency Score:** %.1f/10\n", area.Competency*10))
				if len(area.Technologies) > 0 {
					md.WriteString(fmt.Sprintf("- **Technologies:** %s\n", strings.Join(area.Technologies, ", ")))
				}
				md.WriteString("\n")
				break
			}
		}
	}

	// Repository Analysis
	md.WriteString("## 📊 Repository Analysis\n\n")

	ownedRepos := 0
	contributedRepos := 0
	totalStars := 0
	totalForks := 0

	for _, repo := range prof.Repositories {
		if repo.IsOwner {
			ownedRepos++
		} else {
			contributedRepos++
		}
		totalStars += repo.Stars
		totalForks += repo.Forks
	}

	md.WriteString(fmt.Sprintf("- **Repository Ownership:** %d owned, %d contributed\n", ownedRepos, contributedRepos))
	md.WriteString(fmt.Sprintf("- **Community Impact:** %d stars, %d forks received\n", totalStars, totalForks))
	md.WriteString(fmt.Sprintf("- **Code Volume:** %s total lines across %d repositories\n",
		g.formatNumber(g.getTotalLinesOfCode(prof)), len(prof.Repositories)))
	md.WriteString("\n")

	// Technical Areas
	if len(prof.Skills.TechnicalAreas) > 0 {
		md.WriteString("### Technical Expertise Areas\n\n")

		// Sort by competency
		areas := make([]profile.TechnicalArea, len(prof.Skills.TechnicalAreas))
		copy(areas, prof.Skills.TechnicalAreas)
		sort.Slice(areas, func(i, j int) bool {
			return areas[i].Competency > areas[j].Competency
		})

		for _, area := range areas {
			md.WriteString(fmt.Sprintf("- **%s:** %.1f/10 competency (%d projects, %.1f years active)\n",
				strings.Title(area.Area), area.Competency*10, area.ProjectCount, area.YearsActive))
		}
		md.WriteString("\n")
	}

	// Architecture & Design Patterns
	if len(prof.Insights.ArchitecturalThinking.ArchitecturalPatterns) > 0 {
		md.WriteString("### Architecture & Design Patterns\n\n")
		for _, pattern := range prof.Insights.ArchitecturalThinking.ArchitecturalPatterns {
			md.WriteString(fmt.Sprintf("- %s\n", pattern))
		}
		md.WriteString("\n")
	}

	// Detailed Project Breakdown
	md.WriteString("## 🚀 Project Portfolio\n\n")

	// Group repositories by language
	langRepos := make(map[string][]profile.RepositoryProfile)
	for _, repo := range prof.Repositories {
		if repo.Language != "" {
			langRepos[repo.Language] = append(langRepos[repo.Language], repo)
		}
	}

	// Show top projects for each primary language
	for _, lang := range prof.Skills.PrimaryLanguages {
		if repos, exists := langRepos[lang]; exists {
			// Sort by stars
			sort.Slice(repos, func(i, j int) bool {
				return repos[i].Stars > repos[j].Stars
			})

			md.WriteString(fmt.Sprintf("### %s Projects\n\n", lang))

			count := 0
			for _, repo := range repos {
				if count >= 5 { // Limit to top 5 per language
					break
				}

				md.WriteString(fmt.Sprintf("- **[%s](%s)**", repo.Name, repo.URL))
				if repo.Stars > 0 {
					md.WriteString(fmt.Sprintf(" ⭐ %d", repo.Stars))
				}
				md.WriteString("\n")

				if repo.Description != "" {
					md.WriteString(fmt.Sprintf("  - %s\n", repo.Description))
				}

				if len(repo.Topics) > 0 {
					md.WriteString(fmt.Sprintf("  - Technologies: %s\n", strings.Join(repo.Topics, ", ")))
				}

				count++
			}
			md.WriteString("\n")
		}
	}

	return md.String()
}

// generateExecutiveTemplate creates an executive summary focused template
func (g *Generator) generateExecutiveTemplate(prof *profile.UserProfile) string {
	var md strings.Builder

	md.WriteString(fmt.Sprintf("# Executive Technical Summary - %s\n\n", prof.Username))

	// Executive Summary
	md.WriteString("## Executive Summary\n\n")

	md.WriteString(fmt.Sprintf("%s-level software professional with %.0f years of active development experience. ",
		strings.Title(prof.Insights.CareerLevel), float64(prof.Contributions.ContributionYears)))

	if len(prof.Skills.PrimaryLanguages) > 0 {
		md.WriteString(fmt.Sprintf("Primary expertise in %s development. ",
			strings.Join(prof.Skills.PrimaryLanguages[:min(3, len(prof.Skills.PrimaryLanguages))], ", ")))
	}

	md.WriteString(fmt.Sprintf("Led or contributed to %d software projects with %d community stars received. ",
		len(prof.Repositories), g.getTotalStars(prof)))

	if len(prof.Organizations) > 1 {
		md.WriteString(fmt.Sprintf("Cross-functional collaboration experience across %d organizations. ", len(prof.Organizations)))
	}

	md.WriteString(fmt.Sprintf("Overall technical impact score: %.1f/10.\n\n", prof.Insights.OverallImpactScore*10))

	// Leadership & Impact
	md.WriteString("## Leadership & Impact\n\n")

	if len(prof.Insights.LeadershipIndicators) > 0 {
		for _, indicator := range prof.Insights.LeadershipIndicators {
			md.WriteString(fmt.Sprintf("- **%s:** %s (Confidence: %.1f/10)\n",
				strings.Title(strings.ReplaceAll(indicator.Type, "_", " ")),
				indicator.Description, indicator.Strength*10))
		}
	}

	// Community contributions
	osContributions := g.countOpenSourceContributions(prof)
	if osContributions > 0 {
		md.WriteString(fmt.Sprintf("- **Open Source Leadership:** %d public repositories contributing to the developer community\n", osContributions))
	}

	// Jenkins Community Leadership
	if prof.DiscourseProfile != nil && prof.DiscourseProfile.TrustLevel >= 2 {
		md.WriteString(fmt.Sprintf("- **Community Leadership:** Trust level %d in Jenkins community with %d solutions provided\n",
			prof.DiscourseProfile.TrustLevel, prof.DiscourseProfile.SolutionsCount))

		if prof.DiscourseProfile.CommunityMetrics.PeopleHelped > 0 {
			md.WriteString(fmt.Sprintf("- **Mentorship Impact:** Estimated %d+ community members helped through technical guidance\n",
				prof.DiscourseProfile.CommunityMetrics.PeopleHelped))
		}
	}

	md.WriteString("\n")

	// Strategic Technical Focus
	md.WriteString("## Strategic Technical Focus\n\n")

	if len(prof.Insights.TechnicalFocus) > 0 {
		md.WriteString("### Core Technology Stack\n")
		for _, tech := range prof.Insights.TechnicalFocus {
			// Find the corresponding language stats
			for _, lang := range prof.Languages {
				if strings.EqualFold(lang.Language, tech) {
					md.WriteString(fmt.Sprintf("- **%s:** %.1f%% of codebase, %d projects, %.1f years experience\n",
						tech, lang.Percentage, lang.ProjectCount,
						time.Since(lang.FirstUsed).Hours()/(24*365.25)))
					break
				}
			}
		}
		md.WriteString("\n")
	}

	// Organizational Impact
	if len(prof.Organizations) > 0 {
		md.WriteString("### Organizational Contributions\n")

		// Sort organizations by contribution count
		orgs := make([]profile.OrganizationProfile, len(prof.Organizations))
		copy(orgs, prof.Organizations)
		sort.Slice(orgs, func(i, j int) bool {
			return orgs[i].ContributionCount > orgs[j].ContributionCount
		})

		for _, org := range orgs[:min(5, len(orgs))] { // Top 5 organizations
			md.WriteString(fmt.Sprintf("- **%s:** %s role, %d project contributions\n",
				org.Name, strings.Title(org.Role), org.ContributionCount))
		}
		md.WriteString("\n")
	}

	// Recommended Executive Roles
	if len(prof.Insights.RecommendedRoles) > 0 {
		md.WriteString("## Recommended Leadership Roles\n\n")

		// Filter for senior/leadership roles
		var leadershipRoles []string
		for _, role := range prof.Insights.RecommendedRoles {
			roleLower := strings.ToLower(role)
			if strings.Contains(roleLower, "senior") || strings.Contains(roleLower, "lead") ||
			   strings.Contains(roleLower, "principal") || strings.Contains(roleLower, "staff") ||
			   strings.Contains(roleLower, "architect") {
				leadershipRoles = append(leadershipRoles, role)
			}
		}

		if len(leadershipRoles) == 0 {
			leadershipRoles = prof.Insights.RecommendedRoles // Fallback to all roles
		}

		for _, role := range leadershipRoles {
			md.WriteString(fmt.Sprintf("- %s\n", role))
		}
		md.WriteString("\n")
	}

	// Key Performance Metrics
	md.WriteString("## Key Performance Metrics\n\n")

	md.WriteString(fmt.Sprintf("- **Technical Productivity:** %d commits, %d pull requests, %d issues resolved\n",
		prof.Contributions.TotalCommits, prof.Contributions.TotalPullRequests, prof.Contributions.TotalIssues))

	md.WriteString(fmt.Sprintf("- **Project Leadership:** %d owned repositories, %.1f average stars per project\n",
		g.countOwnedRepositories(prof), g.getAverageStars(prof)))

	md.WriteString(fmt.Sprintf("- **Team Collaboration:** %d organization partnerships, %.1f consistency score\n",
		len(prof.Organizations), prof.Contributions.ConsistencyScore*10))

	md.WriteString(fmt.Sprintf("- **Technical Breadth:** %d programming languages, %d technology areas\n",
		len(prof.Languages), len(prof.Skills.TechnicalAreas)))

	return md.String()
}

// generateATSTemplate creates an ATS-optimized profile
func (g *Generator) generateATSTemplate(prof *profile.UserProfile) string {
	var md strings.Builder

	// ATS-friendly header (no special characters)
	md.WriteString(fmt.Sprintf("GITHUB PROFESSIONAL PROFILE - %s\n\n", strings.ToUpper(prof.Username)))

	if prof.Name != "" {
		md.WriteString(fmt.Sprintf("Name: %s\n", prof.Name))
	}
	if prof.Location != "" {
		md.WriteString(fmt.Sprintf("Location: %s\n", prof.Location))
	}
	if prof.Company != "" {
		md.WriteString(fmt.Sprintf("Current Company: %s\n", prof.Company))
	}
	md.WriteString("\n")

	// Skills section (keyword-heavy for ATS)
	md.WriteString("TECHNICAL SKILLS\n\n")

	md.WriteString("Programming Languages: ")
	var languages []string
	for _, lang := range prof.Languages {
		if lang.Percentage > 1 { // Only include languages with >1% usage
			languages = append(languages, lang.Language)
		}
	}
	md.WriteString(strings.Join(languages, ", "))
	md.WriteString("\n\n")

	if len(prof.Skills.Frameworks) > 0 {
		md.WriteString("Frameworks and Libraries: ")
		frameworks := g.getTechnologyNames(prof.Skills.Frameworks, 10)
		md.WriteString(strings.Join(frameworks, ", "))
		md.WriteString("\n\n")
	}

	if len(prof.Skills.Databases) > 0 {
		md.WriteString("Databases: ")
		databases := g.getTechnologyNames(prof.Skills.Databases, 10)
		md.WriteString(strings.Join(databases, ", "))
		md.WriteString("\n\n")
	}

	if len(prof.Skills.CloudPlatforms) > 0 {
		md.WriteString("Cloud Platforms: ")
		cloud := g.getTechnologyNames(prof.Skills.CloudPlatforms, 10)
		md.WriteString(strings.Join(cloud, ", "))
		md.WriteString("\n\n")
	}

	// Experience section
	md.WriteString("PROFESSIONAL EXPERIENCE\n\n")

	md.WriteString(fmt.Sprintf("Software Developer | %d Years Active Development\n", prof.Contributions.ContributionYears))
	md.WriteString(fmt.Sprintf("Career Level: %s\n\n", strings.Title(prof.Insights.CareerLevel)))

	// Key achievements (bullet points)
	md.WriteString("Key Achievements:\n")
	md.WriteString(fmt.Sprintf("- Developed and maintained %d software repositories\n", len(prof.Repositories)))

	// Add Docker Hub achievements if significant
	if prof.DockerHubProfile != nil && prof.DockerHubProfile.TotalDownloads > 100000 {
		md.WriteString(fmt.Sprintf("- Created %d Docker containers with %s total downloads\n",
			prof.DockerHubProfile.TotalImages, g.formatLargeNumber(prof.DockerHubProfile.TotalDownloads)))
	}

	// Add Discourse community achievements if significant
	if prof.DiscourseProfile != nil && prof.DiscourseProfile.SolutionsCount > 10 {
		md.WriteString(fmt.Sprintf("- Provided %d solutions in Jenkins community with trust level %d recognition\n",
			prof.DiscourseProfile.SolutionsCount, prof.DiscourseProfile.TrustLevel))
	}

	md.WriteString(fmt.Sprintf("- Contributed %d commits across %d programming languages\n",
		prof.Contributions.TotalCommits, len(prof.Languages)))
	md.WriteString(fmt.Sprintf("- Received %d community stars for open source contributions\n", g.getTotalStars(prof)))

	if len(prof.Organizations) > 0 {
		md.WriteString(fmt.Sprintf("- Collaborated across %d professional organizations\n", len(prof.Organizations)))
	}

	if prof.Insights.OverallImpactScore > 0.7 {
		md.WriteString("- Demonstrated high-impact technical leadership and project ownership\n")
	}
	md.WriteString("\n")

	// Organizations
	if len(prof.Organizations) > 0 {
		md.WriteString("ORGANIZATIONAL EXPERIENCE\n\n")

		for _, org := range prof.Organizations {
			if org.ContributionCount > 0 {
				md.WriteString(fmt.Sprintf("%s - %s\n", org.Name, strings.Title(org.Role)))
				md.WriteString(fmt.Sprintf("Contributed to %d projects\n", org.ContributionCount))
				md.WriteString("\n")
			}
		}
	}

	// Projects (top repositories)
	md.WriteString("NOTABLE PROJECTS\n\n")

	notableRepos := g.getNotableRepositories(prof)
	for i, repo := range notableRepos {
		if i >= 5 { // Limit to top 5 for ATS
			break
		}

		md.WriteString(fmt.Sprintf("%s\n", repo.Name))
		if repo.Description != "" {
			md.WriteString(fmt.Sprintf("Description: %s\n", repo.Description))
		}
		md.WriteString(fmt.Sprintf("Technology: %s\n", repo.Language))
		if repo.Stars > 0 {
			md.WriteString(fmt.Sprintf("Community Recognition: %d stars\n", repo.Stars))
		}
		md.WriteString("\n")
	}

	// Education/Certifications equivalent
	md.WriteString("TECHNICAL CERTIFICATIONS AND EXPERTISE\n\n")

	for _, lang := range prof.Languages[:min(5, len(prof.Languages))] {
		profLevel := "Intermediate"
		if lang.Percentage > 25 {
			profLevel = "Advanced"
		} else if lang.Percentage > 40 {
			profLevel = "Expert"
		}

		md.WriteString(fmt.Sprintf("%s Development - %s Level\n", lang.Language, profLevel))
		md.WriteString(fmt.Sprintf("Experience: %d projects, %.1f years\n\n",
			lang.ProjectCount, time.Since(lang.FirstUsed).Hours()/(24*365.25)))
	}

	return md.String()
}

// Helper functions

func (g *Generator) getTotalStars(prof *profile.UserProfile) int {
	total := 0
	for _, repo := range prof.Repositories {
		total += repo.Stars
	}
	return total
}

func (g *Generator) getTotalLinesOfCode(prof *profile.UserProfile) int {
	total := 0
	for _, lang := range prof.Languages {
		total += lang.LinesOfCode
	}
	return total
}

func (g *Generator) getNotableRepositories(prof *profile.UserProfile) []profile.RepositoryProfile {
	var notable []profile.RepositoryProfile

	for _, repo := range prof.Repositories {
		// Consider notable if: has stars, is owned by user, or has significant size
		if repo.Stars > 0 || repo.IsOwner || repo.Size > 1000 {
			notable = append(notable, repo)
		}
	}

	// Sort by stars, then by size
	sort.Slice(notable, func(i, j int) bool {
		if notable[i].Stars != notable[j].Stars {
			return notable[i].Stars > notable[j].Stars
		}
		return notable[i].Size > notable[j].Size
	})

	// Return top 10
	if len(notable) > 10 {
		notable = notable[:10]
	}

	return notable
}

func (g *Generator) getRecentRepositories(prof *profile.UserProfile, days int) []profile.RepositoryProfile {
	cutoff := time.Now().AddDate(0, 0, -days)
	var recent []profile.RepositoryProfile

	for _, repo := range prof.Repositories {
		if repo.UpdatedAt.After(cutoff) {
			recent = append(recent, repo)
		}
	}

	return recent
}

func (g *Generator) getTechnologyNames(skills []profile.TechnologySkill, limit int) []string {
	var names []string

	// Sort by confidence
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Confidence > skills[j].Confidence
	})

	for i, skill := range skills {
		if i >= limit {
			break
		}
		names = append(names, skill.Name)
	}

	return names
}

func (g *Generator) countOpenSourceContributions(prof *profile.UserProfile) int {
	count := 0
	for _, repo := range prof.Repositories {
		if !repo.IsPrivate {
			count++
		}
	}
	return count
}

func (g *Generator) countOwnedRepositories(prof *profile.UserProfile) int {
	count := 0
	for _, repo := range prof.Repositories {
		if repo.IsOwner {
			count++
		}
	}
	return count
}

func (g *Generator) getAverageStars(prof *profile.UserProfile) float64 {
	ownedRepos := g.countOwnedRepositories(prof)
	if ownedRepos == 0 {
		return 0
	}

	totalStars := 0
	for _, repo := range prof.Repositories {
		if repo.IsOwner {
			totalStars += repo.Stars
		}
	}

	return float64(totalStars) / float64(ownedRepos)
}

func (g *Generator) formatNumber(num int) string {
	if num < 1000 {
		return fmt.Sprintf("%d", num)
	} else if num < 1000000 {
		return fmt.Sprintf("%.1fK", float64(num)/1000)
	} else {
		return fmt.Sprintf("%.1fM", float64(num)/1000000)
	}
}

func (g *Generator) formatLargeNumber(num int64) string {
	if num < 1000 {
		return fmt.Sprintf("%d", num)
	} else if num < 1000000 {
		return fmt.Sprintf("%.1fK", float64(num)/1000)
	} else if num < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(num)/1000000)
	} else {
		return fmt.Sprintf("%.1fB", float64(num)/1000000000)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}