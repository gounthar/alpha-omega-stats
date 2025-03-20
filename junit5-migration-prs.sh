#!/bin/bash

# Exit on error
set -e

# Enable debug output if needed
# set -x

echo "Extracting JUnit 5 migration PRs..."

# Check if consolidated files exist
if [ ! -f "data/consolidated/all_prs.json" ]; then
    echo "Error: data/consolidated/all_prs.json not found. Run collect-monthly.sh first."
    exit 1
fi

# Create output directory if it doesn't exist
mkdir -p data/junit5

# Create a temporary file
TEMP_FILE=$(mktemp)

# Clean up on exit
trap 'rm -f "$TEMP_FILE"' EXIT

# Define common patterns for JUnit 5 migration PRs
TITLE_PATTERNS="(?i)junit ?5|(?i)migrate to junit|(?i)junit.*(4|four).*(5|five)|(?i)junit.*(migration|upgrade)|(?i)openrewrite.*junit"
DESC_PATTERNS="(?i)junit ?5|(?i)migrate to junit|(?i)junit.*(4|four).*(5|five)|(?i)junit.*(migration|upgrade)|(?i)openrewrite.*junit|(?i)org\.junit\.jupiter|(?i)junit-jupiter"
LABEL_PATTERNS="junit5|junit-5|junit-migration|openrewrite"

# Filter PRs related to JUnit 5 migration
# Look for patterns in title, description, or labels
# Handle null values safely
jq --arg title_patterns "$TITLE_PATTERNS" \
   --arg desc_patterns "$DESC_PATTERNS" \
   --arg label_patterns "$LABEL_PATTERNS" \
   '[.[] | select(
     ((.title | type == "string") and (.title | test($title_patterns))) or
     ((.description | type == "string") and (.description | test($desc_patterns))) or
     ((.labels | type == "array") and (.labels | any(. | test($label_patterns)))) or
     ((.user | type == "string") and (.user == "strangelookingnerd"))
   )]' "data/consolidated/all_prs.json" > "$TEMP_FILE"

# Verify the JSON array has content
if [ ! -s "$TEMP_FILE" ] || ! jq -e '. | length > 0' "$TEMP_FILE" > /dev/null; then
    echo "No JUnit 5 migration PRs found."
    echo "[]" > "data/junit5/junit5_migration_prs.json"
    exit 0
fi

# Save the filtered PRs
mv "$TEMP_FILE" "data/junit5/junit5_migration_prs.json"

# Count and display stats
TOTAL_PRS=$(jq '. | length' "data/junit5/junit5_migration_prs.json")
OPEN_PRS=$(jq '[.[] | select(.state == "OPEN")] | length' "data/junit5/junit5_migration_prs.json")
MERGED_PRS=$(jq '[.[] | select(.state == "MERGED")] | length' "data/junit5/junit5_migration_prs.json")
CLOSED_PRS=$(jq '[.[] | select(.state == "CLOSED")] | length' "data/junit5/junit5_migration_prs.json")

echo "Found $TOTAL_PRS JUnit 5 migration PRs:"
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

# Group PRs by check status
echo "Grouping JUnit 5 migration PRs by check status..."
jq 'group_by(.checkStatus) | map({status: .[0].checkStatus, count: length, prs: .}) | sort_by(-.count)' \
    "data/junit5/junit5_migration_prs.json" > "data/junit5/junit5_prs_by_status.json"

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

echo "JUnit 5 migration PR analysis completed successfully!"
echo "Results saved to data/junit5/ directory"
