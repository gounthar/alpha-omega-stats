#!/bin/bash

# Disable strict error checking and debug output for more reliable output handling
set -uo pipefail

# Call the script to install JDK versions
script_dir=$(cd "$(dirname "$0")" && pwd)
"$script_dir/install-jdk-versions.sh"

# Path to the input CSV file containing plugin names and their popularity.
CSV_FILE="top-250-plugins.csv"

# Path to the plugins JSON file downloaded from the Jenkins update center.
PLUGINS_JSON="plugins.json"

# Path to the directory where plugins will be cloned and built.
BUILD_DIR="/tmp/plugin-builds"

# Path to the output CSV file where build results will be saved.
RESULTS_FILE="jdk-25-build-results.csv"

# Path to the debug log file where detailed logs will be stored.
DEBUG_LOG="build-debug.log"

# Ensure the build directory exists, creating it if necessary.
mkdir -p "$BUILD_DIR"

# Initialize the results file with a header row.
echo "plugin_name,popularity,build_status" > "$RESULTS_FILE"

# Initialize the debug log file with a header.
echo "Build Debug Log" > "$DEBUG_LOG"

# Check if Maven is installed and accessible
if command -v mvn &>/dev/null; then
    echo "Maven is installed and accessible." >>"$DEBUG_LOG"
    mvn -v >>"$DEBUG_LOG" 2>&1
else
    echo "Error: Maven is not installed or not in the PATH. Please install Maven and try again." >>"$DEBUG_LOG"
    exit 1
fi

# Define a cleanup function to remove the build directory on script exit or interruption.
cleanup() {
    echo "Cleaning up build directory..."
    rm -rf "$BUILD_DIR"
}
# Register the cleanup function to be called on script exit or interruption.
trap cleanup EXIT

# Check if plugins.json exists and is older than one day
if [ ! -f "$PLUGINS_JSON" ] || [ "$(find "$PLUGINS_JSON" -mtime +0)" ]; then
  echo "Downloading $PLUGINS_JSON..."
  curl -L https://updates.jenkins.io/current/update-center.actual.json -o "$PLUGINS_JSON"
else
    echo "plugins.json is up-to-date."
fi

# Function to retrieve the GitHub URL of a plugin from the plugins JSON file.
# Arguments:
#   $1 - The name of the plugin.
# Returns:
#   The GitHub URL of the plugin, or an empty string if not found.
get_github_url() {
    local plugin_name="$1"
    jq -r --arg name "$plugin_name" '.plugins[] | select(.name == $name) | .scm | select(. != null)' "$PLUGINS_JSON"
}

# Function to clone and compile a plugin.
# Arguments:
#   $1 - The name of the plugin.
# Outputs:
#   Logs the build process to the debug log file and returns the build status.
compile_plugin() {
    local plugin_name="$1"
    local plugin_dir="$BUILD_DIR/$plugin_name"
    local build_status="success"

    echo "Processing plugin: $plugin_name" >>"$DEBUG_LOG"

    # Retrieve the GitHub URL for the plugin.
    local github_url
    github_url=$(get_github_url "$plugin_name")

    if [ -z "$github_url" ]; then
        echo "No GitHub URL found for $plugin_name" >>"$DEBUG_LOG"
        build_status="url_not_found"
    else
        git clone "$github_url" "$plugin_dir" >>"$DEBUG_LOG" 2>&1 || build_status="clone_failed"

        if [ "$build_status" == "success" ]; then
            echo "Cloned repository for $plugin_name." >>"$DEBUG_LOG"

            cd "$plugin_dir" >>"$DEBUG_LOG" 2>&1 || {
                echo "Failed to change directory to $plugin_dir" >>"$DEBUG_LOG"
                build_status="cd_failed"
            }
            echo "Reached after cd command" >>"$DEBUG_LOG"
            echo "Successfully changed directory to $plugin_dir" >>"$DEBUG_LOG"
            if [ "$build_status" == "success" ]; then
                if [ -f "pom.xml" ]; then
                    echo "Running Maven build for $plugin_name..." >>"$DEBUG_LOG"
                    echo "Running Maven build for $plugin_name..." >>"$DEBUG_LOG"
                    echo "Executing: mvn clean install -DskipTests" >>"$DEBUG_LOG"
                    "$script_dir/run-maven-build.sh" "$DEBUG_LOG" clean install -DskipTests || build_status="build_failed"
                    echo "Maven output for $plugin_name:" >>"$DEBUG_LOG"
                    cat mvn_output.log >>"$DEBUG_LOG"
                elif [ -f "build.gradle" ]; then
                    echo "Running Gradle build for $plugin_name..." >>"$DEBUG_LOG"
                    echo "Running Gradle build for $plugin_name..." >>"$DEBUG_LOG"
                    "$script_dir/run-gradle-build.sh" "$DEBUG_LOG" build -x test >>"$DEBUG_LOG" 2>&1 || build_status="build_failed"
                else
                    echo "No recognized build file found for $plugin_name" >>"$DEBUG_LOG"
                    build_status="no_build_file"
                fi
            fi

            cd - >>"$DEBUG_LOG" 2>&1 || echo "Failed to return to the previous directory" >>"$DEBUG_LOG"
        fi
    fi

    echo "Build status for $plugin_name: $build_status" >>"$DEBUG_LOG"
    rm -rf "$plugin_dir"
    echo "$build_status"
}

# Read the input CSV file and process each plugin.
while IFS=, read -r name popularity; do
    # Skip the header row in the CSV file.
    if [ "$name" != "name" ]; then
        # Compile the plugin and append the results to the output CSV file.
        build_status=$(compile_plugin "$name")
        echo "$name,$popularity,$build_status" >> "$RESULTS_FILE"
    fi
done < "$CSV_FILE"

# Log the completion of the script and the locations of the results and logs.
echo "Simplified build results have been saved to $RESULTS_FILE."
echo "Debug logs have been saved to $DEBUG_LOG."