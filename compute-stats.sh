#!/bin/bash

# ./compute-stats.sh <GitHub-Username> <Start-Date> <End-Date>
# Replace <GitHub-Username>, <Start-Date>, and <End-Date> with the appropriate values. The dates should be in the format YYYY-MM-DD.
# Variables
USER=$1
START_DATE=$2
END_DATE=$3
ORGS=("jenkinsci" "jenkins-infra")

# Function to fetch PRs for a specific organization
fetch_prs_for_org() {
  local org=$1
  gh pr list --state all --author $USER --json number,title,createdAt,headRepository --search "org:$org created:$START_DATE..$END_DATE"
}

# Main script
echo "Fetching PRs for user: $USER"

PR_LIST="[]"
for ORG in "${ORGS[@]}"; do
  echo "Fetching PRs for organization: $ORG"
  PRS=$(fetch_prs_for_org "$ORG")
  PR_LIST=$(echo "$PR_LIST" | jq --argjson prs "$PRS" --arg org "$ORG" '. + ($prs | map(.repository = "\($org)/\(.headRepository.name)"))')
done

echo "Raw PR list:"
echo "$PR_LIST"

# Filter PRs by date and ensure repository is not null
FILTERED_PR_LIST=$(echo "$PR_LIST" | jq "[.[] | select(.createdAt >= \"$START_DATE\" and .createdAt <= \"$END_DATE\" and .repository != null)]")
echo "Filtered PR list:"
echo "$FILTERED_PR_LIST"

# Sort PRs by creation date (ascending order)
SORTED_PR_LIST=$(echo "$FILTERED_PR_LIST" | jq 'sort_by(.createdAt)')
echo "Sorted PR list (by createdAt):"
echo "$SORTED_PR_LIST"

# Save the sorted PR list to a JSON file
OUTPUT_FILE="prs_${USER}_${START_DATE}_to_${END_DATE}.json"
echo "$SORTED_PR_LIST" > "$OUTPUT_FILE"
echo "Sorted PRs have been saved to $OUTPUT_FILE"
