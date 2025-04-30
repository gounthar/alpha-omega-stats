#!/bin/bash
# This script runs a Maven build and captures all output to a specified log file
set -euo pipefail
# Ensure a log file path is provided
if [ $# -lt 1 ]; then
  echo "Usage: $0 LOG_FILE [mvn_args...]" >&2
  exit 1
fi

# Get the log file path from the first argument
LOG_FILE="$1"
shift

# Run Maven with the remaining arguments and capture all output
echo "=== BEGIN MAVEN OUTPUT ===" >> "$LOG_FILE"
mvn "$@" >> "$LOG_FILE" 2>&1
EXIT_CODE=$?
echo "=== END MAVEN OUTPUT ===" >> "$LOG_FILE"
# Return the Maven exit code
exit $EXIT_CODE