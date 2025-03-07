# Jenkins Plugin PR Collector

This tool efficiently collects pull request data from Jenkins plugin repositories within the jenkinsci GitHub organization. It uses the official Jenkins update center to identify plugin repositories, ensuring accurate data collection.

## Features

- Uses the official Jenkins update center to identify plugin repositories
- Maps pull requests to their corresponding plugins
- Collects comprehensive pull request data within a specified date range
- Handles GitHub API rate limits gracefully with built-in rate-limiting
- Concurrent processing for faster data collection
- Outputs structured JSON data for further analysis

## Requirements

- Go 1.23.2 or higher
- GitHub personal access token with `repo` scope
- Internet access to fetch the Jenkins update center data
- Dependencies:
  - github.com/google/go-github/v47/github
  - golang.org/x/oauth2
  - golang.org/x/time/rate

## Installation

1. Clone this repository
2. Install dependencies:
   ```bash
   go mod tidy
   ```

## Usage

```bash
./jenkins-pr-collector -token <github-token> -start 2024-12-01 -end 2025-01-31 -output report.json
```

### Command-line Arguments

- `-token`: GitHub API token (or set GITHUB_TOKEN environment variable)
- `-start`: Start date in YYYY-MM-DD format
- `-end`: End date in YYYY-MM-DD format (inclusive)
- `-output`: Output JSON file name (default: jenkins_prs.json)
- `-update-center`: Jenkins update center URL (default: <https://updates.jenkins.io/current/update-center.actual.json>)


### Example

```bash
# Set GitHub token as environment variable
export GITHUB_TOKEN=ghp_your_token_here

# Run the collector for January 2023
./jenkins-pr-collector -start 2023-01-01 -end 2023-01-31 -output jan_2023_prs.json
```

## Output Format

The tool generates a JSON file containing an array of pull request objects with the following structure:

```json
[
  {
    "number": 123,
    "title": "Add new feature",
    "state": "closed",
    "createdAt": "2023-01-15T10:30:45Z",
    "updatedAt": "2023-01-16T14:20:12Z",
    "user": "username",
    "repository": "jenkinsci/example-plugin",
    "pluginName": "example-plugin",
    "labels": ["enhancement", "ready-for-review"],
    "url": "https://github.com/jenkinsci/example-plugin/pull/123",
    "description": "This PR implements..."
  },
  ...
]
```

## How It Works

The tool follows these steps to collect pull request data:

1. **Fetch plugin information**: The tool retrieves the official plugin list from the Jenkins update center, which includes the repository URLs for all plugins.

2. **Parse plugin repositories**: From the update center data, the tool extracts repository names and creates a mapping between repository names and plugin information.

3. **Search for pull requests**: Using GitHub's Search API, the tool finds all pull requests within the given date range.

4. **Filter for plugin repositories**: Each pull request is checked against the list of known plugin repositories, and only those in plugin repositories are processed.

5. **Collect pull request details**: The tool gathers comprehensive information about each pull request, including labels and description.

6. **Output results**: The collected data is written to a JSON file for further analysis.

## Rate Limiting
The tool implements a conservative rate-limiting strategy to avoid hitting GitHub's API rate limits.
By default, it makes at most one request per second, which is well below GitHub's limit of 5,000 requests per hour for authenticated users.

## Extending the Tool

To add new features or modify the tool:

1. Add additional filters for specific types of pull requests by examining PR titles, descriptions, or labels
2. Implement analysis of the collected data, such as categorizing PRs by type or identifying trends
3. Add reporting features to generate HTML or Markdown reports from the collected data
4. Include PR metrics such as size, time to merge, or number of comments
