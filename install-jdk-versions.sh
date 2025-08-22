#!/bin/bash

: "${DEBUG_LOG:=$HOME/jdk-install-debug.log}"

# Logs a debug message with a timestamp to standard output.
#
# Arguments:
#
# * message: The message string to log.
#
# Outputs:
#
# * Writes the timestamped debug message to standard output.
#
# Example:
#
# ```bash
# log_message "Starting JDK installation"
# # Output: [DEBUG] 2025-05-15 14:23:01 - Starting JDK installation
# ```
log_message() {
    local message="$1"
    echo "[DEBUG] $(date '+%Y-%m-%d %H:%M:%S') - $message"
}

# Define the JDK installation directory at the beginning of the script
JDK_INSTALL_DIR="$HOME/.jdk-25"

# Determine the directory of the current script
# This ensures that the script can locate other scripts or files relative to its own location.
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"

# Check if SDKMAN is installed in common locations
if [[ ! -s "$HOME/.sdkman/bin/sdkman-init.sh" && ! -s "/usr/local/sdkman/bin/sdkman-init.sh" ]]; then
    echo "SDKMAN is not installed. Attempting to install SDKMAN..."
    # Install SDKMAN inline
    curl -s "https://get.sdkman.io" | bash

    # Check if SDKMAN is successfully installed
    if [[ ! -s "$HOME/.sdkman/bin/sdkman-init.sh" && ! -s "/usr/local/sdkman/bin/sdkman-init.sh" ]]; then
        echo "Failed to install SDKMAN. Please install SDKMAN manually."
        exit 1
    fi

    echo "SDKMAN installed successfully."
fi

# Temporarily modify shell options for SDKMAN compatibility
_SHELLOPTS_ORIGINAL="$-"
if [[ "${_SHELLOPTS_ORIGINAL}" == *u* ]]; then
  set +u # Disable 'exit on unset variable' for SDKMAN initialization
fi

# Initialize SDKMAN from the appropriate location
if [[ -s "$HOME/.sdkman/bin/sdkman-init.sh" ]]; then
    source "$HOME/.sdkman/bin/sdkman-init.sh"
elif [[ -s "/usr/local/sdkman/bin/sdkman-init.sh" ]]; then
    source "/usr/local/sdkman/bin/sdkman-init.sh"
fi

# After SDKMAN initialization
export SDKMAN_OFFLINE_MODE=false
sdk update > /dev/null || true

# Restore original shell options
if [[ "${_SHELLOPTS_ORIGINAL}" == *u* ]]; then
  set -u # Re-enable 'exit on unset variable' if it was originally set
fi
unset _SHELLOPTS_ORIGINAL # Clean up temporary variable

# Detects the system architecture and sets the ARCHITECTURE variable to an API-compatible value.
#
# Globals:
#   ARCHITECTURE (set): The detected and mapped architecture string for API use.
#
# Outputs:
#   Logs the detected architecture and errors to standard output.
#
# Returns:
#   Exits with status 1 if the architecture is unsupported or cannot be determined.
#
# Example:
#
#   set_architecture
#   # ARCHITECTURE will be set to "x64", "aarch64", or "riscv64" depending on the system.
set_architecture() {
    ARCHITECTURE=$(uname -m)
    log_message "Detected system architecture: $ARCHITECTURE"

    # Map architecture to the expected values for the API
    case "$ARCHITECTURE" in
        x86_64)
            ARCHITECTURE="x64";;
        aarch64)
            ARCHITECTURE="aarch64";;
        riscv64)
            ARCHITECTURE="riscv64";;
        *)
            log_message "Error: Unsupported architecture $ARCHITECTURE"
            ARCHITECTURE="unknown";;
    esac

    if [[ "$ARCHITECTURE" == "unknown" ]]; then
        log_message "Error: Unable to determine a valid architecture for the API."
        exit 1
    fi
}

# Call set_architecture before any API calls
set_architecture

