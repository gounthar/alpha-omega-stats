# Jenkins Plugin PR Statistics Collector

This repository contains tools and automation for collecting and analyzing Pull Request (PR) statistics for Jenkins plugins. It helps track open, merged, and failing PRs across the Jenkins ecosystem.

## Overview

The system collects PR data from GitHub repositories related to Jenkins plugins, processes this data, and uploads statistics to Google Sheets for analysis. The collection process runs both automatically (via GitHub Actions) and can be run manually when needed.

## Scripts and Their Functions

### Core Scripts

1. `jenkins-pr-collector.go`
   - Main data collection script written in Go
   - Queries GitHub's GraphQL API to fetch PR data for Jenkins plugins
   - Usage: `go run jenkins-pr-collector.go -start "YYYY-MM-DD" -end "YYYY-MM-DD" -output "output_file.json"`
   - Logs output to stdout/stderr for monitoring

2. `collect-monthly.sh`
   - Collects PR data for a specific month
   - Parameters:
     - `YYYY-MM`: Target month (optional, defaults to last month)
     - `UPDATE_SHEETS`: Boolean flag to update Google Sheets (optional, defaults to false)
   - Creates monthly data files in `data/monthly/`
   - Updates consolidated data files in `data/consolidated/`
   - Usage: `./collect-monthly.sh "2024-03" true`
   - Logs progress and errors to stdout

3. `count_prs.sh`
   - Counts pull requests for specified repositories
   - Takes a text file containing repository names and a year
   - Generates repository-specific PR statistics
   - Usage: `./count_prs.sh repos.txt 2024`
   - Outputs counts to stdout and generates summary report

4. `compute-stats.sh`
   - Generates detailed PR statistics for specific users
   - Analyzes PR patterns and contributions
   - Parameters:
     - List of GitHub usernames (comma-separated)
     - Date range (start and end dates)
   - Usage: `./compute-stats.sh user1,user2 YYYY-MM-DD YYYY-MM-DD`
   - Outputs detailed statistics report

5. `group-prs.sh`
   - Processes and groups PR data by title and status
   - Called by `collect-monthly.sh`
   - Requires `plugins.json` file for plugin information
   - Usage: `./group-prs.sh "input_file.json" "plugins.json"`
   - Logs grouping statistics to stdout

6. `retry-collection.sh`
   - Bulk data collection script with retry mechanism
   - Collects data from July 2024 onwards
   - Implements exponential backoff for failed attempts
   - Updates Google Sheets only after all data is collected
   - Usage: `./retry-collection.sh`
   - Logs retry attempts and progress to stdout

### Supporting Scripts

7. `upload_to_sheets.py`
   - Python script for uploading data to Google Sheets
   - Requires Google Sheets API credentials
   - Called by other scripts when `UPDATE_SHEETS` is true
   - Logs upload status and any API errors to stdout

## Directory Structure

\`\`\`
.
├── data/
│   ├── monthly/      # Monthly PR data files
│   ├── consolidated/ # Consolidated data files
│   ├── archive/      # Archived data (older than 6 months)
│   └── backup/       # Backup directory for data files
├── .github/
│   └── workflows/    # GitHub Actions workflow files
├── updatecli/
│   ├── updatecli.d/  # Updatecli manifests
│   └── .       # Configuration values for Updatecli
└── scripts/         # Collection and processing scripts
\`\`\`

## Automated Workflows

### PR Stats Workflow (`pr-stats.yml`)
- **Monthly Collection** (2nd of each month)
  - Runs full data collection for the previous month
  - Updates consolidated statistics
  - Updates Google Sheets
  - Creates a backup of all data before running
  - Logs available in GitHub Actions run history
  - Expected duration: 15–30 minutes

- **Daily Updates** (midnight UTC)
  - Updates current month's data
  - Updates open and failing PR statistics
  - Updates Google Sheets with latest data
  - Creates a backup of current data
  - Logs available in GitHub Actions run history
  - Expected duration: 5–10 minutes

### Updatecli Workflow (`updatecli.yml`)
- **Daily Check** (midnight UTC)
  - Checks for updates to the `top-250-plugins.csv` file from the upstream source
  - Creates a pull request when changes are detected
  - Updates the local file with the latest content
  - Logs available in GitHub Actions run history
  - Expected duration: 1–2 minutes

### Required Secrets and Permissions

The workflows require proper authentication to access GitHub's API. Set up the following:

1. **GitHub Token**:
   - Go to repository Settings → Secrets and variables → Actions
   - Add a new repository secret named `GH_TOKEN` or `PAT_TOKEN`
   - Use a Personal Access Token (PAT) with the following permissions:
     - `repo` (Full repository access)
     - `read:org` (Read organization data)
     - `read:user` (Read user data)
   - The token should have sufficient scope to access Jenkins organization repositories

2. **Updatecli GitHub Token**:
   - For the Updatecli workflow, the default `GITHUB_TOKEN` is used
   - No additional configuration is needed as the workflow uses the built-in token

3. **Workflow Permissions**:
   - Go to repository Settings → Actions → General
   - Under "Workflow permissions", select:
     - "Read and write permissions"
     - Check "Allow GitHub Actions to create and approve pull requests"

### PR Collector Test (`pr-collector-test.yml`)
- Runs every Tuesday at 07:18 UTC
- Tests the PR collector functionality
- Creates a pull request with updated statistics
- Uses Docker for isolated testing environment
- Logs available in GitHub Actions run history
- Expected duration: 10–15 minutes

## Logging and Monitoring

### GitHub Actions Logs
- All automated runs log their output to GitHub Actions
- Access logs through the "Actions" tab in the repository
- Logs are retained for 90 days
- Each run includes:
  - Setup steps
  - Script execution output
  - Error messages (if any)
  - Completion status

### Data Collection Logs
- Scripts log to stdout/stderr
- Key information logged includes:
  - Start and end times of operations
  - Number of PRs processed
  - API rate limit status
  - Error messages and retry attempts
  - Google Sheets update status

### Monitoring Points
1. **GitHub Actions Status**
   - Check Actions tab for failed runs
   - Review logs for rate limit warnings
   - Verify backup creation

2. **Data Integrity**
   - Verify monthly files are created
   - Check consolidated data updates
   - Confirm Google Sheets updates

3. **Storage Management**
   - Monitor backup directory size
   - Check archive rotation
   - Verify data retention policies

## Getting Started

### Initial Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/your-org/alpha-omega-stats.git
   cd alpha-omega-stats
   ```

