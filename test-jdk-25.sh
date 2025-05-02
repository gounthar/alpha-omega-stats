#!/bin/bash

# Ensure DEBUG_LOG is defined and exported
DEBUG_LOG="build-debug.log"
export DEBUG_LOG

# Disable strict error checking and debug output for more reliable output handling
set -uo pipefail

# Detect the system architecture dynamically
ARCHITECTURE=$(uname -m)

# Map architecture to the expected values for the API
case "$ARCHITECTURE" in
    x86_64)
        ARCHITECTURE="x64";;
    aarch64)
        ARCHITECTURE="aarch64";;
    riscv64)
        ARCHITECTURE="riscv64";;
    *)
        echo "Error: Unsupported architecture $ARCHITECTURE" >> "$DEBUG_LOG"
        exit 1;;
esac

# Call the script to install JDK versions
# The script directory is determined and stored in the variable `script_dir`.
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
    # Log Maven installation details to the debug log.
    echo "Maven is installed and accessible." >>"$DEBUG_LOG"
    mvn -v >>"$DEBUG_LOG" 2>&1
else
    # Log an error message and exit if Maven is not installed.
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
    # Download the latest plugins JSON file from the Jenkins update center.
    echo "Downloading $PLUGINS_JSON..."
    curl -L https://updates.jenkins.io/current/update-center.actual.json -o "$PLUGINS_JSON"
else
    # Log that the plugins JSON file is up-to-date.
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

    # Log the start of processing for the plugin.
    echo "Processing plugin: $plugin_name" >>"$DEBUG_LOG"

    # Retrieve the GitHub URL for the plugin.
    local github_url
    github_url=$(get_github_url "$plugin_name")

    if [ -z "$github_url" ]; then
        # Log an error if no GitHub URL is found for the plugin.
        echo "No GitHub URL found for $plugin_name" >>"$DEBUG_LOG"
        build_status="url_not_found"
    else
        # Clone the plugin repository and log the result.
        git clone "$github_url" "$plugin_dir" >>"$DEBUG_LOG" 2>&1 || build_status="clone_failed"

        if [ "$build_status" == "success" ]; then
            echo "Cloned repository for $plugin_name." >>"$DEBUG_LOG"

            # Add logging before attempting cd
            echo "Attempting to cd into $plugin_dir" >>"$DEBUG_LOG"
            # Change to the plugin directory and capture exit code
            cd "$plugin_dir"
            local cd_exit_code=$?
            # Log the exit code
            echo "cd exit code: $cd_exit_code" >>"$DEBUG_LOG"

            if [ $cd_exit_code -ne 0 ]; then
                echo "Failed to change directory to $plugin_dir" >>"$DEBUG_LOG"
                build_status="cd_failed"
            else
                # Log success *after* checking exit code
                echo "Successfully changed directory to $plugin_dir" >>"$DEBUG_LOG"

                # Check for build files only if cd was successful
                if [ -f "pom.xml" ]; then
                    # Run a Maven build if a pom.xml file is found.
                    echo "Running Maven build for $plugin_name..." >>"$DEBUG_LOG"
                    echo "Executing: mvn clean install -DskipTests" >>"$DEBUG_LOG"
                    # Create an absolute path for the temporary Maven output log
                    maven_log_file="$(pwd)/mvn_output.log"
                    # Use the absolute path when calling run-maven-build.sh
                    "$script_dir/run-maven-build.sh" "$maven_log_file" clean install -DskipTests
                    maven_exit_code=$?
                    # Always read the log file regardless of build success/failure
                    echo "Maven output for $plugin_name:" >>"$DEBUG_LOG"
                    cat "$maven_log_file" >>"$DEBUG_LOG" 2>/dev/null || echo "Failed to read Maven output log" >>"$DEBUG_LOG"
                    # Then check exit code
                    if [ $maven_exit_code -ne 0 ]; then
                        build_status="build_failed"
                    fi
                    rm "$maven_log_file"
                elif [ -f "./gradlew" ]; then
                    # Run a Gradle build if a Gradle wrapper is found.
                    echo "Running Gradle wrapper build for $plugin_name..." >>"$DEBUG_LOG"
                    "$script_dir/run-gradle-build.sh" "$DEBUG_LOG" build -x test >>"$DEBUG_LOG" 2>&1 || build_status="build_failed"
                else
                    # Log an error if no recognized build file is found.
                    echo "No recognized build file found for $plugin_name" >>"$DEBUG_LOG"
                    build_status="no_build_file"
                fi

                # Return to the previous directory only if cd was successful
                echo "Attempting to cd back from $plugin_dir" >> "$DEBUG_LOG" # Add log before cd -
                cd - >>"$DEBUG_LOG" 2>&1 || echo "Failed to return to the previous directory" >>"$DEBUG_LOG"
                echo "Successfully cd'd back from $plugin_dir" >> "$DEBUG_LOG" # Add log after cd -
            fi # End of if cd_exit_code == 0
        fi # End of if clone status == success
    fi # End of if github_url exists

    # Log the build status for the plugin and clean up the plugin directory.
    echo "Build status for $plugin_name: $build_status" >>"$DEBUG_LOG"
    rm -rf "$plugin_dir"
    echo "$build_status"
}

# Read the input CSV file using file descriptor 3 to avoid consuming stdin
line_number=0
while IFS=, read -r name popularity <&3; do
    line_number=$((line_number + 1))
    echo "Read line $line_number: name='$name', popularity='$popularity'" >> "$DEBUG_LOG"

    # Skip the header row in the CSV file.
    if [ "$name" != "name" ]; then
        echo "Processing plugin '$name' from line $line_number" >> "$DEBUG_LOG"
        build_status=$(compile_plugin "$name")
        echo "Finished processing plugin '$name' from line $line_number with status: $build_status" >> "$DEBUG_LOG"
        echo "$name,$popularity,$build_status" >> "$RESULTS_FILE"
    else
        echo "Skipping header line $line_number" >> "$DEBUG_LOG"
    fi
done 3< "$CSV_FILE" # Use file descriptor 3 for reading the CSV

echo "Finished reading $CSV_FILE after $line_number lines." >> "$DEBUG_LOG"

# Log the completion of the script and the locations of the results and logs.
echo "Simplified build results have been saved to $RESULTS_FILE."
echo "Debug logs have been saved to $DEBUG_LOG."