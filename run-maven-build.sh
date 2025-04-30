#!/bin/bash
# This script runs a Maven build and captures all output to a specified log file

# Get the log file path from the first argument
LOG_FILE="$1"
shift

# Run Maven with the remaining arguments and capture all output
echo "=== BEGIN MAVEN OUTPUT ===" >> "$LOG_FILE"
mvn "$@" 2>&1 >> "$LOG_FILE"
EXIT_CODE=$?
echo "=== END MAVEN OUTPUT ===" >> "$LOG_FILE"

# Return the Maven exit code
exit $EXIT_CODE