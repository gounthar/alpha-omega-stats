#!/bin/bash

# Check if the input JSON file is provided
if [ -z "$1" ]; then
  echo "Usage: $0 <input-json-file>"
  exit 1
fi

# Input JSON file (passed as a parameter)
INPUT_JSON="$1"

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

# Function to get the first name of a GitHub user
get_first_name() {
  local github_handle=$1
  local full_name=$(gh api users/$github_handle --jq .name)
  local first_name=$(echo "$full_name" | cut -d' ' -f1)
  echo "$first_name"
}

# Function to generate the Markdown report
generate_report() {
  echo "# ${MONTH_NAME} ${YEAR} - Jenkins CSP Project Update" > "$OUTPUT_MD"
  echo "" >> "$OUTPUT_MD"
  echo "## Pull Requests by Repository" >> "$OUTPUT_MD"
  echo "" >> "$OUTPUT_MD"

  # Step 1: Group PRs by repository
  jq -r '
    group_by(.repository)[] |
    "### \(.[0].repository)\n" + (
      # Step 2: Group PRs by user within each repository
      group_by(.user)[] |
      "#### User: \(.[0].user)\n" + (
        # Step 3: List all PRs for the user
        .[] |
        "- [\(.title)](https://github.com/\(.repository)/pull/\(.number)) (\(.createdAt))"
      ) + "\n"
    )
  ' "$INPUT_JSON" >> "$OUTPUT_MD"

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
}

# Run the report generation
generate_report

echo "Markdown report generated: $OUTPUT_MD"