# Retrieves the latest available Temurin JDK 25 early access version string for the current architecture from the Adoptium API.
#
# Returns:
#
# * The semantic version string of the latest JDK 25 early access release, or "unknown" if retrieval fails.
#
# Example:
#
# ```bash
# latest_version=$(get_latest_jdk25_version)
# echo "Latest JDK 25 version: $latest_version"
# ```
get_latest_jdk25_version() {
    local api_url="https://api.adoptium.net/v3/assets/feature_releases/25/ea?architecture=$ARCHITECTURE&heap_size=normal&image_type=jdk&jvm_impl=hotspot&os=linux&page_size=1&project=jdk&sort_order=DESC&vendor=eclipse"
    log_message "Fetching latest JDK 25 version from API: $api_url"

    # Fetch the version data from the API
    local version_data
    version_data=$(curl -s "$api_url" | jq -r '.[0].version_data.semver')

    if [[ -z "$version_data" || "$version_data" == "null" ]]; then
        log_message "Error: Unable to retrieve the latest JDK 25 version from the API."
        echo "unknown"
    else
        echo "$version_data"
    fi
}

# Checks if the installed Temurin JDK 25 is up-to-date by comparing the local version with the latest available version from the Adoptium API.
#
# Returns:
#
# * 0 if the installed JDK 25 version matches the latest available version.
# * 1 if JDK 25 is not installed or is outdated.
#
# Example:
#
# ```bash
# if is_jdk25_up_to_date; then
#   echo "JDK 25 is current."
# else
#   echo "JDK 25 needs to be installed or updated."
# fi
# ```
is_jdk25_up_to_date() {
    log_message "Checking if JDK 25 is up-to-date..."
    local installed_version_full
    local installed_version_short
    local latest_version_full
    local latest_version_short

    # Explicitly invoke the java binary from the JDK 25 installation directory
    if [[ -x "$JDK_INSTALL_DIR/bin/java" ]]; then
        log_message "Output of '$JDK_INSTALL_DIR/bin/java -version':"
        local java_version_output
        java_version_output=$("$JDK_INSTALL_DIR/bin/java" -version 2>&1)
        echo "$java_version_output" | while read -r line; do log_message "$line"; done

        # Extract version from the Runtime Environment line (e.g., Temurin-25+20...)
        # Use ERE for clarity and correct matching of digits+digits
        installed_version_full=$(echo "$java_version_output" | grep 'OpenJDK Runtime Environment' | sed -E -n 's/.*Temurin-([0-9]+)\+([0-9]+).*/\1+\2/p')
        log_message "Detected installed JDK 25 full version string: $installed_version_full"

        if [[ -z "$installed_version_full" ]]; then
             log_message "Could not extract full version string from installed JDK."
             # Fallback to simpler version detection if full string extraction fails
             installed_version_short=$("$JDK_INSTALL_DIR/bin/java" -version 2>&1 | grep -oE '"25[^"]*"' | tr -d '"')
             log_message "Fallback: Detected installed JDK 25 short version: $installed_version_short"
        else
            # Use the extracted full version for comparison
            installed_version_short="$installed_version_full"
        fi

    else
        log_message "No JDK 25 version is currently installed."
        return 1
    fi

    # Get the latest version of JDK 25 from the API
    latest_version_full=$(get_latest_jdk25_version)
    log_message "Latest available JDK 25 version from API: $latest_version_full"

    # Extract comparable part from API version (e.g., 25+20)
    # Use ERE for clarity and correct matching of digits+digits from API string
    latest_version_short=$(echo "$latest_version_full" | sed -E -n 's/([0-9]+)\.[0-9]+\.[0-9]+-beta\+([0-9]+)\..*/\1+\2/p')
    log_message "Extracted comparable latest version: $latest_version_short"

    if [[ -z "$latest_version_short" ]]; then
        log_message "Could not extract comparable version string from API version."
        # Fallback to comparing full API string if extraction fails
        latest_version_short="$latest_version_full"
    fi

    log_message "Comparing installed version '$installed_version_short' with latest version '$latest_version_short'"

    if [[ "$installed_version_short" == "$latest_version_short" ]]; then
        log_message "JDK 25 is up-to-date (version $installed_version_full). Skipping installation."
        return 0
    else
        log_message "Installed JDK 25 version ($installed_version_full) is not up-to-date. Latest version is $latest_version_full."
        return 1
    fi
}

