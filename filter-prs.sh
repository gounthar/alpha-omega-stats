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

# Debug: Show the structure of the input JSON
echo "Input JSON structure:"
jq 'type' "$INPUT_JSON"

# Get total number of PRs before filtering
TOTAL_PRS=$(jq '. | if type == "array" then length else 0 end' "$INPUT_JSON")
echo "Total PRs before filtering: $TOTAL_PRS"

# Filter PRs that are related to Jenkins plugins and exclude those created by Dependabot and Renovate
FILTERED_PRS=$(jq --argjson plugin_repos "$PLUGIN_REPOS_JSON" '
  # Handle null or empty input by returning an empty array
  if . == null then
    []
  else
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
  end' "$INPUT_JSON")

# Save filtered PRs to a new file
FILTERED_JSON="filtered_prs_$(basename "$INPUT_JSON")"
echo "$FILTERED_PRS" > "$FILTERED_JSON"

# Get total number of PRs after filtering
FILTERED_TOTAL=$(echo "$FILTERED_PRS" | jq '. | length')
echo "Total PRs after filtering: $FILTERED_TOTAL"

echo "Filtered PRs have been saved to $FILTERED_JSON"

# Disable debug mode
set +x
