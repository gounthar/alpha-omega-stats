#!/bin/bash
# This script runs a Gradle build and captures all output to a specified log file
set -euo pipefail
# Ensure a log file path is provided
if [ $# -lt 1 ]; then
  echo "Usage: $0 LOG_FILE [gradle_args...]" >&2
  exit 1
fi

# Get the log file path from the first argument
LOG_FILE="$1"
shift

# Run Gradle with the remaining arguments and capture all output
echo "=== BEGIN GRADLE OUTPUT ===" >> "$LOG_FILE"
./gradlew "$@" >> "$LOG_FILE" 2>&1
EXIT_CODE=$?
echo "=== END GRADLE OUTPUT ===" >> "$LOG_FILE"
# Return the Gradle exit code
exit $EXIT_CODE