# Downloads and installs the latest Temurin JDK 25 early access binary, updating environment variables and verifying installation.
#
# Globals:
#   JDK_INSTALL_DIR: Directory where JDK 25 will be installed.
#   ARCHITECTURE: System architecture string for API queries.
#
# Outputs:
#   Logs progress and errors to standard output.
#
# Example:
#
#   install_temurin_jdk25 || true
log_message "install-jdk-versions.sh finished"
#   # Installs or updates Temurin JDK 25 early access in $HOME/.jdk-25 and updates PATH and JAVA_HOME.
install_temurin_jdk25() {
    log_message "Starting installation of Temurin JDK 25..."
    if is_jdk25_up_to_date; then
        log_message "Skipping installation as JDK 25 is already up-to-date."
        return
    fi

    # Update the installation directory to a user-writable location
    log_message "JDK installation directory set to: $JDK_INSTALL_DIR"

    local api_url="https://api.adoptium.net/v3/assets/feature_releases/25/ea?architecture=$ARCHITECTURE&heap_size=normal&image_type=jdk&jvm_impl=hotspot&os=linux&page_size=1&project=jdk&sort_order=DESC&vendor=eclipse"
    local download_url

    # Fetch the latest JDK 25 early access binary URL
    log_message "Fetching latest JDK 25 binary URL from API: $api_url"
    download_url=$(curl -s "$api_url" | jq -r '.[0].binaries[0].package.link')

    if [ -z "$download_url" ]; then
        log_message "Error: Unable to fetch Temurin JDK 25 early access binary URL."
        exit 1
    fi

    log_message "Downloading JDK 25 binary from: $download_url"
    curl -L "$download_url" -o /tmp/jdk-25.tar.gz
    mkdir -p "$JDK_INSTALL_DIR"
    tar -xzf /tmp/jdk-25.tar.gz -C "$JDK_INSTALL_DIR" --strip-components=1

    # Update PATH to include the new JDK
    export PATH="$JDK_INSTALL_DIR/bin:$PATH"
    log_message "Temurin JDK 25 early access installed successfully."

    # Call the function to update PATH after installation
    update_path_for_jdk25

    # Verify the JDK installation by running a simple Java command
    verify_jdk_installation
}

# Updates PATH and JAVA_HOME to prioritize the installed JDK 25 and refreshes the shell environment.
#
# Ensures the JDK 25 binary directory is at the front of PATH and sets JAVA_HOME to the JDK 25 installation directory.
# Refreshes the shell's command lookup table and logs the output of 'java -version' to confirm the correct Java binary is active.
#
# Globals:
#   JDK_INSTALL_DIR - Path to the JDK 25 installation directory.
#
# Outputs:
#   Logs debug messages and the output of 'java -version' to standard output.
#
# Example:
# Updates environment variables to use the installed Temurin JDK 25.
#
# Ensures that the JDK 25 `bin` directory is at the front of the `PATH` and sets `JAVA_HOME` to the JDK 25 installation directory. Refreshes the shell's command lookup and logs the output of `java -version` to the debug log for verification.
#
# Globals:
#   JDK_INSTALL_DIR: Directory where Temurin JDK 25 is installed.
#   DEBUG_LOG: Path to the debug log file.
#
# Outputs:
#   Appends the output of `java -version` to the debug log file.
#
# Example:
#   update_path_for_jdk25
update_path_for_jdk25() {
    if [[ ":$PATH:" != *":$JDK_INSTALL_DIR/bin:"* ]]; then
        export PATH="$JDK_INSTALL_DIR/bin:$PATH"
        log_message "Updated PATH to include JDK 25 installation directory."
    fi

    export JAVA_HOME="$JDK_INSTALL_DIR"
    log_message "Set JAVA_HOME to JDK 25 installation directory: $JAVA_HOME"

    # Ensure the current shell session uses the updated PATH
    hash -r
    log_message "Refreshed shell environment to use updated PATH."

    # Verify the java command points to the correct binary
    echo "DEBUG: Output of 'java -version' after PATH update (in update_path_for_jdk25 from install-jdk-versions.sh):" >> "$DEBUG_LOG"
    java -version >> "$DEBUG_LOG" 2>&1
}

