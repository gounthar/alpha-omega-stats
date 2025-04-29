#!/bin/bash

# Path to the input CSV file
CSV_FILE="top-250-plugins.csv"

# Path to the directory where plugins will be cloned and built
BUILD_DIR="/tmp/plugin-builds"

# Path to the output CSV file for build results
RESULTS_FILE="jdk-25-build-results.csv"

# Path to the debug log file
DEBUG_LOG="build-debug.log"

# Ensure the build directory exists
mkdir -p "$BUILD_DIR"

# Initialize the results file with a header
echo "plugin_name,popularity,build_status" > "$RESULTS_FILE"

# Initialize the debug log file
echo "Build Debug Log" > "$DEBUG_LOG"

# Trap to clean up on exit or interruption
cleanup() {
    echo "Cleaning up build directory..."
    rm -rf "$BUILD_DIR"
}
trap cleanup EXIT

compile_plugin() {
    local plugin_name="$1"
    local plugin_dir="$BUILD_DIR/$plugin_name"
    local build_status="success"

    echo "Processing plugin: $plugin_name" >>"$DEBUG_LOG"

    # Clone the plugin repository
    git clone "https://github.com/jenkinsci/${plugin_name}-plugin.git" "$plugin_dir" &>>"$DEBUG_LOG" || build_status="clone_failed"

    # Change to the plugin directory
    if [ "$build_status" == "success" ]; then
        cd "$plugin_dir" &>>"$DEBUG_LOG" || build_status="cd_failed"

        # Attempt to compile the plugin
        if [ -f "pom.xml" ]; then
            echo "Running Maven build for $plugin_name..." >>"$DEBUG_LOG"
            mvn clean install -DskipTests >>"$DEBUG_LOG" 2>&1 || build_status="build_failed"
        elif [ -f "build.gradle" ]; then
            echo "Running Gradle build for $plugin_name..." >>"$DEBUG_LOG"
            ./gradlew build -x test >>"$DEBUG_LOG" 2>&1 || build_status="build_failed"
        else
            echo "No recognized build file found for $plugin_name" >>"$DEBUG_LOG"
            build_status="no_build_file"
        fi

        # Return to the original directory
        cd - &>>"$DEBUG_LOG" || build_status="cd_failed"
    fi

    # Log the final status
    echo "Build status for $plugin_name: $build_status" >>"$DEBUG_LOG"

    # Clean up the plugin directory
    rm -rf "$plugin_dir"

    # Return the build status
    echo "$build_status"
}
# Read the CSV file and compile each plugin
while IFS=, read -r name popularity; do
    # Skip the header line
    if [ "$name" != "name" ]; then
        build_status=$(compile_plugin "$name")
        echo "$name,$popularity,$build_status" >> "$RESULTS_FILE"
    fi
done < "$CSV_FILE"

echo "Simplified build results have been saved to $RESULTS_FILE."
echo "Debug logs have been saved to $DEBUG_LOG."