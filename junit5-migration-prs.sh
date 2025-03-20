#!/bin/bash

# Exit on error
set -e

# Enable debug output if needed
# set -x

echo "Collecting JUnit 5 migration PRs..."

# Create output directory if it doesn't exist
mkdir -p data/junit5

# Create a temporary file
TEMP_FILE=$(mktemp)

# Clean up on exit
trap 'rm -f "$TEMP_FILE"' EXIT

# Check if PR URLs file exists
PR_URLS_FILE="junit5_pr_urls.txt"
if [ ! -f "$PR_URLS_FILE" ]; then
    # Create default file with example PRs
    cat > "$PR_URLS_FILE" << EOF
https://github.com/jenkinsci/lockable-resources-plugin/pull/761
https://github.com/jenkinsci/pipeline-as-yaml-plugin/pull/127
https://github.com/jenkinsci/build-token-root-plugin/pull/185
https://github.com/jenkinsci/javadoc-plugin/pull/186
https://github.com/jenkinsci/ws-cleanup-plugin/pull/230
https://github.com/jenkinsci/locale-plugin/pull/290
https://github.com/jenkinsci/file-leak-detector-plugin/pull/142
EOF
    echo "Created default PR URLs file: $PR_URLS_FILE"
fi

# Function to extract repo and PR number from GitHub URL
extract_repo_pr() {
    local url="$1"
    local repo=$(echo "$url" | sed -E 's|https://github.com/([^/]+/[^/]+)/pull/([0-9]+).*|\1|')
    local pr_number=$(echo "$url" | sed -E 's|https://github.com/([^/]+/[^/]+)/pull/([0-9]+).*|\2|')
    echo "$repo $pr_number"
}

# Initialize the temp file with an empty array
echo "" > "$TEMP_FILE"

# Process each PR URL
echo "Processing JUnit 5 migration PRs from $PR_URLS_FILE..."
while IFS= read -r url || [ -n "$url" ]; do
    # Skip empty lines and comments
    [[ -z "$url" || "$url" =~ ^# ]] && continue
    
    # Extract repo and PR number
    repo_pr=$(extract_repo_pr "$url")
    repo=$(echo "$repo_pr" | cut -d' ' -f1)
    pr_number=$(echo "$repo_pr" | cut -d' ' -f2)
    
    echo "Processing $repo PR #$pr_number..."
    
    # Use GitHub CLI to get PR info
    PR_INFO=$(gh pr view "$pr_number" --repo "$repo" --json state,statusCheckRollup,title,url,author,body,labels)
    gh_status=$?
    
    if [ $gh_status -eq 0 ]; then
        # Transform the PR info to match our expected format
        jq --arg repo "$repo" --arg number "$pr_number" '{
            number: ($number | tonumber),
            title: .title,
            state: .state,
            user: .author.login,
            repository: $repo,
            pluginName: ($repo | split("/") | .[1] | sub("-plugin$"; "")),
            labels: [.labels[].name],
            url: .url,
            description: .body,
            checkStatus: (if .statusCheckRollup and .statusCheckRollup[0] then .statusCheckRollup[0].state else "UNKNOWN" end)
        }' <<< "$PR_INFO" >> "$TEMP_FILE"
        echo "  Successfully processed PR info"
    else
        echo "  Failed to get PR info for $url"
    fi
done < "$PR_URLS_FILE"

# Convert the file to a proper JSON array
jq -s '.' "$TEMP_FILE" > "data/junit5/junit5_migration_prs.json"

# Verify we have valid JSON
if ! jq '.' "data/junit5/junit5_migration_prs.json" > /dev/null 2>&1; then
    echo "Error: Failed to create valid JSON. Check the GitHub CLI output."
    exit 1
fi

# Count and display stats
TOTAL_PRS=$(jq 'length' "data/junit5/junit5_migration_prs.json")

# Check if we have any PRs before trying to analyze them
if [ "$TOTAL_PRS" -eq 0 ]; then
    echo "No JUnit 5 migration PRs were successfully processed. Please check GitHub CLI access."
    exit 0
fi

echo "Found $TOTAL_PRS JUnit 5 migration PRs:"
OPEN_PRS=$(jq '[.[] | select(.state == "OPEN")] | length' "data/junit5/junit5_migration_prs.json")
MERGED_PRS=$(jq '[.[] | select(.state == "MERGED")] | length' "data/junit5/junit5_migration_prs.json")
CLOSED_PRS=$(jq '[.[] | select(.state == "CLOSED")] | length' "data/junit5/junit5_migration_prs.json")

echo "  - Open: $OPEN_PRS"
echo "  - Merged: $MERGED_PRS"
echo "  - Closed: $CLOSED_PRS"

# Group PRs by repository
echo "Grouping JUnit 5 migration PRs by repository..."
jq 'group_by(.repository) | map({repository: .[0].repository, count: length, prs: .}) | sort_by(-.count)' \
    "data/junit5/junit5_migration_prs.json" > "data/junit5/junit5_prs_by_repo.json"

# Group PRs by user
echo "Grouping JUnit 5 migration PRs by user..."
jq 'group_by(.user) | map({user: .[0].user, count: length, prs: .}) | sort_by(-.count)' \
    "data/junit5/junit5_migration_prs.json" > "data/junit5/junit5_prs_by_user.json"

# Group PRs by state
echo "Grouping JUnit 5 migration PRs by state..."
jq 'group_by(.state) | map({state: .[0].state, count: length, prs: .}) | sort_by(-.count)' \
    "data/junit5/junit5_migration_prs.json" > "data/junit5/junit5_prs_by_state.json"

# Create a summary file with key information
echo "Creating summary file..."
jq '[.[] | {
    repository: .repository,
    number: .number,
    title: .title,
    url: .url,
    state: .state,
    user: .user,
    checkStatus: .checkStatus
}]' "data/junit5/junit5_migration_prs.json" > "data/junit5/junit5_summary.json"

# Define common patterns for JUnit 5 migration PRs based on our analysis
# This will be useful for future automatic detection
cat > "data/junit5/detection_patterns.txt" << EOF
# JUnit 5 Migration PR Detection Patterns
# Generated on $(date)

## Title Patterns
(?i)migrate tests? to junit ?5
(?i)junit ?5
(?i)migrate to junit
(?i)junit.*(4|four).*(5|five)
(?i)junit.*(migration|upgrade)
(?i)openrewrite.*junit

## Description Patterns
(?i)migrate (all )?tests? to junit ?5
(?i)junit ?5
(?i)migrate to junit
(?i)junit.*(4|four).*(5|five)
(?i)junit.*(migration|upgrade)
(?i)openrewrite.*junit
(?i)org\.junit\.jupiter
(?i)junit-jupiter

## Label Patterns
junit5
junit-5
junit-migration
openrewrite
tests?
test-migration

## Common Authors
strangelookingnerd
EOF

echo "JUnit 5 migration PR analysis completed successfully!"
echo "Results saved to data/junit5/ directory"
