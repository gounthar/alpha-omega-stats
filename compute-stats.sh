#!/bin/bash

# ./compute-stats.sh <GitHub-Handles> <Start-Date> <End-Date>
# Replace <GitHub-Handles>, <Start-Date>, and <End-Date> with the appropriate values.
# GitHub-Handles should be a comma-separated list (e.g., "user1,user2,user3").
# Dates should be in the format YYYY-MM-DD.

# Variables
IFS=',' read -r -a USERS <<< "$1"  # Split the comma-separated list of users into an array
START_DATE=$2
END_DATE=$3
ORGS=("jenkinsci" "jenkins-infra" "jenkins-docs")

# Function to fetch repositories with releases during the specified timeframe
fetch_repos_with_releases() {
  local start_date=$1
  local end_date=$2

  # Validate input parameters
  if [[ ! $start_date =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || \
     [[ ! $end_date =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
    echo "Error: Invalid date format. Expected YYYY-MM-DD" >&2
    return 1
  fi

  local repos=()
  # Add inclusive date comparison
  local end_date_inclusive
  end_date_inclusive=$(date -d "$end_date + 1 day" +%Y-%m-%d) || {
    echo "Error: Failed to process end date" >&2
    return 1
  }
}

# Function to check rate limit
check_rate_limit() {
  local remaining
  remaining=$(gh api rate_limit --jq '.rate.remaining') || return 1
  if ((remaining < 100)); then
    echo "Warning: GitHub API rate limit is low ($remaining remaining). Waiting..." >&2
    sleep 60
  fi
}

# Function to fetch PRs for a specific organization and user
fetch_prs_for_org() {
  local org=$1
  local user=$2
  local date=$3
  gh pr list --state all --author "$user" --json number,title,createdAt,updatedAt,headRepository,state --search "org:$org created:$date..$date updated:$date..$date"
}

# Function to generate a list of dates between start and end dates
generate_date_range() {
  local start_date=$1
  local end_date=$2
  local current_date="$start_date"

  while [[ "$current_date" < "$end_date" ]] || [[ "$current_date" == "$end_date" ]]; do
    echo "$current_date"
    current_date=$(date -I -d "$current_date + 1 day")
  done
}

# Main script
echo "Fetching PRs for users: ${USERS[*]}"

PR_LIST="[]"
for USER in "${USERS[@]}"; do
  echo "Fetching PRs for user: $USER"
  for ORG in "${ORGS[@]}"; do
    echo "Fetching PRs for organization: $ORG"
    for DATE in $(generate_date_range "$START_DATE" "$END_DATE"); do
      echo "Fetching PRs for date: $DATE"
      # Check rate limit before making API calls
      check_rate_limit || continue
      PRS=$(fetch_prs_for_org "$ORG" "$USER" "$DATE")
      # Add the user information to each PR
      PRS=$(echo "$PRS" | jq --arg user "$USER" 'map(. + {user: $user})')
      PR_LIST=$(echo "$PR_LIST" | jq --argjson prs "$PRS" --arg org "$ORG" '. + ($prs | map(.repository = "\($org)/\(.headRepository.name)"))')
    done
  done
done

echo "Raw PR list:"
echo "$PR_LIST"

# Filter PRs by date and ensure repository is not null
FILTERED_PR_LIST=$(echo "$PR_LIST" | jq "[.[] | select((.createdAt >= \"$START_DATE\" and .createdAt <= \"$END_DATE\" and .repository != null) or (.updatedAt >= \"$START_DATE\" and .updatedAt <= \"$END_DATE\" and .repository != null))]")
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

# Clean up temporary files on exit
cleanup() {
  local exit_code=$?
  rm -f "$REPOS_FILE"
  exit $exit_code
}
trap cleanup EXIT

# Fetch and store the list of repositories with releases during the specified timeframe
if ! REPOS_WITH_RELEASES=$(fetch_repos_with_releases "$START_DATE" "$END_DATE" "$OUTPUT_FILE"); then
  echo "Error: Failed to fetch repositories with releases" >&2
  exit 1
fi

REPOS_FILE="repos_with_releases_${START_DATE}_to_${END_DATE}.txt"
echo "$REPOS_WITH_RELEASES" > "$REPOS_FILE"
echo "Repositories with releases have been saved to $REPOS_FILE"

# Pass the list of repositories to the generate-report.sh script
if ! ./generate-report.sh "$OUTPUT_FILE" "$REPOS_FILE"; then
  echo "Error: Failed to generate report" >&2
  exit 1
fi
