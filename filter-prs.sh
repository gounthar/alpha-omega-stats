#!/bin/bash

# Input JSON file (passed as a parameter)
INPUT_JSON="$1"
PLUGINS_JSON="$2"

# Enable debug mode
set -x

# Validate that the input JSON file exists and is readable
if [ ! -r "$INPUT_JSON" ]; then
  echo "Error: Input file '$INPUT_JSON' does not exist or is not readable" >&2
  exit 1
fi

# Validate that the plugins JSON file exists and is readable
if [ ! -r "$PLUGINS_JSON" ]; then
  echo "Error: Plugins file '$PLUGINS_JSON' does not exist or is not readable" >&2
  exit 1
fi

# Extract the list of Jenkins plugin repositories from plugins.json
PLUGIN_REPOS=$(jq -r '.plugins | to_entries[] | select(.value.scm != null) | .value.scm | sub("https://github.com/"; "")' "$PLUGINS_JSON") || {
    echo "Error: Failed to parse plugins JSON file" >&2
    exit 1
}

echo "Extracted plugin repositories:"
echo "$PLUGIN_REPOS" | head -n 5  # Show first 5 repositories

# Validate that PLUGIN_REPOS is not empty
if [ -z "$PLUGIN_REPOS" ]; then
  echo "Error: No plugin repositories were extracted from '$PLUGINS_JSON'" >&2
  exit 1
fi

# Convert the list of repository names into a JSON array for use with --argjson
PLUGIN_REPOS_JSON=$(echo "$PLUGIN_REPOS" | jq -R -s -c 'split("\n") | map(select(. != ""))')

# Debug: Show the structure of the first item in the input JSON
echo "First item structure in input JSON:"
jq '.[0] | keys' "$INPUT_JSON"

# Get total number of PRs before filtering
TOTAL_PRS=$(jq '. | length' "$INPUT_JSON")

# Filter PRs that are related to Jenkins plugins and exclude those created by Dependabot and Renovate
FILTERED_PRS=$(jq --argjson plugin_repos "$PLUGIN_REPOS_JSON" '
  def is_valid_pr:
    type == "object" and
    has("repository") and
    has("user") and
    .user != "dependabot" and
    .user != "renovate";
  
  map(select(
    is_valid_pr and
    .repository as $repo |
    $plugin_repos | index($repo) != null
  ))
' "$INPUT_JSON") || {
    echo "Error: Failed to filter PRs" >&2
    # Try to show more context around the error
    echo "Context around the error:"
    jq '.[6134:6138]' "$INPUT_JSON"
    exit 1
}

# Output the filtered PRs to a new JSON file
FILTERED_JSON="filtered_prs_$(basename "$INPUT_JSON")"
echo "$FILTERED_PRS" > "$FILTERED_JSON"

# Get number of filtered PRs
FILTERED_COUNT=$(echo "$FILTERED_PRS" | jq '. | length')

# Validate the filtered output
if [ "$FILTERED_COUNT" -eq 0 ]; then
  echo "Warning: No PRs matched the filtering criteria" >&2
fi

# Show summary
echo "Summary:"
echo "Total PRs processed: $TOTAL_PRS"
echo "PRs matching criteria: $FILTERED_COUNT"
echo "Filtered PRs have been saved to $FILTERED_JSON"

# Disable debug mode
set +x
