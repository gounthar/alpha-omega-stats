#!/usr/bin/env bash
# Strict mode: exit on error, undefined var, and failures in pipelines
# Ensure we are running under Bash (not sh/dash/zsh) to support 'pipefail'
if [ -z "${BASH_VERSION:-}" ]; then
  echo "This script must be run with bash. Try: bash \"$0\"" >&2
  # If sourced into a non-bash shell, avoid killing the shell
  return 1 2>/dev/null || exit 1
fi
set -euo pipefail
IFS=$'\n\t'

# Determine script directory once
script_dir=$(cd "$(dirname "$0")" && pwd)

# Initialize and export DEBUG_LOG before any output or redirects
DEBUG_LOG="$script_dir/build-debug.log"
export DEBUG_LOG

# Prepare and activate Python virtual environment safely
venv_dir="$script_dir/venv"
activate_file="$venv_dir/bin/activate"

if [ ! -d "$venv_dir" ] || [ ! -f "$activate_file" ]; then
    echo "Python virtualenv not found at $venv_dir. Attempting to create it..." >> "$DEBUG_LOG"
    if ! command -v python3 >/dev/null 2>&1; then
        echo "Error: python3 is not installed or not in PATH. Cannot create virtual environment." | tee -a "$DEBUG_LOG"
        exit 1
    fi
    if ! python3 -m venv "$venv_dir"; then
        echo "Error: Failed to create virtual environment at $venv_dir." | tee -a "$DEBUG_LOG"
        exit 1
    fi
fi

if [ ! -f "$activate_file" ]; then
    echo "Error: virtualenv activation script missing at $activate_file." | tee -a "$DEBUG_LOG"
    exit 1
fi

# Source the virtual environment; exit if activation fails
if ! source "$activate_file"; then
    echo "Error: Failed to activate virtual environment at $activate_file." | tee -a "$DEBUG_LOG"
    exit 1
fi

# Export Google Sheet to TSV before running the rest of the script
SPREADSHEET_ID_OR_NAME="1_XHzakLNwA44cUnRsY01kQ1X1SymMcJGFxXzhr5s3_s" # or use the Sheet ID
WORKSHEET_NAME="Java 25 compatibility progress" # Change to your worksheet name
OUTPUT_TSV="plugins-jdk25.tsv"
TSV_FILE="$OUTPUT_TSV"

# Ensure VENV_DIR points to our venv directory (for clarity in later calls)
VENV_DIR="$venv_dir"
export VENV_DIR

# Install required Python packages using the venv's pip, logging stdout and stderr
"$VENV_DIR/bin/pip" install -r "$script_dir/requirements.txt" >> "$DEBUG_LOG" 2>&1

# Export Google Sheet to TSV using the venv's python, logging stdout and stderr
if ! "$VENV_DIR/bin/python" "$script_dir/export_sheet_to_tsv.py" "$SPREADSHEET_ID_OR_NAME" "$WORKSHEET_NAME" "$OUTPUT_TSV" >> "$DEBUG_LOG" 2>&1; then
    echo "Failed to export Google Sheet to TSV. Continuing without TSV data." >> "$DEBUG_LOG"
    TSV_FILE=""
fi

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
# Run installer in a separate bash process to avoid set -e propagation from sourced scripts
echo "DEBUG: Starting install-jdk-versions.sh" >> "$DEBUG_LOG"
set +e
bash "$script_dir/install-jdk-versions.sh" >> "$DEBUG_LOG" 2>&1
installer_ec=$?
set -e
if [ $installer_ec -ne 0 ]; then
  echo "WARN: install-jdk-versions.sh exited with code $installer_ec, continuing." >> "$DEBUG_LOG"
else
  echo "DEBUG: install-jdk-versions.sh completed successfully." >> "$DEBUG_LOG"
fi

# Ensure JDK 25 is used for all Java and Maven commands
export JAVA_HOME="$HOME/.jdk-25"
export PATH="$JAVA_HOME/bin:$PATH"
hash -r

# Configure per-build threads and cross-plugin concurrency (defaults to 4)
export BUILD_THREADS="${BUILD_THREADS:-4}"
export PLUGIN_CONCURRENCY="${PLUGIN_CONCURRENCY:-4}"

# Validate numeric, positive integers
case "$BUILD_THREADS" in ''|*[!0-9]*|0) BUILD_THREADS=4;; esac
case "$PLUGIN_CONCURRENCY" in ''|*[!0-9]*|0) PLUGIN_CONCURRENCY=4;; esac

