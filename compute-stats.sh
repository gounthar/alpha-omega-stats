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
  gh pr list --state all --author "$user" --json number,title,createdAt,headRepository,state --search "org:$org created:$START_DATE..$END_DATE"
}

# Function to fetch repositories with releases during the specified timeframe
fetch_repos_with_releases() {
  local start_date=$1
  local end_date=$2
  local repos=()
  # Add inclusive date comparison
  local end_date_inclusive=$(date -d "$end_date + 1 day" +%Y-%m-%d)

  for org in "${ORGS[@]}"; do
    # Declare and assign separately to handle errors
    local org_repos
    if ! org_repos=$(gh repo list "$org" --json name --jq '.[].name'); then
      echo "Error: Failed to fetch repositories for $org" >&2
      continue
    fi

    for repo in $org_repos; do
      local releases
      if ! releases=$(gh release list -R "$org/$repo" --json tagName,publishedAt \
        --jq ".[] | select(.publishedAt >= \"$start_date\" and .publishedAt < \"$end_date_inclusive\")"); then
        echo "Warning: Failed to fetch releases for $org/$repo" >&2
        continue
      fi
      if [[ -n "$releases" ]]; then
        repos+=("$org/$repo")
      fi
    done
  done
  echo "${repos[@]}" | tr ' ' '\n' | sort -u
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

# Fetch and store the list of repositories with releases during the specified timeframe
REPOS_WITH_RELEASES=$(fetch_repos_with_releases "$START_DATE" "$END_DATE")
REPOS_FILE="repos_with_releases_${START_DATE}_to_${END_DATE}.txt"
echo "$REPOS_WITH_RELEASES" > "$REPOS_FILE"
echo "Repositories with releases have been saved to $REPOS_FILE"

# Pass the list of repositories to the generate-report.sh script
./generate-report.sh "$OUTPUT_FILE" "$REPOS_FILE"
