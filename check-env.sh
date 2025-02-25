#!/usr/bin/env bash
# set -x -o errexit -o nounset -o pipefail
set -euo pipefail

source log-utils.sh

# Check if the DEBUG_MODE environment variable is set
if [ "${DEBUG_MODE:-false}" = "true" ]; then
  # If DEBUG_MODE is set to true, print a debug message
  debug "Debug mode is on."
else
  # If DEBUG_MODE is not set to true, print an info message
  info "Debug mode is off. To turn it on, set the DEBUG_MODE environment variable to true."
  DEBUG_MODE=false
  export DEBUG_MODE
fi


# Ensure jq is installed. jq is a command-line JSON processor.
# We use it to parse the JSON response from the GitHub API.
if ! [ -x "$(command -v jq)" ]; then
  error 'jq is not installed.'
  info 'You can install it by running: sudo apt-get install jq (for Ubuntu/Debian) or brew install jq (for MacOS)'
  exit 1
fi

# Ensure parallel is installed. parallel is a shell tool for executing jobs in parallel.
# We use it to process multiple repositories concurrently.
if ! [ -x "$(command -v parallel)" ]; then
  error 'parallel is not installed.'
  info 'You can install it by running: sudo apt-get install parallel (for Ubuntu/Debian) or brew install parallel (for MacOS)'
  exit 1
fi

# Ensure GITHUB_TOKEN is set. GITHUB_TOKEN is a GitHub Personal Access Token that we use to authenticate with the GitHub API.
# You need to generate this token in your GitHub account settings and set it as an environment variable before running this script.
if [ -z "${GITHUB_TOKEN-}" ]; then
  error 'The GITHUB_TOKEN env var is not set.'
  exit 1
fi

# Ensure gh is installed. gh is the GitHub CLI tool.
# We use it to interact with GitHub from the command line.
if ! [ -x "$(command -v gh)" ]; then
  error 'gh is not installed.'
  info 'You can install it by following the instructions at: https://github.com/cli/cli#installation'
  exit 1
fi

# Ensure mvn is installed
if ! [ -x "$(command -v mvn)" ]; then
  error 'Error: mvn is not installed.' >&2
  exit 1
fi

# Ensure JAVA_HOME is set
if [ -z "${JAVA_HOME-}" ]; then
  # Print a warning message in yellow
  warning "Warning: JAVA_HOME environment variable is not set." >&2
  # Try to infer JAVA_HOME from java command path
  if command -v java >/dev/null; then
    temp_java=$(command -v java)
    JAVA_HOME=$(dirname "$(dirname "$(readlink -f "$temp_java")")")
    export JAVA_HOME
    # Print a success message in green
    success "JAVA_HOME set to $JAVA_HOME"
  else
    # Print an error message in red
    error "Error: java command not found. Cannot set JAVA_HOME." >&2
    exit 1
  fi
fi

# Ensure Bash version is 5 or higher
BASH_VERSION_MAJOR=${BASH_VERSION%%.*}
if (( BASH_VERSION_MAJOR < 5 )); then
  error 'Your Bash version is less than 5. We need it for the associative arrays.'
  info 'You can upgrade it by running: brew install bash (for MacOS)'
  exit 1
fi