# Keep total concurrency roughly bounded by CPU count when both are unset or excessive
CPU_COUNT="$(getconf _NPROCESSORS_ONLN 2>/dev/null || nproc 2>/dev/null || echo 4)"
# Force base-10 to avoid octal interpretation of leading zeros
bt=$((10#${BUILD_THREADS}))
pc=$((10#${PLUGIN_CONCURRENCY}))
if [ "$CPU_COUNT" -gt 0 ] && [ $(( bt * pc )) -gt "$CPU_COUNT" ]; then
  # Reduce per-build threads to fit within CPU_COUNT
  BUILD_THREADS=$(( (CPU_COUNT + pc - 1) / pc ))
  [ "$BUILD_THREADS" -lt 1 ] && BUILD_THREADS=1
  echo "Clamped BUILD_THREADS to ${BUILD_THREADS} based on CPU_COUNT=${CPU_COUNT} and PLUGIN_CONCURRENCY=${PLUGIN_CONCURRENCY}" >> "$DEBUG_LOG"
fi

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
TSV_RESULTS_FILE="jdk-25-build-results-extended.csv"

# Path to the debug log file where detailed logs will be stored. Using DEBUG_LOG defined above.

# Directory for per-plugin logs
PLUGIN_LOG_DIR="$(cd "$(dirname "$0")" && pwd)/data/plugin-build-logs"
mkdir -p "$PLUGIN_LOG_DIR"

# Ensure the build directory exists, creating it if necessary.
mkdir -p "$BUILD_DIR"

# Initialize the results file with a header row.
echo "plugin_name,popularity,build_status" > "$RESULTS_FILE"
echo "plugin_name,install_count,pr_url,is_merged,build_status" > "$TSV_RESULTS_FILE"

# Initialize the debug log file with a header.
echo "Build Debug Log - $(date -u +'%Y-%m-%dT%H:%M:%SZ')" >> "$DEBUG_LOG"

# Verify required CLI tools early to avoid set -e failures
require_cmd() { command -v "$1" >/dev/null 2>&1 || { echo "Error: '$1' not found in PATH." >>"$DEBUG_LOG"; exit 1; }; }
require_cmd git
require_cmd curl
require_cmd jq
require_cmd timeout
# Ensure flock is available
# flock is optional; fall back to unlocked appends if not present
if command -v flock >/dev/null 2>&1; then
  HAS_FLOCK=1
else
  HAS_FLOCK=0
  echo "WARNING: 'flock' not found; using non-atomic appends to $RESULTS_FILE" >> "$DEBUG_LOG"
fi

# Serialize writes to RESULTS_FILE across background jobs
append_result() {
  local line="$1"
  if [ "${HAS_FLOCK:-0}" -eq 1 ]; then
    {
      flock 9
      printf '%s\n' "$line" >> "$RESULTS_FILE"
    } 9>>"$RESULTS_FILE.lock"
  else
    printf '%s\n' "$line" >> "$RESULTS_FILE"
  fi
}
# Check if Maven is installed and accessible
if command -v mvn &>/dev/null; then
    # Log Maven installation details to the debug log.
    echo "Maven is installed and accessible." >>"$DEBUG_LOG"
    echo "DEBUG: Output of 'mvn -v' after JDK 25 setup (in test-jdk-25.sh):" >> "$DEBUG_LOG"
    mvn -v >>"$DEBUG_LOG" 2>&1
else
    # Log an error message and exit if Maven is not installed.
    echo "Error: Maven is not installed or not in the PATH. Please install Maven and try again." >>"$DEBUG_LOG"
    exit 1
fi

# Define a cleanup function to remove the build directory on script exit or interruption.
cleanup() {
    echo "Cleaning up build directory..." >> "$DEBUG_LOG"
    rm -rf "$BUILD_DIR"
}
# Register the cleanup function to be called on script exit or interruption.
trap cleanup EXIT

# Check if plugins.json exists and is older than one day
if [ ! -f "$PLUGINS_JSON" ] || [ "$(find "$PLUGINS_JSON" -mtime +0)" ]; then
    # Download the latest plugins JSON file from the Jenkins update center.
    echo "Downloading $PLUGINS_JSON..."
    echo "Downloading $PLUGINS_JSON..." >> "$DEBUG_LOG"
    curl -L https://updates.jenkins.io/current/update-center.actual.json -o "$PLUGINS_JSON"
else
    # Log that the plugins JSON file is up-to-date.
    echo "plugins.json is up-to-date."
    echo "plugins.json is up-to-date." >> "$DEBUG_LOG"
fi

# Generate top-250-plugins.csv if it does not exist or is older than plugins.json
if [ ! -f "$CSV_FILE" ] || [ "$CSV_FILE" -ot "$PLUGINS_JSON" ]; then
    echo "Generating $CSV_FILE from $PLUGINS_JSON..."
    echo "Generating $CSV_FILE from $PLUGINS_JSON..." >> "$DEBUG_LOG"
    "$script_dir/get-most-popular-plugins.sh"
else
    echo "$CSV_FILE is up-to-date."
    echo "$CSV_FILE is up-to-date." >> "$DEBUG_LOG"
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
        git clone --depth 1 "$github_url" "$plugin_dir" >>"$DEBUG_LOG" 2>&1 || build_status="clone_failed"

        if [ "$build_status" == "success" ]; then
            echo "Cloned repository for $plugin_name." >>"$DEBUG_LOG"

            # Change to the plugin directory and log the result.
            cd "$plugin_dir" >>"$DEBUG_LOG" 2>&1 || {
                echo "Failed to change directory to $plugin_dir" >>"$DEBUG_LOG"
                build_status="cd_failed"
            }
            if [ "$build_status" = "success" ]; then
                echo "Reached after cd command" >>"$DEBUG_LOG"
                echo "Successfully changed directory to $plugin_dir" >>"$DEBUG_LOG"
            fi
            if [ "$build_status" == "success" ]; then
                if [ -f "pom.xml" ]; then
                    # Ensure Maven's stdout and stderr are consistently captured in the per-plugin log
                    echo "Running Maven build for $plugin_name..." >>"$DEBUG_LOG"
                    echo "Executing: timeout 20m mvn -B --no-transfer-progress -T \"${BUILD_THREADS}\" clean install -Dmaven.test.skip=true" >>"$DEBUG_LOG"
                    timeout 20m mvn -B --no-transfer-progress -T "${BUILD_THREADS}" clean install -Dmaven.test.skip=true >"$plugin_log_file" 2>&1
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
                    echo "Executing: timeout 10m $script_dir/run-gradle-build.sh $plugin_log_file --no-daemon --parallel --max-workers=\"${BUILD_THREADS}\" build" >>"$DEBUG_LOG"
                    timeout 10m "$script_dir/run-gradle-build.sh" "$plugin_log_file" --no-daemon --parallel --max-workers="${BUILD_THREADS}" build || build_status="build_failed"
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

# Phase 1: Build the set of plugins already JDK25 from TSV (those with last field TRUE)
JDK25_TRUE_SET_FILE=""
if [ -n "$TSV_FILE" ] && [ -f "$TSV_FILE" ]; then
    JDK25_TRUE_SET_FILE="$(mktemp)"
    line_number=0
    while IFS=$'\t' read -r name install_count pr_url is_merged; do
        line_number=$((line_number + 1))
        echo "Read TSV line $line_number: name='$name', is_merged='$is_merged'" >> "$DEBUG_LOG"
        # Normalize potential CR from Windows line endings in TSV fields
        name=${name%$'\r'}
        install_count=${install_count%$'\r'}
        pr_url=${pr_url%$'\r'}
        is_merged=${is_merged%$'\r'}
        # Skip header
        if [ $line_number -eq 1 ] || [ "$name" = "plugin" ] || [ "$name" = "name" ] || [ "$name" = "Name" ]; then
            continue
        fi
        status_upper=$(echo "$is_merged" | tr '[:lower:]' '[:upper:]')
        if [ "$status_upper" = "TRUE" ]; then
            echo "$name" >> "$JDK25_TRUE_SET_FILE"
            echo "Added '$name' to JDK25 TRUE set from TSV." >> "$DEBUG_LOG"
        fi
    done < "$TSV_FILE"
    echo "Built JDK25 TRUE set from TSV (file: $JDK25_TRUE_SET_FILE)" >> "$DEBUG_LOG"
else
    echo "TSV file not available; skip set will be empty." >> "$DEBUG_LOG"
fi

# Helper to check membership in the skip set
in_jdk25_true_set() {
    local plugin="$1"
    [ -n "$JDK25_TRUE_SET_FILE" ] && grep -Fxq "$plugin" "$JDK25_TRUE_SET_FILE"
}

# Phase 2: Process CSV, skipping builds for plugins already JDK25 TRUE
line_number=0
active_jobs=0
max_jobs="${PLUGIN_CONCURRENCY}"
while IFS=, read -r name popularity <&3; do
    line_number=$((line_number + 1))
    echo "Read CSV line $line_number: name='$name', popularity='$popularity'" >> "$DEBUG_LOG"

    # Skip header row in the CSV
    if [ $line_number -eq 1 ]; then
        echo "Skipping CSV header line $line_number" >> "$DEBUG_LOG"
        continue
    fi

    echo "Processing plugin '$name' from CSV line $line_number" >> "$DEBUG_LOG"

    if in_jdk25_true_set "$name"; then
        echo "Skipping build for '$name' (already using JDK25 per TSV)." >> "$DEBUG_LOG"
        append_result "$name,$popularity,success"
    else
        {
            status=$(compile_plugin "$name")
            echo "Finished processing plugin '$name' from CSV line $line_number with status: $status" >> "$DEBUG_LOG"
            append_result "$name,$popularity,$status"
        } &
        active_jobs=$((active_jobs + 1))
        if [ "$active_jobs" -ge "$max_jobs" ]; then
            if wait -n 2>/dev/null; then
                :
            else
                # Portable fallback: if wait -n is unsupported, wait for all, then reset the counter.
                # This sacrifices fine-grained throttling but preserves correctness.
                wait
                active_jobs=0
            fi
            # In the wait -n path, the completed job reduces the active count by one.
            [ "$active_jobs" -gt 0 ] && active_jobs=$((active_jobs - 1))
        fi
    fi
done 3<"$CSV_FILE" # Use file descriptor 3 for reading the CSV

# Wait for any remaining background jobs to finish
wait

echo "Finished reading $CSV_FILE after $line_number lines." >> "$DEBUG_LOG"

# Populate the extended TSV results by joining with the CSV results (no builds launched here)
if [ -n "$TSV_FILE" ] && [ -f "$TSV_FILE" ]; then
    line_number=0
    while IFS=$'\t' read -r name install_count pr_url is_merged; do
        line_number=$((line_number + 1))
        echo "Preparing extended row from TSV line $line_number: name='$name'" >> "$DEBUG_LOG"
        # Normalize potential CR from Windows line endings in TSV fields
        name=${name%$'\r'}
        install_count=${install_count%$'\r'}
        pr_url=${pr_url%$'\r'}
        is_merged=${is_merged%$'\r'}

        # Skip header
        if [ $line_number -eq 1 ] || [ "$name" = "plugin" ] || [ "$name" = "name" ] || [ "$name" = "Name" ]; then
            echo "Skipping TSV header line $line_number" >> "$DEBUG_LOG"
            continue
        fi

        status_upper=$(echo "$is_merged" | tr '[:lower:]' '[:upper:]')
        if [ "$status_upper" = "TRUE" ]; then
            build_status="success"
            echo "Plugin '$name' is merged (TRUE) in TSV; setting build_status to 'success'." >> "$DEBUG_LOG"
        else
            # Lookup build_status from RESULTS_FILE for this plugin if present
            build_status=$(awk -F',' -v n="$name" 'NR>1 && $1==n {print $3; found=1} END{ if(!found) print "" }' "$RESULTS_FILE")
            if [ -z "$build_status" ]; then
                build_status="not_in_csv"
                echo "Plugin '$name' not found in CSV results; setting build_status to 'not_in_csv'." >> "$DEBUG_LOG"
            fi
            echo "Plugin '$name' is not merged (FALSE) in TSV; using build_status from CSV: $build_status" >> "$DEBUG_LOG"
        fi

        echo "$name,$install_count,$pr_url,$is_merged,$build_status" >> "$TSV_RESULTS_FILE"
    done < "$TSV_FILE"

    echo "Finished producing extended results from $TSV_FILE after $line_number lines." >> "$DEBUG_LOG"
else
    echo "TSV file not available; skipping extended TSV results population." >> "$DEBUG_LOG"
fi

# Clean up temp set file if created
if [ -n "$JDK25_TRUE_SET_FILE" ] && [ -f "$JDK25_TRUE_SET_FILE" ]; then
    rm -f "$JDK25_TRUE_SET_FILE"
fi

# Log the completion of the script and the locations of the results and logs.
echo "Simplified build results have been saved to $RESULTS_FILE."
echo "Simplified build results have been saved to $RESULTS_FILE." >> "$DEBUG_LOG"
echo "Extended TSV build results have been saved to $TSV_RESULTS_FILE."
echo "Extended TSV build results have been saved to $TSV_RESULTS_FILE." >> "$DEBUG_LOG"
echo "Debug logs have been saved to $DEBUG_LOG."
echo "Debug logs have been saved to $DEBUG_LOG." >> "$DEBUG_LOG"
