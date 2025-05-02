#!/bin/bash

# Exit on error
set -e

# Enable debug output if needed
# set -x

echo "Analyzing specific JUnit 5 migration PRs..."

# Create output directory if it doesn't exist
mkdir -p data/junit5

# Create a temporary file for PR data
TEMP_FILE=$(mktemp)

# Clean up on exit
trap 'rm -f "$TEMP_FILE"' EXIT

# Function to extract repo and PR number from GitHub URL
extract_repo_pr() {
    local url="$1"
    local repo=$(echo "$url" | sed -E 's|https://github.com/([^/]+/[^/]+)/pull/([0-9]+).*|\1|')
    local pr_number=$(echo "$url" | sed -E 's|https://github.com/([^/]+/[^/]+)/pull/([0-9]+).*|\2|')
    echo "$repo $pr_number"
}

# Process each PR URL
process_pr() {
    local url="$1"
    local repo_pr=$(extract_repo_pr "$url")
    local repo=$(echo "$repo_pr" | cut -d' ' -f1)
    local pr_number=$(echo "$repo_pr" | cut -d' ' -f2)
    
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
        echo "  Failed to get PR info"
    fi
}

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

# Initialize the temp file with an empty array
echo "" > "$TEMP_FILE"

# Process each PR URL
echo "Processing PRs from $PR_URLS_FILE..."
while IFS= read -r url || [ -n "$url" ]; do
    # Skip empty lines and comments
    [[ -z "$url" || "$url" =~ ^# ]] && continue
    process_pr "$url"
done < "$PR_URLS_FILE"

# Convert the file to a proper JSON array
jq -s '.' "$TEMP_FILE" > "data/junit5/junit5_specific_prs.json"

# Verify we have valid JSON
if ! jq '.' "data/junit5/junit5_specific_prs.json" > /dev/null 2>&1; then
    echo "Error: Failed to create valid JSON. Check the GitHub CLI output."
    exit 1
fi

# Count and display stats
TOTAL_PRS=$(jq 'length' "data/junit5/junit5_specific_prs.json")
echo "Found $TOTAL_PRS JUnit 5 migration PRs:"

# Check if we have any PRs before trying to analyze them
if [ "$TOTAL_PRS" -eq 0 ]; then
    echo "No PRs were successfully processed. Please check GitHub CLI access."
    exit 0
fi

# Now we can safely analyze the PRs
OPEN_PRS=$(jq '[.[] | select(.state == "OPEN")] | length' "data/junit5/junit5_specific_prs.json")
MERGED_PRS=$(jq '[.[] | select(.state == "MERGED")] | length' "data/junit5/junit5_specific_prs.json")
CLOSED_PRS=$(jq '[.[] | select(.state == "CLOSED")] | length' "data/junit5/junit5_specific_prs.json")

echo "  - Open: $OPEN_PRS"
echo "  - Merged: $MERGED_PRS"
echo "  - Closed: $CLOSED_PRS"

# Extract common patterns from these PRs to improve our detection
echo "Analyzing PRs for common patterns..."

# Create a patterns file to store our findings
PATTERNS_FILE="data/junit5/common_patterns.txt"
{
    echo "# Common patterns found in JUnit 5 migration PRs"
    echo "# Generated on $(date)"
    echo ""
    
    # Extract common words from titles
    echo "## Common words in titles:"
    jq -r '.[] | .title' "data/junit5/junit5_specific_prs.json" | \
        tr '[:upper:]' '[:lower:]' | \
        tr -cs '[:alnum:]' '\n' | \
        grep -v '^$' | \
        sort | uniq -c | sort -nr | head -10 >> "$PATTERNS_FILE"
    
    echo "" >> "$PATTERNS_FILE"
    
    # Extract common words from descriptions
    echo "## Common words in descriptions:" >> "$PATTERNS_FILE"
    jq -r '.[] | .description' "data/junit5/junit5_specific_prs.json" | \
        tr '[:upper:]' '[:lower:]' | \
        tr -cs '[:alnum:]' '\n' | \
        grep -v '^$' | \
        sort | uniq -c | sort -nr | head -20 >> "$PATTERNS_FILE"
    
    echo "" >> "$PATTERNS_FILE"
    
    # Extract common labels
    echo "## Common labels:" >> "$PATTERNS_FILE"
    jq -r '.[] | .labels[]?' "data/junit5/junit5_specific_prs.json" | \
        grep -v '^$' | \
        sort | uniq -c | sort -nr >> "$PATTERNS_FILE"
    
    echo "" >> "$PATTERNS_FILE"
    
    # Extract common authors
    echo "## Common authors:" >> "$PATTERNS_FILE"
    jq -r '.[] | .user' "data/junit5/junit5_specific_prs.json" | \
        grep -v '^$' | \
        sort | uniq -c | sort -nr >> "$PATTERNS_FILE"
} > "$PATTERNS_FILE" 2>/dev/null || true

echo "Analysis completed. Results saved to data/junit5/junit5_specific_prs.json"
echo "Common patterns saved to $PATTERNS_FILE"
echo "You can add more PR URLs to $PR_URLS_FILE and run this script again to analyze more PRs."
