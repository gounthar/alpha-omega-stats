#!/bin/bash

# Input JSON file
INPUT_JSON="prs_yaroslavafenkin_2024-12-01_to_2024-12-31.json"
# Output Markdown file
OUTPUT_MD="jenkins-csp-december-report.md"

# Function to generate the Markdown report
generate_report() {
  echo "# December 2024 - Jenkins CSP Project Update" > "$OUTPUT_MD"
  echo "" >> "$OUTPUT_MD"
  echo "## Pull Requests by Repository" >> "$OUTPUT_MD"
  echo "" >> "$OUTPUT_MD"

  # Group PRs by repository
  jq -r 'group_by(.repository)[] | "### \(.[0].repository)\n" + (.[] | "- [\(.title)](https://github.com/\(.repository)/pull/\(.number)) (\(.createdAt))") + "\n"' "$INPUT_JSON" >> "$OUTPUT_MD"

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
