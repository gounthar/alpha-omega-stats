#!/bin/bash

# Check if the correct number of arguments are provided
if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <repositories_file> <year>"
    exit 1
fi

REPOS_FILE=$1
YEAR=$2

# Check if the file exists
if [ ! -f "$REPOS_FILE" ]; then
    echo "File $REPOS_FILE does not exist."
    exit 1
fi

# Initialize total count
TOTAL_PR_COUNT=0

# Read repositories from the file
while IFS= read -r repo; do
    echo "Processing repository: $repo"

    # Initialize PR count for the current repository
    REPO_PR_COUNT=0

    # Fetch merged PRs for the given year, handling pagination
    PAGE=1
    while true; do
        # Fetch PRs for the current page using gh api
        PRS=$(gh api -X GET "repos/$repo/pulls?state=closed&sort=updated&direction=desc&per_page=100&page=$PAGE" --jq '.[] | select(.merged_at != null) | select(.merged_at | startswith("'"$YEAR"'"))')

        # Check if there are no more PRs
        if [ -z "$PRS" ]; then
            break
        fi

        # Count the number of PRs on the current page
        COUNT=$(echo "$PRS" | jq -s length)
        REPO_PR_COUNT=$((REPO_PR_COUNT + COUNT))

        # Increment the page number
        PAGE=$((PAGE + 1))
    done

    echo "Repository $repo has $REPO_PR_COUNT PRs merged in $YEAR."
    TOTAL_PR_COUNT=$((TOTAL_PR_COUNT + REPO_PR_COUNT))

done < "$REPOS_FILE"

echo "Total PRs merged across all repositories in $YEAR: $TOTAL_PR_COUNT"