# Verifies that the installed Temurin JDK 25 binary is present and reports the correct version.
#
# Globals:
# * JDK_INSTALL_DIR: Directory where Temurin JDK 25 is installed.
#
# Outputs:
# * Prints verification status messages to STDOUT.
#
# Returns:
# * Exits with status 1 if verification fails.
#
# Example:
#
# ```bash
# verify_jdk_installation
# Verifies that Temurin JDK 25 is correctly installed by checking the output of the java -version command.
#
# Globals:
#   JDK_INSTALL_DIR - Directory where Temurin JDK 25 is installed.
#   DEBUG_LOG - Path to the debug log file.
#
# Outputs:
#   Appends verification steps and java -version output to the debug log file.
#
# Returns:
#   None. Logs an error if verification fails but does not exit or return a value.
#
# Example:
#   verify_jdk_installation
verify_jdk_installation() {
    echo "DEBUG: Verifying Temurin JDK 25 installation (in verify_jdk_installation from install-jdk-versions.sh):" >> "$DEBUG_LOG"
    echo "DEBUG: Running: $JDK_INSTALL_DIR/bin/java -version" >> "$DEBUG_LOG"
    local version_output
    version_output=$("$JDK_INSTALL_DIR/bin/java" -version 2>&1)
    echo "$version_output" >> "$DEBUG_LOG"

    if echo "$version_output" | grep -qE 'version "25'; then
        echo "DEBUG: Temurin JDK 25 installation verified successfully (from verify_jdk_installation)." >> "$DEBUG_LOG"
    else
        echo "DEBUG: Error: Temurin JDK 25 installation verification failed (from verify_jdk_installation)." >> "$DEBUG_LOG"
        # exit 1
    fi
}

# Declare the JDK versions you're interested in
# These are the JDK versions that the script will attempt to install.
declare -a jdk_versions=("8" "11" "17" "21")

# Verify that sdk is on PATH
# This ensures that the `sdk` command is available before proceeding.
if ! command -v sdk &>/dev/null; then
    echo "Error: sdk is not installed or not in the PATH. Please install SDKMAN and try again."
    exit 1
fi

# Loop through each JDK version
# For each version, determine the appropriate identifier and install it using SDKMAN.
for version in "${jdk_versions[@]}"; do
    # Define the pattern to match the JDK version in the SDKMAN list
    pattern=" $version\.0.*-tem"

    # Retrieve the identifier for the JDK version using SDKMAN
    # The identifier is extracted by listing available JDKs, filtering with grep, and selecting the last field.
    identifier=$(
      PAGER=cat sdk list java \
        | grep -E -- "$pattern" \
        | awk '{print $NF}' \
        | head -n1
    )

    # Check if a valid identifier was found
    if [ -n "$identifier" ]; then
        echo "Installing JDK version $version with identifier $identifier"
        # Disable nounset for SDKMAN install to avoid unbound variable error
        set +u
        yes | sdk install java "$identifier"
        set -u
    else
        # Log a message if no suitable JDK version is found
        echo "No suitable JDK version found for $version"
    fi
done

# Call the function to install Temurin JDK 25
install_temurin_jdk25 || true
log_message "install-jdk-versions.sh finished"
