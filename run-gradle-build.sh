#!/bin/bash
# This script runs a Gradle build and captures all output to a specified log file.

# Exit immediately if a command exits with a non-zero status, if an undefined variable is used,
# or if any command in a pipeline fails.
set -euo pipefail

# Ensure a log file path is provided as the first argument.
if [ $# -lt 1 ]; then
  # Print usage information to standard error and exit with a non-zero status.
  echo "Usage: $0 LOG_FILE [gradle_args...]" >&2
  exit 1
fi

# Get the log file path from the first argument.
LOG_FILE="$1"
# Shift the positional arguments so that the remaining arguments can be passed to Gradle.
shift

# Ensure the Gradle wrapper script is executable.
chmod +x ./gradlew

# Run the Gradle build with the remaining arguments and capture all output to the specified log file.
# Append a header to the log file indicating the start of Gradle output.
echo "=== BEGIN GRADLE OUTPUT ===" >> "$LOG_FILE"
# Execute the Gradle wrapper with the provided arguments, redirecting both stdout and stderr to the log file.
./gradlew "$@" >> "$LOG_FILE" 2>&1
# Capture the exit code of the Gradle command.
EXIT_CODE=$?
# Append a footer to the log file indicating the end of Gradle output.
echo "=== END GRADLE OUTPUT ===" >> "$LOG_FILE"

# Return the Gradle exit code to the caller.
exit $EXIT_CODE