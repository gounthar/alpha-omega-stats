#!/bin/bash

# Enable debug output
# set -x

# Check arguments
if [ $# -ne 2 ]; then
    echo "Usage: $0 <pr_list_file> <output_file>"
    exit 1
fi

PR_LIST_FILE="$1"
OUTPUT_FILE="$2"

# Verify PR list was created and has content
if [ ! -s "$PR_LIST_FILE" ]; then
    echo "Error: PR list file is empty or does not exist"
    exit 1
fi

# Count PRs to process
PR_COUNT=$(wc -l < "$PR_LIST_FILE")
echo "Found $PR_COUNT PRs to process"

# Process each PR
total_prs=$PR_COUNT
current_pr=0

# Process each PR
i=1
while [ $i -le $PR_COUNT ]; do
    # Get the i-th line from the PR list file
    LINE=$(sed -n "${i}p" "$PR_LIST_FILE")
    
    # Extract repo and number
    repo=$(echo "$LINE" | cut -d' ' -f1)
    number=$(echo "$LINE" | cut -d' ' -f2)
    
    ((current_pr++))
    echo "[$current_pr/$total_prs] Checking $repo PR #$number..."
    
    # Use GitHub CLI to get current PR status
    echo "Running: gh pr view $number --repo $repo --json state,statusCheckRollup,title,url"
    PR_INFO=$(gh pr view "$number" --repo "$repo" --json state,statusCheckRollup,title,url)
    gh_status=$?
    echo "GitHub CLI exit status: $gh_status"
    
    if [ $gh_status -eq 0 ]; then
        # Validate JSON structure before appending
        if echo "$PR_INFO" | jq -e '.state != null and .url != null and .title != null' > /dev/null; then
            echo "$PR_INFO" >> "$OUTPUT_FILE"
            echo "  ✓ Successfully updated PR info"
        else
            echo "  ✗ Warning: Invalid JSON structure for $repo PR #$number, skipping"
        fi
    else
        echo "  ✗ Warning: Failed to get status for $repo PR #$number"
    fi
    
    # Increment counter manually
    i=$((i+1))
    echo "Processed PR $i of $PR_COUNT"
done

echo "PR processing complete!"
exit 0
