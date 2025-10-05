# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Data Collection
- **Monthly PR collection**: `./collect-monthly.sh "YYYY-MM" [true/false]` - Collects PR data for specified month, optional Google Sheets update
- **Daily updates**: `./update-daily.sh` - Updates current month's data and failing PR statistics
- **Bulk collection with retries**: `./retry-collection.sh` - Collects all data from July 2024 onwards with exponential backoff
- **JDK 25 testing**: `./test-jdk-25.sh` - Tests top 250 plugins with JDK 25, writes results to CSV
- **Install JDK versions**: `./install-jdk-versions.sh` - Installs multiple JDK versions for compatibility testing

### Development Setup
- **Go dependencies**: `go mod download && go mod tidy`
- **Go build**: `go build jenkins-pr-collector.go` - Builds the main collector binary
- **Go run directly**: `go run jenkins-pr-collector.go -start YYYY-MM-DD -end YYYY-MM-DD -output file.json`
- **Python environment**: `python -m venv venv && source venv/bin/activate && pip install -r requirements.txt`
- **Environment check**: `./check-env.sh` - Validates required tools and credentials

### Testing and Analysis
- **Find JUnit 5 migration PRs**: `./find-junit5-prs.sh` - Searches for JUnit 5 migration-related PRs
- **Test plugin builds**: `./test-pr-builds.sh` - Tests building plugins from PR branches
- **Validate plugin list**: `./validate-top-plugins.sh` - Validates top-250-plugins.csv format
- **Analyze JUnit 5 PRs**: `./analyze-junit5-prs.sh` - Analyzes JUnit 5 migration patterns
- **Generate reports**: `./generate-report.sh` - Creates consolidated statistics reports

### Utility Scripts
- **Filter PRs**: `./filter-prs.sh input.json` - Filters PR data by Jenkins-related criteria
- **Group PRs**: `./group-prs.sh input.json plugins.json` - Groups PRs by repository/plugin
- **Count PRs**: `./count_prs.sh repos.txt year` - Counts PRs for specific repositories
- **Process PRs**: `./process_prs.sh` - General PR data processing pipeline

### GitHub Profile Tools
- **Build analyzer**: `cd github-profile-tools && go build -o github-user-analyzer ./cmd/github-user-analyzer` - Builds the GitHub profile analyzer binary
- **Analyze user**: `./github-profile-tools/github-user-analyzer -user=username` - Generates comprehensive GitHub profile analysis with all templates by default
- **Analyze with specific template**: `./github-profile-tools/github-user-analyzer -user=username -template=resume` - Generates profile with specific template (resume, technical, executive, ats)
- **Analyze with token**: `./github-profile-tools/github-user-analyzer -user=username -token="$GITHUB_TOKEN"` - Uses explicit GitHub token for API access

## Architecture

### Core Components
1. **jenkins-pr-collector.go** - Main Go application that queries GitHub GraphQL API to collect PR data
2. **Shell scripts ecosystem** - Bash scripts orchestrate data collection, processing, and reporting
3. **Python integration** - Handles Google Sheets uploads and data processing via `upload_to_sheets.py`
4. **GitHub Actions workflows** - Automated scheduling and execution in `.github/workflows/`

### Data Flow
1. **Collection**: `jenkins-pr-collector.go` fetches PR data via GitHub GraphQL API
2. **Processing**: Scripts filter, group, and transform raw PR data
3. **Storage**: Data stored in `data/` directory (monthly/, consolidated/, archive/)
4. **Reporting**: Processed data uploaded to Google Sheets and stored as JSON artifacts

### Key Data Structures
- **Raw PR data**: JSON files containing GitHub PR objects with metadata
- **Filtered data**: PRs filtered by criteria (Jenkins-related, specific time periods)
- **Grouped data**: PRs organized by plugin/repository for analysis
- **Build results**: CSV files with plugin build status and JDK compatibility

### Authentication
- **GitHub API**: Requires `GITHUB_TOKEN` or `PAT_TOKEN` environment variable with repo, read:org, read:user scopes
- **Google Sheets**: Requires `GOOGLE_CREDENTIALS` JSON service account file (set via environment or file path)
- **Rate limiting**: Built-in exponential backoff and retry mechanisms for both GitHub and Google APIs

