#!/bin/bash
# This script runs a Maven build and captures all output to a specified log file.

# Exit immediately if a command exits with a non-zero status, if an undefined variable is used,
# or if any command in a pipeline fails.
set -euo pipefail

# Check if at least one argument (the log file path) is provided.
if [ $# -lt 1 ]; then
  # Print usage information to standard error and exit with a non-zero status.
  echo "Usage: $0 LOG_FILE [mvn_args...]" >&2
  exit 1
fi

# Get the log file path from the first argument.
LOG_FILE="$1"
# Shift the positional arguments so that the remaining arguments can be passed to Maven.
shift

# Append a header to the log file indicating the start of Maven output.
echo "=== BEGIN MAVEN OUTPUT ===" >> "$LOG_FILE"
# Run Maven with the provided arguments, redirecting both stdout and stderr to the log file.
mvn "$@" >> "$LOG_FILE" 2>&1
# Capture the exit code of the Maven command.
EXIT_CODE=$?
# Append a footer to the log file indicating the end of Maven output.
echo "=== END MAVEN OUTPUT ===" >> "$LOG_FILE"

# Return the Maven exit code to the caller.
exit $EXIT_CODE