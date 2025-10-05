# GitHub Profile Analyzer

Generate professional resume-ready profiles from GitHub user data. Transform your GitHub activity into compelling career narratives optimized for job applications, ATS systems, and technical interviews.

## 🚀 Features

- **Multiple Profile Templates**
  - `resume` - Professional resume enhancement
  - `technical` - Deep technical analysis and skills breakdown
  - `executive` - Leadership and high-level impact focus
  - `ats` - Optimized for Applicant Tracking Systems

- **Comprehensive Analysis**
  - Programming language proficiency scoring
  - Organization and collaboration patterns
  - Project impact assessment
  - Career level determination
  - Technology stack inference
  - Leadership indicators

- **Smart Insights**
  - AI-powered career recommendations
  - Skill gap analysis
  - Impact scoring
  - Role suggestions based on experience

## 📦 Installation

### Option 1: Pre-built Binaries (Recommended)

Download the latest release for your platform from the [Releases page](https://github.com/gounthar/alpha-omega-stats/releases).

Available platforms:
- **Windows (x64)**: `github-profile-tools-v1.0.x-windows-amd64.zip`
- **Linux (x64)**: `github-profile-tools-v1.0.x-linux-amd64.tar.gz`
- **Linux (ARM64)**: `github-profile-tools-v1.0.x-linux-arm64.tar.gz`
- **macOS (Intel)**: `github-profile-tools-v1.0.x-darwin-amd64.tar.gz`
- **macOS (Apple Silicon)**: `github-profile-tools-v1.0.x-darwin-arm64.tar.gz`

Extract the archive and run the binary directly - no additional setup required!

### Option 2: Build from Source

#### Prerequisites

- Go 1.21 or higher
- GitHub personal access token
- Internet connection

#### Setup

1. **Clone or download the project**
   ```bash
   git clone https://github.com/gounthar/alpha-omega-stats.git
   cd alpha-omega-stats/github-profile-tools
   ```

2. **Install Go dependencies**
   ```bash
   go mod download
   ```

3. **Get a GitHub Token**
   - Visit [GitHub Settings > Personal Access Tokens](https://github.com/settings/tokens)
   - Create a new token with `repo`, `read:org`, and `read:user` scopes
   - Set the environment variable:
     ```bash
     export GITHUB_TOKEN="your_token_here"
     ```

4. **Build the application**
   ```bash
   go build -o github-user-analyzer ./cmd/github-user-analyzer
   ```

## 🎯 Quick Start

### Basic Usage

```bash
# Analyze a GitHub user with default (all templates)
./github-user-analyzer -user octocat

# Generate technical deep-dive profile
./github-user-analyzer -user octocat -template technical

# Create ATS-optimized profile
./github-user-analyzer -user octocat -template ats -format markdown

# Executive summary for leadership roles
./github-user-analyzer -user octocat -template executive -output ./resumes
```

### Command Line Options

```
Usage: ./github-user-analyzer -user USERNAME [OPTIONS]

Required:
  -user string          GitHub username to analyze

Options:
  -token string         GitHub API token (or set GITHUB_TOKEN env var)
  -template string      Template type: resume, technical, executive, ats, all (default "all")
  -format string        Output format: markdown, json, both (default "both")
  -output string        Output directory (default "./data/profiles")
  -timeout string       Analysis timeout (e.g., '30m', '2h', '6h') (default "6h")
  -debug-log string     Debug log file path (default "github-user-analyzer-debug.log")
  -verbose              Enable verbose logging
  -version              Show version and exit
```

### Using the Shell Script (Linux/macOS)

```bash
# Make script executable
chmod +x scripts/analyze-user.sh

# Run with convenient wrapper
./scripts/analyze-user.sh octocat
./scripts/analyze-user.sh -t technical -v octocat
./scripts/analyze-user.sh --template ats --output ./resumes octocat
```

## 📋 Templates Overview

### 1. Resume Template (`resume`)
Perfect for enhancing existing resumes with GitHub data.

**Focus Areas:**
- Professional contribution overview
- Organization involvement
- Notable projects with impact metrics
- Technical skills with proficiency levels
- Career insights and recommendations

**Best For:** General job applications, career transitions

### 2. Technical Template (`technical`)
Deep dive into technical expertise and coding patterns.

**Focus Areas:**
- Language proficiency analysis with years of experience
- Repository analysis by technology
- Architecture and design patterns
- Project portfolio breakdown
- Technical expertise areas

**Best For:** Senior developer roles, technical interviews, peer review

### 3. Executive Template (`executive`)
Leadership-focused profile for management and senior positions.

**Focus Areas:**
- Strategic technical leadership
- Cross-organizational impact
- Team collaboration metrics
- Business-level project outcomes
- Executive role recommendations

**Best For:** Tech lead, engineering manager, CTO positions

### 4. ATS Template (`ats`)
Optimized for Applicant Tracking Systems and automated screening.

**Focus Areas:**
- Keyword-optimized skills sections
- Clean formatting for parsing
- Quantified achievements
- Industry-standard terminology
- Structured experience sections

**Best For:** Large company applications, automated screening systems

## 📊 Analysis Insights

The analyzer provides comprehensive insights including:

### Career Level Assessment
- **Junior** (0-2 years) - Learning and growing
- **Mid** (2-5 years) - Established contributor
- **Senior** (5+ years) - Expert and mentor
- **Principal** (8+ years) - Technical leader

### Impact Scoring (0-10 scale)
- Repository ownership and stars
- Contribution consistency
- Cross-team collaboration
- Technical breadth and depth

### Technology Proficiency
- Language usage percentage and years
- Framework and tool experience
- Cloud platform familiarity
- Architecture pattern recognition

### Leadership Indicators
- Project ownership patterns
- Mentorship signs in code reviews
- Cross-organizational contributions
- Community impact metrics

## 📁 Output Examples

### Generated Files
```
data/profiles/
├── octocat_profile.json              # Raw analysis data
├── octocat_profile_resume.md         # Resume-focused profile
├── octocat_profile_technical.md      # Technical deep-dive
├── octocat_profile_executive.md      # Leadership summary
└── octocat_profile_ats.md           # ATS-optimized version
```

### Sample Resume Profile Output
```markdown
# GitHub Professional Profile - octocat

## 📊 Contribution Overview
- **1,337** total contributions across **8** years of active development
- **42** repositories with **15,234** stars received
- Active contributor in **5** organizations
- Proficient in **12** programming languages
- Career Level: **Senior**

## 🏢 Organization Contributions
### GitHub
- **Role:** Member | **Contributions:** 25 repositories
- **Key Projects:** Hello-World, Spoon-Knife, octocat.github.io

## 💼 Notable Projects
### [Hello-World](https://github.com/octocat/Hello-World) ⭐ 1,420
**Description:** My first repository on GitHub!
- **Language:** JavaScript | **Size:** 2.1 MB
- **Technologies:** node, npm, javascript
- **Contributions:** 87 commits (+2,341/-892 lines)
```

## 🔧 Development

### Project Structure
```
github-profile-tools/
├── cmd/
│   ├── github-user-analyzer/         # Main CLI application
│   └── github-profile-updater/       # Incremental updater (future)
├── internal/
│   ├── github/                       # GitHub API client
│   ├── profile/                      # Profile analysis logic
│   ├── markdown/                     # Markdown generation
│   └── storage/                      # Data persistence (future)
├── templates/                        # Profile templates
├── scripts/                          # Convenience scripts
└── data/profiles/                   # Generated profiles
```

### Adding New Templates

1. **Define template in `markdown/generator.go`**
   ```go
   const NewTemplate TemplateType = "new"
   ```

2. **Implement generation function**
   ```go
   func (g *Generator) generateNewTemplate(prof *profile.UserProfile) string {
       // Template implementation
   }
   ```

3. **Add to switch statement in `GenerateMarkdown`**

### Extending Analysis

1. **Add new data fields to `profile/types.go`**
2. **Implement collection logic in `profile/analyzer.go`**
3. **Update templates to include new insights**

## 🛠 Advanced Usage

### Batch Analysis
```bash
# Analyze multiple users
for user in user1 user2 user3; do
    ./github-user-analyzer -user "$user" -template resume
done
```

### Custom Output Processing
```bash
# Generate and immediately copy to clipboard (macOS)
./github-user-analyzer -user octocat -format markdown | pbcopy

# Pipe to other tools for further processing
./github-user-analyzer -user octocat -format json | jq '.insights.recommendedRoles'
```

### Integration with Resume Tools
```bash
# Generate LaTeX-friendly format for academic CVs
./github-user-analyzer -user octocat -template technical -format markdown > github_section.md

# Create multiple versions for different job applications
./github-user-analyzer -user octocat -template ats -output ./job-applications/company1/
./github-user-analyzer -user octocat -template executive -output ./job-applications/company2/
```

## 🔒 Privacy & Security

- **Public Data Only**: Analyzes only public GitHub activity
- **Token Security**: GitHub tokens are not stored or logged
- **Data Retention**: Generated files are stored locally only
- **Rate Limiting**: Respects GitHub API rate limits with exponential backoff

## 🚧 Roadmap

### Phase 1: MVP (Current)
- [x] Core GitHub data collection
- [x] Four template types
- [x] CLI interface
- [x] Basic insights generation

### Phase 2: Enhanced Features
- [ ] Incremental profile updates
- [ ] Job description optimization
- [ ] Multi-platform integration (GitLab, Stack Overflow)
- [ ] Web interface

### Phase 3: Advanced Intelligence
- [ ] AI-powered resume enhancement suggestions
- [ ] Industry benchmarking
- [ ] A/B testing for profile optimization
- [ ] Integration with job boards

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

### Common Issues

**"Rate limit exceeded"**
- Wait a few minutes and try again
- Ensure your GitHub token has sufficient quota

**"User not found"**
- Verify the username spelling
- Ensure the user has public repositories

**"Token authentication failed"**
- Check your GitHub token is valid
- Ensure token has `repo`, `read:org`, `read:user` scopes

### Getting Help

1. Check the [Issues](https://github.com/gounthar/alpha-omega-stats/issues) page
2. Create a new issue with:
   - Command used
   - Error message
   - Expected vs actual behavior

---

**Transform your GitHub activity into career opportunities! 🚀**