#!/bin/bash

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

# Define a cleanup function to remove the build directory on script exit or interruption.
cleanup() {
    echo "Cleaning up build directory..."
    rm -rf "$BUILD_DIR"
}
# Register the cleanup function to be called on script exit or interruption.
trap cleanup EXIT

# Download the plugins JSON file from the Jenkins update center.
curl -L https://updates.jenkins.io/current/update-center.actual.json -o plugins.json

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
        # Log and set status if no GitHub URL is found.
        echo "No GitHub URL found for $plugin_name" >>"$DEBUG_LOG"
        build_status="url_not_found"
    else
        # Clone the plugin repository.
        git clone "$github_url" "$plugin_dir" >>"$DEBUG_LOG" 2>&1 || build_status="clone_failed"

        # Change to the plugin directory if cloning succeeded.
        if [ "$build_status" == "success" ]; then
            cd "$plugin_dir" >>"$DEBUG_LOG" 2>&1 || build_status="cd_failed"

            # Attempt to compile the plugin using Maven or Gradle.
            if [ -f "pom.xml" ]; then
                echo "Running Maven build for $plugin_name..." >>"$DEBUG_LOG"
                mvn clean install -DskipTests >>"$DEBUG_LOG" 2>&1 || build_status="build_failed"
            elif [ -f "build.gradle" ]; then
                echo "Running Gradle build for $plugin_name..." >>"$DEBUG_LOG"
                ./gradlew build -x test >>"$DEBUG_LOG" 2>&1 || build_status="build_failed"
            else
                # Log and set status if no recognized build file is found.
                echo "No recognized build file found for $plugin_name" >>"$DEBUG_LOG"
                build_status="no_build_file"
            fi

            # Return to the original directory.
            cd - >>"$DEBUG_LOG" 2>&1 || build_status="cd_failed"
        fi
    fi

    # Log the final build status.
    echo "Build status for $plugin_name: $build_status" >>"$DEBUG_LOG"

    # Remove the plugin directory to clean up.
    rm -rf "$plugin_dir"

    # Return the build status.
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