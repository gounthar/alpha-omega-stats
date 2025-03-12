#!/bin/bash

# Exit on error
set -e

echo "Starting daily PR status update..."

# Check if consolidated files exist
if [ ! -f "data/consolidated/all_prs.json" ]; then
    echo "Error: data/consolidated/all_prs.json not found. Run collect-monthly.sh first."
    exit 1
fi

# Create a temporary file for updated PRs
TEMP_FILE=$(mktemp)
trap 'rm -f "$TEMP_FILE"' EXIT

# Read open PRs and update their status using the GitHub API
echo "Updating PR statuses..."
jq -r '.[] | select(.state == "OPEN") | "\(.repository) \(.number)"' "data/consolidated/open_prs.json" | \
while read -r repo number; do
    echo "Checking $repo PR #$number..."
    # Use GitHub CLI to get current PR status
    PR_INFO=$(gh pr view "$number" --repo "$repo" --json state,statusCheckRollup,title,url)
    if [ $? -eq 0 ]; then
        echo "$PR_INFO" >> "$TEMP_FILE"
    else
        echo "Warning: Failed to get status for $repo PR #$number"
    fi
done

# Update the consolidated files with new status information
if [ -s "$TEMP_FILE" ]; then
    echo "Updating consolidated files with new status information..."
    
    # Update all_prs.json with new status information
    jq -s '
        reduce (.[]) as $item (inputs[0];
            map(if .url == $item.url then
                . + {
                    state: $item.state,
                    checkStatus: ($item.statusCheckRollup[0].state // "UNKNOWN")
                }
                else . end)
        )
    ' "data/consolidated/all_prs.json" "$TEMP_FILE" > "data/consolidated/all_prs.json.tmp" && \
    mv "data/consolidated/all_prs.json.tmp" "data/consolidated/all_prs.json"

    # Update open_prs.json
    echo "Updating open_prs.json..."
    jq '[.[] | select(.state == "OPEN")]' "data/consolidated/all_prs.json" > \
        "data/consolidated/open_prs.json"

    # Update failing_prs.json
    echo "Updating failing_prs.json..."
    jq '[.[] | select(.state == "OPEN" and .checkStatus == "FAILURE")]' \
        "data/consolidated/all_prs.json" > "data/consolidated/failing_prs.json"
fi

# Update Google Sheets with the latest data
echo "Updating Google Sheets..."
python3 upload_to_sheets.py "data/consolidated/all_prs.json" false

echo "Daily update completed successfully!"
echo "Updated files:"
echo "  - data/consolidated/all_prs.json"
echo "  - data/consolidated/open_prs.json"
echo "  - data/consolidated/failing_prs.json"
echo "Google Sheets dashboard has been updated." 