2. Install dependencies:
   ```bash
   # Go dependencies
   go mod download

   # Python dependencies
   python -m venv venv
   source venv/bin/activate  # or `venv\Scripts\activate` on Windows
   pip install -r requirements.txt
   ```

3. Set up credentials:
   - Create a GitHub token with necessary permissions
   - Set up Google Sheets API credentials
   - Configure environment variables as needed

### Running Data Collection

#### For Initial Data Collection
```bash
# This will collect all data from July 2024 onwards
./retry-collection.sh
```

#### For Monthly Maintenance
```bash
# Collect data for a specific month
./collect-monthly.sh "YYYY-MM" true
```

## Example Usage

Here are some common usage examples:

```bash
# Collect PR data for March 2024 and update Google Sheets
./collect-monthly.sh "2024-03" true

# Process and group PRs from a JSON file
./group-prs.sh "data/monthly/prs_2024_03.json" "plugins.json"

# Collect historical data with automatic retries
./retry-collection.sh
```

### Command Explanations

#### Monthly Collection
The `collect-monthly.sh` script collects PR data for a specific month:
- First argument: Month in YYYY-MM format (optional, defaults to last month)
- Second argument: Whether to update Google Sheets (optional, defaults to false)

#### Group PRs
The `group-prs.sh` script organizes pull requests by plugin:
- First argument: JSON file containing PR data
- Second argument: Plugin configuration file

#### Retry Collection
The `retry-collection.sh` script performs bulk data collection with retry mechanism:
- No arguments required
- Collects all data from July 2024 onwards
- Implements automatic retries with exponential backoff

## Sequence Diagram(s)

```mermaid
sequenceDiagram
    participant GitHub as GitHub Actions
    participant Runner as Workflow Runner
    participant Checkout as Checkout Code
    participant EnvSetup as Environment Setup (Go, Python, CLI)
    participant Script as Collection/Update Script
    participant Artifact as Data Artifacts

    GitHub->>Runner: Trigger workflow (Scheduled/Manual)
    Runner->>Checkout: Checkout repository
    Runner->>EnvSetup: Set up environments & install dependencies (jq, GitHub CLI, Python deps)
    EnvSetup-->>Runner: Environment ready
    Runner->>Script: Execute script based on event type
    alt Scheduled Monthly
        Script->>Script: Run collect-monthly.sh
    else Daily/Manual
        Script->>Script: Run update-daily.sh
    end
    Script->>Artifact: Upload updated PR JSON artifacts
    Artifact-->>Runner: Artifacts stored
```

```mermaid
sequenceDiagram
    participant Client as GraphQLClient
    participant API as GitHub GraphQL API
    participant Retry as Retry Logic
    participant Storage as Partial Data Storage

    Client->>API: Execute GraphQL query
    API-->>Client: Response/Error
    alt Error is retryable?
        Client->>Retry: Call isRetryableError
        Retry-->>Client: Error qualifies, initiate exponential backoff
        loop Up to max attempts
            Client->>API: Retry GraphQL query
        end
    else Successful Response
        Client->>Storage: Save partial data if needed
        Client-->>Client: Process and return data
    end
```

## Maintenance Tasks

### Monthly Tasks
1. Check the automated collection ran successfully on the 2nd
   - Review GitHub Actions logs
   - Verify data files are created
   - Check Google Sheets updates

2. Verify data in Google Sheets is updated
   - Check latest data timestamp
   - Verify all sheets are updated
   - Review data consistency

3. Review any failed collections in the GitHub Actions logs
   - Check for rate limit issues
   - Review error messages
   - Plan retries if needed

### As-Needed Tasks
1. Review and clean up archived data
   - Verify archive rotation
   - Check storage usage
   - Clean up old backups

2. Verify backup integrity
   - Test backup restoration
   - Check backup completeness
   - Update backup strategy if needed

3. Update dependencies as needed
   - Check for security updates
   - Review dependency versions
   - Test updates in development

## Troubleshooting

1. **Rate Limiting**
   - The scripts include built-in retry mechanisms with exponential backoff
   - Check GitHub API quota in the logs
   - Adjust collection timing if needed
   - Monitor rate limit headers in responses

2. **Failed Collections**
   - Check the logs in `data/monthly/` for specific errors
   - Use `retry-collection.sh` to retry failed periods
   - Verify GitHub token permissions
   - Review network connectivity issues

3. **Google Sheets Issues**
   - Verify API credentials are valid
   - Check Python virtual environment is activated
   - Review logs for API errors
   - Verify sheet permissions

4. **Data Inconsistencies**
   - Compare monthly and consolidated data
   - Check for missing or duplicate entries
   - Verify data format consistency
   - Review archive integrity

## Contributing

1. Fork the repository
2. Create a feature branch
3. Submit a pull request with a clear description of changes

## License

This project is licensed under the MIT License - see the LICENSE file for details.


