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

2. `collect-monthly.sh`
   - Collects PR data for a specific month
   - Parameters:
     - `YYYY-MM`: Target month (optional, defaults to last month)
     - `UPDATE_SHEETS`: Boolean flag to update Google Sheets (optional, defaults to false)
   - Creates monthly data files in `data/monthly/`
   - Updates consolidated data files in `data/consolidated/`
   - Usage: `./collect-monthly.sh "2024-03" true`

3. `group-prs.sh`
   - Processes and groups PR data by title and status
   - Called by `collect-monthly.sh`
   - Requires `plugins.json` file for plugin information
   - Usage: `./group-prs.sh "input_file.json" "plugins.json"`

4. `retry-collection.sh`
   - Bulk data collection script with retry mechanism
   - Collects data from July 2024 onwards
   - Implements exponential backoff for failed attempts
   - Updates Google Sheets only after all data is collected
   - Usage: `./retry-collection.sh`

### Supporting Scripts

5. `upload_to_sheets.py`
   - Python script for uploading data to Google Sheets
   - Requires Google Sheets API credentials
   - Called by other scripts when `UPDATE_SHEETS` is true

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
└── scripts/         # Collection and processing scripts
\`\`\`

## Automated Workflows

### PR Stats Workflow (`pr-stats.yml`)
- **Monthly Collection** (2nd of each month)
  - Runs full data collection for the previous month
  - Updates consolidated statistics
  - Updates Google Sheets

- **Daily Updates** (midnight UTC)
  - Updates current month's data
  - Updates open and failing PR statistics
  - Updates Google Sheets with latest data

### PR Collector Test (`pr-collector-test.yml`)
- Runs every Tuesday at 07:18 UTC
- Tests the PR collector functionality
- Creates a pull request with updated statistics
- Uses Docker for isolated testing environment

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

## Maintenance Tasks

### Monthly Tasks
1. Check the automated collection ran successfully on the 2nd
2. Verify data in Google Sheets is updated
3. Review any failed collections in the GitHub Actions logs

### As-Needed Tasks
1. Review and clean up archived data
2. Verify backup integrity
3. Update dependencies as needed

## Troubleshooting

1. **Rate Limiting**
   - The scripts include built-in retry mechanisms with exponential backoff
   - Check GitHub API quota in the logs
   - Adjust collection timing if needed

2. **Failed Collections**
   - Check the logs in `data/monthly/` for specific errors
   - Use `retry-collection.sh` to retry failed periods
   - Verify GitHub token permissions

3. **Google Sheets Issues**
   - Verify API credentials are valid
   - Check Python virtual environment is activated
   - Review logs for API errors

## Contributing

1. Fork the repository
2. Create a feature branch
3. Submit a pull request with a clear description of changes

## License

[Add your license information here]

```
./count_prs.sh repos.txt 2024
./compute-stats.sh gounthar,jonesbusy 2024-12-01 2025-01-15
./group-prs.sh prs_gounthar_and_others_2024-12-01_to_2025-01-15.json plugins.json
```
