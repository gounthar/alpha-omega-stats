#!/bin/bash

# ./compute-stats.sh <GitHub-Handles> <Start-Date> <End-Date>
# Replace <GitHub-Handles>, <Start-Date>, and <End-Date> with the appropriate values.
# GitHub-Handles should be a comma-separated list (e.g., "user1,user2,user3").
# Dates should be in the format YYYY-MM-DD.

# Variables
IFS=',' read -r -a USERS <<< "$1"  # Split the comma-separated list of users into an array
START_DATE=$2
END_DATE=$3
ORGS=("jenkinsci" "jenkins-infra")

# Function to fetch PRs for a specific organization and user
fetch_prs_for_org() {
  local org=$1
  local user=$2
  gh pr list --state all --author "$user" --json number,title,createdAt,headRepository --search "org:$org created:$START_DATE..$END_DATE"
}

# Main script
echo "Fetching PRs for users: ${USERS[*]}"

PR_LIST="[]"
for USER in "${USERS[@]}"; do
  echo "Fetching PRs for user: $USER"
  for ORG in "${ORGS[@]}"; do
    echo "Fetching PRs for organization: $ORG"
    PRS=$(fetch_prs_for_org "$ORG" "$USER")
    # Add the user information to each PR
    PRS=$(echo "$PRS" | jq --arg user "$USER" 'map(. + {user: $user})')
    PR_LIST=$(echo "$PR_LIST" | jq --argjson prs "$PRS" --arg org "$ORG" '. + ($prs | map(.repository = "\($org)/\(.headRepository.name)"))')
  done
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
OUTPUT_FILE="prs_${USERS[0]}_and_others_${START_DATE}_to_${END_DATE}.json"
echo "$SORTED_PR_LIST" > "$OUTPUT_FILE"
echo "Sorted PRs have been saved to $OUTPUT_FILE"
