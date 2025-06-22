#!/bin/bash

# Exit on error
set -e

# Enable debug output if needed
# set -x

echo "Searching for JUnit 5 migration PRs in jenkinsci organization..."

# Create output directory if it doesn't exist
mkdir -p data/junit5

# Create a temporary file for search results
TEMP_FILE=$(mktemp)

# Clean up on exit
trap 'rm -f "$TEMP_FILE"' EXIT

# Define search terms based on our pattern analysis
SEARCH_TERMS=(
    "junit5"
    "junit 5"
    "migrate tests to junit"
    "junit jupiter"
    "openrewrite junit"
)

# Output file for PR URLs
OUTPUT_FILE="junit5_candidate_prs.txt"
echo "# JUnit 5 migration PR candidates found on $(date)" > "$OUTPUT_FILE"
echo "# Add relevant URLs to junit5_pr_urls.txt after verification" >> "$OUTPUT_FILE"
echo "" >> "$OUTPUT_FILE"

# Search for PRs using GitHub CLI
for term in "${SEARCH_TERMS[@]}"; do
    echo "Searching for: $term"
    
    # Search in PR titles
    gh search prs --owner jenkinsci --limit 100 --json url,title,state,repository "in:title $term" | \
    jq -r '.[] | "# " + .repository.name + " - " + .title + " (" + .state + ")\n" + .url' >> "$TEMP_FILE" || echo "No results for 'in:title $term'"
    
    # Search in PR bodies
    gh search prs --owner jenkinsci --limit 100 --json url,title,state,repository "in:body $term" | \
    jq -r '.[] | "# " + .repository.name + " - " + .title + " (" + .state + ")\n" + .url' >> "$TEMP_FILE" || echo "No results for 'in:body $term'"
done

# Search for PRs by specific authors known for JUnit 5 migrations
echo "Searching for PRs by known JUnit 5 migration authors..."
gh search prs --owner jenkinsci --limit 100 --json url,title,state,repository,author "author:strangelookingnerd" | \
jq -r '.[] | "# " + .repository.name + " - " + .title + " (" + .state + ") by " + .author.login + "\n" + .url' >> "$TEMP_FILE" || echo "No results for author:strangelookingnerd"

# Remove duplicates and sort
sort -u "$TEMP_FILE" >> "$OUTPUT_FILE"

# Count results
PR_COUNT=$(grep -c "^https://" "$OUTPUT_FILE" || true)

echo "Found $PR_COUNT potential JUnit 5 migration PR candidates"
echo "Results saved to $OUTPUT_FILE"
echo "Please review the results and add relevant URLs to junit5_pr_urls.txt"

# Compare with existing PRs
echo "Checking for new PRs not already in junit5_pr_urls.txt..."
if [ -f "junit5_pr_urls.txt" ]; then
    NEW_PRS=$(grep -v -f junit5_pr_urls.txt "$OUTPUT_FILE" | grep "^https://" || true)
    NEW_PR_COUNT=$(echo "$NEW_PRS" | grep -c "^https://" || true)
    
    echo "Found $NEW_PR_COUNT new PR candidates not already in junit5_pr_urls.txt"
    
    if [ "$NEW_PR_COUNT" -gt 0 ]; then
        echo "New PR candidates:"
        echo "$NEW_PRS" | while read -r url; do
            # Get the line before the URL which contains the comment
            comment=$(grep -B 1 "^$url$" "$OUTPUT_FILE" | head -n 1)
            echo "$comment"
            echo "$url"
            echo ""
        done
    fi
fi

# Alternative approach: Use the existing consolidated PR data
echo "Searching in consolidated PR data..."
if [ -f "data/consolidated/all_prs.json" ]; then
    echo "Searching consolidated PR data for JUnit 5 related PRs..."
    
    # Define common patterns for JUnit 5 migration PRs based on our analysis
    TITLE_PATTERNS="(?i)migrate tests? to junit ?5|(?i)junit ?5|(?i)migrate to junit|(?i)junit.*(4|four).*(5|five)|(?i)junit.*(migration|upgrade)|(?i)openrewrite.*junit"
    DESC_PATTERNS="(?i)migrate (all )?tests? to junit ?5|(?i)junit ?5|(?i)migrate to junit|(?i)junit.*(4|four).*(5|five)|(?i)junit.*(migration|upgrade)|(?i)openrewrite.*junit|(?i)org\.junit\.jupiter|(?i)junit-jupiter"
    LABEL_PATTERNS="junit5|junit-5|junit-migration|openrewrite|tests?|test-migration"
    AUTHOR_PATTERNS="strangelookingnerd"
    
    # Filter PRs related to JUnit 5 migration from consolidated data
    jq --arg title_patterns "$TITLE_PATTERNS" \
       --arg desc_patterns "$DESC_PATTERNS" \
       --arg label_patterns "$LABEL_PATTERNS" \
       --arg author_patterns "$AUTHOR_PATTERNS" \
       '[.[] | select(
         ((.title | type == "string") and (.title | test($title_patterns))) or
         ((.description | type == "string") and (.description | test($desc_patterns))) or
         ((.labels | type == "array") and (.labels | any(. | test($label_patterns)))) or
         ((.user | type == "string") and (.user | test($author_patterns)))
       )]' "data/consolidated/all_prs.json" > "data/junit5/junit5_candidates_from_consolidated.json"
    
    # Extract URLs and add to candidates file
    CONSOLIDATED_COUNT=$(jq 'length' "data/junit5/junit5_candidates_from_consolidated.json")
    echo "Found $CONSOLIDATED_COUNT potential candidates in consolidated data"
    
    if [ "$CONSOLIDATED_COUNT" -gt 0 ]; then
        echo "" >> "$OUTPUT_FILE"
        echo "# Candidates from consolidated PR data:" >> "$OUTPUT_FILE"
        jq -r '.[] | "# " + .repository + " - " + .title + " (" + .state + ")\n" + .url' "data/junit5/junit5_candidates_from_consolidated.json" >> "$OUTPUT_FILE"
        
        # Update total count
        PR_COUNT=$(grep -c "^https://" "$OUTPUT_FILE" || true)
        echo "Total candidates after searching consolidated data: $PR_COUNT"
    fi
else
    echo "Consolidated PR data not found. Skipping this search method."
fi

echo "Script completed successfully!"
