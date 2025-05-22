#!/bin/bash

# Script to find Jenkins plugins released in the last X hours
# Usage: ./get-recent-releases.sh [hours]
# Default is 24 hours if not specified

# Number of hours to look back (default: 24)
HOURS=${1:-24}

# Path to the release history JSON file
RELEASE_HISTORY_JSON="release-history.json"
RELEASE_HISTORY_URL="https://westeurope.cloudflare.jenkins.io/current/release-history.json"

# Download the latest release history JSON file
echo "Downloading latest release history data..."
if command -v curl &> /dev/null; then
    curl -L "$RELEASE_HISTORY_URL" -o "$RELEASE_HISTORY_JSON"
fi

# Calculate the timestamp for X hours ago in seconds since epoch
HOURS_AGO_SECONDS=$(date -d "$HOURS hours ago" +%s 2>/dev/null || date -v-${HOURS}H +%s 2>/dev/null || powershell -Command "(Get-Date).AddHours(-$HOURS).ToUniversalTime().ToFileTimeUtc()/10000000-11644473600")

echo "Finding plugins released in the last $HOURS hours..."
echo ""
echo "PLUGIN NAME                VERSION   RELEASE DATE          CHANGELOG"
echo "-------------------------  --------  -------------------   ------------------------------------------"

# Use jq to extract plugins released in the last X hours
jq -r --argjson hours_ago "$HOURS_AGO_SECONDS" '
  .releaseHistory[] | 
  .releases[] | 
  select(.timestamp | tonumber | . / 1000 > $hours_ago) | 
  {
    name: (.url | split("/") | last), 
    version: .version, 
    date: (.timestamp | tonumber / 1000 | strftime("%Y-%m-%d %H:%M:%S")),
    url: .url
  } | 
  [.name, .version, .date, .url] | 
  @tsv
' "$RELEASE_HISTORY_JSON" | sort -k3,3r | while IFS=$'\t' read -r name version date url; do
    printf "%-25s  %-8s  %-19s   %s\n" "$name" "$version" "$date" "$url"
done

# Clean up
rm "$RELEASE_HISTORY_JSON"

echo ""
echo "Note: For more details, visit https://plugins.jenkins.io/ and search for the plugin name."