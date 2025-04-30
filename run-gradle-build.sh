#!/bin/bash
# This script runs a Gradle build and captures all output to a specified log file

# Get the log file path from the first argument
LOG_FILE="$1"
shift

# Run Gradle with the remaining arguments and capture all output
echo "=== BEGIN GRADLE OUTPUT ===" >> "$LOG_FILE"
./gradlew "$@" 2>&1 >> "$LOG_FILE"
EXIT_CODE=$?
echo "=== END GRADLE OUTPUT ===" >> "$LOG_FILE"

# Return the Gradle exit code
exit $EXIT_CODE