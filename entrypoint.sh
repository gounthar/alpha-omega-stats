#!/usr/bin/env bash

# Enable strict error handling
set -euo pipefail

# Validate required environment variables

# Check if GITHUB_TOKEN is set, if not, print an error message and exit
if [ -z "$GITHUB_TOKEN" ]; then
    echo "Error: GITHUB_TOKEN environment variable is required"
    exit 1
fi

# Check if START_DATE is set, if not, print an error message and exit
if [ -z "$START_DATE" ]; then
    echo "Error: START_DATE environment variable is required (format: YYYY-MM-DD)"
    echo "Note: While a default is set in the Dockerfile (2024-08-01), you can override it by setting START_DATE"
    exit 1
fi

# Validate START_DATE format

# Check if START_DATE matches the YYYY-MM-DD format, if not, print an error message and exit
if ! echo "$START_DATE" | grep -Eq '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'; then
    echo "Error: START_DATE must be in YYYY-MM-DD format"
    echo "Current value: $START_DATE"
    exit 1
fi

# Validate required files exist and are writable
for file in "plugins.json" "report.json" "found_prs.json" "jenkins_prs.json"; do
    if [ ! -f "$file" ]; then
        echo "Error: Required file '$file' does not exist or is not a regular file"
        exit 1
    fi
done

# Set end date to current date
END_DATE=$(date +%Y-%m-%d)

# Print the date range being used
echo "Running jenkins-pr-collector with date range: $START_DATE to $END_DATE"

# Execute the jenkins-pr-collector command with the provided GitHub token and date range
./jenkins-pr-collector \
    -token "$GITHUB_TOKEN" \
    -start "$START_DATE" \
    -end "$END_DATE" \
    -found-prs "found_prs.json" \
    -output "jenkins_prs.json"
