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
source "$script_dir/install-jdk-versions.sh" # Changed from direct execution to sourcing

# Ensure JDK 25 is used for all Java and Maven commands
export JAVA_HOME="$HOME/.jdk-25"
export PATH="$JAVA_HOME/bin:$PATH"
hash -r

echo "DEBUG: Output of 'java -version' after sourcing install-jdk-versions.sh (in test-jdk-25.sh):" >> "$DEBUG_LOG"
java -version >> "$DEBUG_LOG" 2>&1
echo "DEBUG: Output of 'mvn -v' after sourcing install-jdk-versions.sh (in test-jdk-25.sh):" >> "$DEBUG_LOG"
mvn -v >> "$DEBUG_LOG" 2>&1

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

# Directory for per-plugin logs
PLUGIN_LOG_DIR="$(cd "$(dirname "$0")" && pwd)/data/plugin-build-logs"
mkdir -p "$PLUGIN_LOG_DIR"

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
    echo "DEBUG: Output of 'mvn -v' before potential JDK 25 switch (in test-jdk-25.sh):" >> "$DEBUG_LOG"
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

# Generate top-250-plugins.csv if it does not exist or is older than plugins.json
if [ ! -f "$CSV_FILE" ] || [ "$CSV_FILE" -ot "$PLUGINS_JSON" ]; then
    echo "Generating $CSV_FILE from $PLUGINS_JSON..."
    "$script_dir/get-most-popular-plugins.sh"
else
    echo "$CSV_FILE is up-to-date."
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
# Attempts to clone and build a Jenkins plugin, logging the process and returning the build status.
#
# Arguments:
#
# * plugin_name: The name of the Jenkins plugin to build.
#
# Returns:
#
# * A string indicating the build status, which may be one of: "success", "url_not_found", "clone_failed", "cd_failed", "build_failed", or "no_build_file".
#
# Example:
#
# ```bash
# status=$(compile_plugin "git")
# echo "Build status: $status"
# Attempts to clone and build a Jenkins plugin, returning the build status as a string.
#
# For the specified plugin name, retrieves its GitHub repository URL, clones the repository, and attempts to build it using Maven (if `pom.xml` is present) or Gradle (if `gradlew` is present), each with a 10-minute timeout. Build output is saved to a per-plugin log file. Returns a status string indicating the result, such as `success`, `url_not_found`, `clone_failed`, `cd_failed`, `build_failed`, `timeout`, or `no_build_file`.
#
# Arguments:
#
# * plugin_name: The name of the Jenkins plugin to build.
#
# Returns:
#
# * A string representing the build status: `success`, `url_not_found`, `clone_failed`, `cd_failed`, `build_failed`, `timeout`, or `no_build_file`.
#
# Example:
#
# ```bash
# status=$(compile_plugin "git")
# echo "Build status: $status"
# ```
compile_plugin() {
    local plugin_name="$1"
    local plugin_dir="$BUILD_DIR/$plugin_name"
    local build_status="success"
    local plugin_log_file="$PLUGIN_LOG_DIR/${plugin_name}.log"

    # Ensure the per-plugin log directory exists (handles parallel/recursive calls)
    mkdir -p "$PLUGIN_LOG_DIR"

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

            # Change to the plugin directory and log the result.
            cd "$plugin_dir" >>"$DEBUG_LOG" 2>&1 || {
                echo "Failed to change directory to $plugin_dir" >>"$DEBUG_LOG"
                build_status="cd_failed"
            }
            echo "Reached after cd command" >>"$DEBUG_LOG"
            echo "Successfully changed directory to $plugin_dir" >>"$DEBUG_LOG"
            if [ "$build_status" == "success" ]; then
                if [ -f "pom.xml" ]; then
                    # Ensure Maven's stdout and stderr are consistently captured in the per-plugin log
                    echo "Running Maven build for $plugin_name..." >>"$DEBUG_LOG"
                    echo "Executing: timeout 10m mvn clean install -DskipTests" >>"$DEBUG_LOG"
                    timeout 20m mvn clean install \
                      -Dmaven.test.skip=true \
                      -Dmaven.javadoc.skip=true \
                      -Dspotbugs.skip=true \
                      -Dcheckstyle.skip=true \
                      -Dlicense.skip=true \
                      -Daccess-modifier-checker.skip=true \
                      -Dmaven.compiler.fork=false \
                      -Dmaven.compiler.source=17 \
                      -Dmaven.compiler.target=17 \
                      -Dmaven.compiler.release=17 \
                      -Dgroovy.source.level=17 \
                      -Dorg.slf4j.simpleLogger.log.org.apache.maven.cli.transfer.Slf4jMavenTransferListener=warn \
                      -Dlicense.disableCheck=true \
                      -Dspotless.check.skip=true \
                      -Dpmd.skip=true \
                      -Dmaven.license.skip=true >"$plugin_log_file" 2>&1
                    maven_exit_code=$?
                    echo "Maven output for $plugin_name is in $plugin_log_file" >>"$DEBUG_LOG"
                    if [ $maven_exit_code -eq 124 ]; then
                        build_status="timeout"
                    elif [ $maven_exit_code -ne 0 ]; then
                        build_status="build_failed"
                    fi
                elif [ -f "./gradlew" ]; then
                    # Run a Gradle build if a Gradle wrapper is found.
                    echo "Running Gradle wrapper build for $plugin_name..." >>"$DEBUG_LOG"
                    echo "Executing: timeout 10m $script_dir/run-gradle-build.sh $DEBUG_LOG build -x test" >>"$DEBUG_LOG"
                    timeout 10m "$script_dir/run-gradle-build.sh" "$DEBUG_LOG" build -x test >"$plugin_log_file" 2>&1 || build_status="build_failed"
                else
                    # Log an error if no recognized build file is found.
                    echo "No recognized build file found for $plugin_name" >>"$DEBUG_LOG"
                    build_status="no_build_file"
                fi
            fi

            # Return to the previous directory and log the result.
            cd - >>"$DEBUG_LOG" 2>&1 || echo "Failed to return to the previous directory" >>"$DEBUG_LOG"
        fi
    fi

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
