#!/bin/bash

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

# Initialize SDKMAN from the appropriate location
if [[ -s "$HOME/.sdkman/bin/sdkman-init.sh" ]]; then
    source "$HOME/.sdkman/bin/sdkman-init.sh"
elif [[ -s "/usr/local/sdkman/bin/sdkman-init.sh" ]]; then
    source "/usr/local/sdkman/bin/sdkman-init.sh"
fi

# Fix the function to fetch the latest available version of Temurin JDK 25 from the API
get_latest_jdk25_version() {
    local api_url="https://api.adoptium.net/v3/assets/feature_releases/25/ea?architecture=$ARCHITECTURE&heap_size=normal&image_type=jdk&jvm_impl=hotspot&os=linux&page_size=1&project=jdk&sort_order=DESC&vendor=eclipse"
    curl -s "$api_url" | jq -r '.[0].version_data.semver' || echo "unknown"
}

# Enhanced logging for better debugging
log_message() {
    local message="$1"
    echo "[DEBUG] $(date '+%Y-%m-%d %H:%M:%S') - $message"
}

# Enhanced function to detect the installed JDK 25 version with detailed logging
is_jdk25_up_to_date() {
    log_message "Checking if JDK 25 is up-to-date..."
    local installed_version
    local latest_version

    # Capture the output of java -version for debugging
    log_message "Output of 'java -version':"
    java -version 2>&1 | while read -r line; do log_message "$line"; done

    # Check if the java command points to the JDK 25 installation directory
    if [[ $(java -version 2>&1 | grep -oE '"25[^"]*"' | tr -d '"') ]]; then
        installed_version=$(java -version 2>&1 | grep -oE '"25[^"]*"' | tr -d '"')
        log_message "Detected installed JDK 25 version via java command: $installed_version"
    elif [[ -x "$JDK_INSTALL_DIR/bin/java" ]]; then
        log_message "Output of '$JDK_INSTALL_DIR/bin/java -version':"
        $JDK_INSTALL_DIR/bin/java -version 2>&1 | while read -r line; do log_message "$line"; done
        installed_version=$($JDK_INSTALL_DIR/bin/java -version 2>&1 | grep -oE '"25[^"]*"' | tr -d '"')
        log_message "Detected installed JDK 25 version via JDK_INSTALL_DIR: $installed_version"
    else
        log_message "No JDK 25 version is currently installed."
        return 1
    fi

    # Get the latest version of JDK 25 from the API
    latest_version=$(get_latest_jdk25_version)
    log_message "Latest available JDK 25 version from API: $latest_version"

    if [ "$installed_version" == "$latest_version" ]; then
        log_message "JDK 25 is up-to-date (version $installed_version). Skipping installation."
        return 0
    else
        log_message "Installed JDK 25 version ($installed_version) is not up-to-date. Latest version is $latest_version."
        return 1
    fi
}

# Enhanced function to fetch and install Temurin JDK 25 early access binaries with logging
install_temurin_jdk25() {
    log_message "Starting installation of Temurin JDK 25..."
    if is_jdk25_up_to_date; then
        log_message "Skipping installation as JDK 25 is already up-to-date."
        return
    fi

    # Detect the system architecture dynamically
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
            exit 1;;
    esac

    # Update the installation directory to a user-writable location
    JDK_INSTALL_DIR="$HOME/.jdk-25"
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

# Ensure the PATH and JAVA_HOME are updated to prioritize the newly installed JDK 25
update_path_for_jdk25() {
    if [[ ":$PATH:" != *":$JDK_INSTALL_DIR/bin:"* ]]; then
        export PATH="$JDK_INSTALL_DIR/bin:$PATH"
        log_message "Updated PATH to include JDK 25 installation directory."
    fi

    export JAVA_HOME="$JDK_INSTALL_DIR"
    log_message "Set JAVA_HOME to JDK 25 installation directory: $JAVA_HOME"
}

# Verify the JDK installation by running a simple Java command
verify_jdk_installation() {
    echo "Verifying Temurin JDK 25 installation..."
    if java -version 2>&1 | grep -qE "version \"25"; then
        echo "Temurin JDK 25 installation verified successfully."
    else
        echo "Error: Temurin JDK 25 installation verification failed."
        exit 1
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
        # Install the JDK version using SDKMAN
        yes | sdk install java "$identifier"
    else
        # Log a message if no suitable JDK version is found
        echo "No suitable JDK version found for $version"
    fi
done

# Call the function to install Temurin JDK 25
install_temurin_jdk25