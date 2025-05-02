#!/bin/bash

# Determine the directory of the current script
# This ensures that the script can locate other scripts or files relative to its own location.
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"

# Check if SDKMAN is installed
# SDKMAN is required to manage and install JDK versions.
if [[ ! -s "$HOME/.sdkman/bin/sdkman-init.sh" ]]; then
    echo "SDKMAN is not installed. Attempting to install SDKMAN..."
    # Attempt to install SDKMAN by calling the install-sdk.sh script from the same directory
    if [[ -x "$script_dir/install-sdk.sh" ]]; then
        "$script_dir/install-sdk.sh"
        # Check if SDKMAN is successfully installed after running the installation script
        if [[ ! -s "$HOME/.sdkman/bin/sdkman-init.sh" ]]; then
            echo "Failed to install SDKMAN. Please install SDKMAN manually."
            exit 1
        fi
    else
        # Exit if the install-sdk.sh script is not found or not executable
        echo "install-sdk.sh script not found in $script_dir"
        exit 1
    fi
fi

# Initialize SDKMAN
# This loads SDKMAN into the current shell session, making its commands available.
source "$HOME/.sdkman/bin/sdkman-init.sh"

# Function to fetch and install Temurin JDK 25 early access binaries
install_temurin_jdk25() {
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
            echo "Error: Unsupported architecture $ARCHITECTURE"
            exit 1;;
    esac

    # Update the installation directory to a user-writable location
    JDK_INSTALL_DIR="$HOME/.jdk-25"

    local api_url="https://api.adoptium.net/v3/assets/feature_releases/25/ea?architecture=$ARCHITECTURE&heap_size=normal&image_type=jdk&jvm_impl=hotspot&os=linux&page_size=1&project=jdk&sort_order=DESC&vendor=eclipse"
    local download_url

    # Fetch the latest JDK 25 early access binary URL
    download_url=$(curl -s "$api_url" | jq -r '.[0].binaries[0].package.link')

    if [ -z "$download_url" ]; then
        echo "Error: Unable to fetch Temurin JDK 25 early access binary URL."
        exit 1
    fi

    # Download and extract the JDK binary
    echo "Downloading Temurin JDK 25 early access binary..."
    curl -L "$download_url" -o /tmp/jdk-25.tar.gz
    mkdir -p "$JDK_INSTALL_DIR"
    tar -xzf /tmp/jdk-25.tar.gz -C "$JDK_INSTALL_DIR" --strip-components=1

    # Update PATH to include the new JDK
    export PATH="$JDK_INSTALL_DIR/bin:$PATH"
    echo "Temurin JDK 25 early access installed successfully."

    # Verify the JDK installation by running a simple Java command
    verify_jdk_installation
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