### Automation Workflows
- **pr-stats.yml**: Monthly collection (2nd of month at 00:00 UTC) and daily updates (midnight UTC)
- **test-jdk-25.yml**: Weekly JDK 25 compatibility testing (Tuesdays at 00:00 UTC) with 6-hour timeout
- **updatecli.yml**: Daily updates to top-250-plugins.csv from upstream Jenkins data
- **auto-merge-bot-prs.yml**: Automatically merges bot-created PRs with "automation" label
- **pr-collector-test.yml**: Weekly testing of PR collector functionality (Tuesdays at 07:18 UTC)
- **generate-top-plugins.yml**: Updates plugin popularity data
- **run-update-daily-on-merge.yml**: Triggers daily update when changes are merged
- **release-github-profile-tools.yml**: Automated releases of GitHub Profile Tools with cross-platform binaries

### Release Automation Plan
The GitHub Profile Tools binary will be automatically released through GitHub Actions:

#### Release Strategy
- **Trigger**: Tag-based releases using semantic versioning (v1.0.0, v1.1.0, etc.)
- **Platforms**: Cross-platform binaries for Windows (x64), Linux (x64, ARM64), and macOS (x64, ARM64)
- **Artifacts**: Compressed binaries with checksums for verification
- **Changelog**: Auto-generated release notes from commit messages and PR titles

#### Build Matrix
All builds use `ubuntu-latest` with Go cross-compilation:
- **Windows x64**: `GOOS=windows GOARCH=amd64` → `github-user-analyzer-windows-amd64.exe`
- **Linux x64**: `GOOS=linux GOARCH=amd64` → `github-user-analyzer-linux-amd64`
- **Linux ARM64**: `GOOS=linux GOARCH=arm64` → `github-user-analyzer-linux-arm64`
- **macOS x64**: `GOOS=darwin GOARCH=amd64` → `github-user-analyzer-darwin-amd64`
- **macOS ARM64**: `GOOS=darwin GOARCH=arm64` → `github-user-analyzer-darwin-arm64`

#### Release Process
1. **Tag Creation**: Create and push a version tag (e.g., `git tag v1.0.0 && git push origin v1.0.0`)
2. **Automated Build**: GitHub Action builds binaries for all supported platforms
3. **Testing**: Run basic smoke tests on each binary
4. **Packaging**: Compress binaries and generate SHA256 checksums
5. **Release**: Create GitHub release with auto-generated changelog and download links
6. **Notification**: Optional notifications to relevant channels

### Directory Structure
- `data/monthly/` - Monthly PR collection files (prs_YYYY_MM.json, filtered_*, grouped_*)
- `data/consolidated/` - Aggregated data across time periods
- `data/archive/` - Older data files (>6 months)
- `data/junit5/` - JUnit 5 migration analysis data
- `data/profiles/` - Generated GitHub profile analyses and templates
- `data/cache/` - Cached analysis data for efficient template regeneration
- `github-profile-tools/` - GitHub profile analyzer Go application
  - `cmd/github-user-analyzer/` - Main CLI application entry point
  - `internal/github/` - GitHub API client and GraphQL queries
  - `internal/profile/` - Profile analysis logic and types
  - `internal/docker/` - Docker Hub integration
  - `internal/discourse/` - Discourse community analysis
  - `templates/` - Profile generation templates (resume, technical, executive, ats)
- `updatecli/` - Updatecli configuration for dependency updates
- `.github/workflows/` - GitHub Actions automation workflows

### Key Files and Formats
- **Raw PR data**: `prs_YYYY_MM.json` - Complete GitHub PR objects from GraphQL API
- **Filtered data**: `filtered_prs_YYYY_MM.json` - Jenkins-related PRs only
- **Grouped data**: `grouped_prs_YYYY_MM.json` - PRs organized by repository/plugin
- **JDK compatibility**: `jdk-25-build-results.csv` - Plugin build results with JDK 25
- **Plugin list**: `top-250-plugins.csv` - Most popular Jenkins plugins for testing

### Error Handling
- Scripts use `set -e` for immediate exit on errors
- Rate limiting handled with exponential backoff for GitHub and Google APIs
- Partial data preservation during collection failures with resume capability
- Comprehensive logging to stdout/stderr and dedicated log files
- Debug logging to `build-debug.log`, `fetch_prs_debug.log`, and other log files