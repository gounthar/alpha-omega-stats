#!/bin/bash

# Input JSON file (passed as a parameter)
INPUT_JSON="$1"
PLUGINS_JSON="$2"

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
PLUGIN_REPOS=$(jq -r '.plugins | to_entries[] | select(.value.scm != null) | .value.scm | sub("https://github.com/"; "")' "$PLUGINS_JSON")
echo "Extracted plugin repositories:"
echo "$PLUGIN_REPOS"

# Validate that PLUGIN_REPOS is not empty
if [ -z "$PLUGIN_REPOS" ]; then
  echo "Error: No plugin repositories were extracted from '$PLUGINS_JSON'" >&2
  exit 1
fi

# Convert the list of repository names into a JSON array for use with --argjson
PLUGIN_REPOS_JSON=$(echo "$PLUGIN_REPOS" | jq -R -s -c 'split("\n") | map(select(. != ""))')

# Filter PRs that are related to Jenkins plugins
FILTERED_PRS=$(jq --argjson plugin_repos "$PLUGIN_REPOS_JSON" '
  map(select(
    .repository as $repo |
    $plugin_repos | index($repo) != null
  ))
' "$INPUT_JSON")

# Output the filtered PRs to a new JSON file
FILTERED_JSON="filtered_prs_$(basename "$INPUT_JSON")"
echo "$FILTERED_PRS" > "$FILTERED_JSON"

echo "Filtered PRs have been saved to $FILTERED_JSON"
