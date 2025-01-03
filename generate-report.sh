#!/bin/bash

# Check if the input JSON file is provided
if [ -z "$1" ]; then
  echo "Usage: $0 <input-json-file>"
  exit 1
fi

# Input JSON file (passed as a parameter)
INPUT_JSON="$1"

# Validate that the file exists and is readable
if [ ! -r "$INPUT_JSON" ]; then
  echo "Error: Input file '$INPUT_JSON' does not exist or is not readable" >&2
  exit 1
fi

# Validate JSON structure
if ! jq empty "$INPUT_JSON" 2>/dev/null; then
  echo "Error: Invalid JSON format in '$INPUT_JSON'" >&2
  exit 1
fi

# Validate required fields are present
if ! jq -e 'all(has("repository") and has("user") and has("title") and has("number") and has("state"))' "$INPUT_JSON" >/dev/null; then
  echo "Error: JSON is missing required fields (repository, user, title, number, or state)" >&2
  exit 1
fi

# Debugging: Print the input filename
echo "Input filename: $INPUT_JSON"

# Extract the year and month number from the filename
YEAR=$(basename "$INPUT_JSON" | cut -d'_' -f5 | cut -d'-' -f1)
MONTH=$(basename "$INPUT_JSON" | cut -d'_' -f5 | cut -d'-' -f2)

# Debugging: Print the extracted year and month
echo "Extracted year: $YEAR"
echo "Extracted month: $MONTH"

# Map month number to month name
case $MONTH in
  01) MONTH_NAME="January" ;;
  02) MONTH_NAME="February" ;;
  03) MONTH_NAME="March" ;;
  04) MONTH_NAME="April" ;;
  05) MONTH_NAME="May" ;;
  06) MONTH_NAME="June" ;;
  07) MONTH_NAME="July" ;;
  08) MONTH_NAME="August" ;;
  09) MONTH_NAME="September" ;;
  10) MONTH_NAME="October" ;;
  11) MONTH_NAME="November" ;;
  12) MONTH_NAME="December" ;;
  *) MONTH_NAME="Unknown" ;;
esac

# Debugging: Print the month name
echo "Month name: $MONTH_NAME"

# Output Markdown file with year and month
OUTPUT_MD="jenkins-csp-${MONTH_NAME,,}-${YEAR}-report.md"

# Function to generate the Markdown report
generate_report() {
  echo "# ${MONTH_NAME} ${YEAR} - Jenkins CSP Project Update" > "$OUTPUT_MD"
  echo "" >> "$OUTPUT_MD"

  # Summary section
  if ! SUMMARY=$(jq -r '{
    prs: length,
    repos: (group_by(.repository) | length),
    users: (group_by(.user) | length),
    open: map(select(.state == "OPEN")) | length,
    closed: map(select(.state == "CLOSED")) | length,
    merged: map(select(.state == "MERGED")) | length
  } | . += {
    open_percent: (.open / .prs * 100 | floor),
    closed_percent: (.closed / .prs * 100 | floor),
    merged_percent: (.merged / .prs * 100 | floor)
  }' "$INPUT_JSON" 2>/dev/null); then
    echo "Error: Failed to process JSON data for summary" >&2
    exit 1
  fi
  TOTAL_PRS=$(echo "$SUMMARY" | jq '.prs')
  TOTAL_REPOS=$(echo "$SUMMARY" | jq '.repos')
  TOTAL_USERS=$(echo "$SUMMARY" | jq '.users')
  OPEN_PRS=$(echo "$SUMMARY" | jq '.open')
  CLOSED_PRS=$(echo "$SUMMARY" | jq '.closed')
  MERGED_PRS=$(echo "$SUMMARY" | jq '.merged')

  echo "## Summary" >> "$OUTPUT_MD"
  echo "- Total PRs: $TOTAL_PRS" >> "$OUTPUT_MD"
  echo "- Total Repositories: $TOTAL_REPOS" >> "$OUTPUT_MD"
  echo "- Total Users: $TOTAL_USERS" >> "$OUTPUT_MD"
  echo "- Open PRs: $OPEN_PRS ($(echo "$SUMMARY" | jq '.open_percent')%)" >> "$OUTPUT_MD"
  echo "- Closed PRs: $CLOSED_PRS ($(echo "$SUMMARY" | jq '.closed_percent')%)" >> "$OUTPUT_MD"
  echo "- Merged PRs: $MERGED_PRS ($(echo "$SUMMARY" | jq '.merged_percent')%)" >> "$OUTPUT_MD"
  echo "" >> "$OUTPUT_MD"

  echo "## Pull Requests by Repository" >> "$OUTPUT_MD"
  echo "" >> "$OUTPUT_MD"

  # Step 1: Group PRs by repository
  jq -r '
    def get_first_name(handle):
      handle as $original |
      try (
        $first_names[handle] // 
        empty
      ) // $original;

    reduce (inputs | split("\n")[]) as $line (
      [];
      . + [
        if $line | test("^#### User: ") then
          sub("^#### User: "; "") | get_first_name(.)
        else
          $line
        end
      ]
    ) | .[] |
    group_by(.repository)[] |
    "### \(.[0].repository)\n" + (
      # Step 2: Group PRs by user within each repository
      group_by(.user)[] |
      "#### User: \(.[0].user)\n" + (
        # Step 3: List all PRs for the user with the new format
        .[] |
        if .state == "OPEN" then
          "- \(.user) is working on [\(.title // "[No title]")](https://github.com/\(.repository)/pull/\(.number)) (\(.createdAt))\n"
        elif .state == "CLOSED" or .state == "MERGED" then
          "- \(.user) has worked on [\(.title // "[No title]")](https://github.com/\(.repository)/pull/\(.number)) (\(.createdAt))\n"
        else
          "- [\(.title // "[No title]")](https://github.com/\(.repository)/pull/\(.number)) (\(.createdAt)) - Status: \(.state // "null")\n"
        end
      )
    )
  ' "$INPUT_JSON" > "$OUTPUT_MD"

  echo "" >> "$OUTPUT_MD"
  # Read template sections from file
  if [ -f "report-template.md" ]; then
   cat "report-template.md" >> "$OUTPUT_MD"
 else
     echo "" >> "$OUTPUT_MD"
     echo "## Key Highlights" >> "$OUTPUT_MD"
     echo "- Continued progress in modernizing Jenkins plugins" >> "$OUTPUT_MD"
     echo "- Systematic removal of legacy JavaScript and inline event handlers" >> "$OUTPUT_MD"
     echo "- Enhanced Content Security Policy (CSP) compatibility" >> "$OUTPUT_MD"
     echo "- Proactive identification and resolution of potential security vulnerabilities" >> "$OUTPUT_MD"
     echo "" >> "$OUTPUT_MD"
     echo "## Next Steps" >> "$OUTPUT_MD"
     echo "- Continue plugin modernization efforts" >> "$OUTPUT_MD"
     echo "- Prioritize plugins with known CSP challenges" >> "$OUTPUT_MD"
     echo "- Expand CSP scanner capabilities" >> "$OUTPUT_MD"
     echo "- Collaborate with plugin maintainers to implement best practices" >> "$OUTPUT_MD"
  fi
}

# Run the report generation
generate_report

echo "Markdown report generated: $OUTPUT_MD"
