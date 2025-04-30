#!/bin/bash
set -euo pipefail

# Determine the directory of the current script
script_dir=$(dirname "$0")

# Check if SDKMAN is installed
if [[ ! -s "$HOME/.sdkman/bin/sdkman-init.sh" ]]; then
    echo "SDKMAN is not installed. Attempting to install SDKMAN..."
    # Attempt to install SDKMAN by calling install-sdk.sh from the same directory
    if [[ -x "$script_dir/install-sdk.sh" ]]; then
        "$script_dir/install-sdk.sh"
        # Check if SDKMAN is successfully installed
        if [[ ! -s "$HOME/.sdkman/bin/sdkman-init.sh" ]]; then
            echo "Failed to install SDKMAN. Please install SDKMAN manually."
            exit 1
        fi
    else
        echo "install-sdk.sh script not found in $script_dir"
        exit 1
    fi
fi

# Initialize SDKMAN
source "$HOME/.sdkman/bin/sdkman-init.sh"

# Declare the JDK versions you're interested in
declare -a jdk_versions=("8" "11" "17" "21" "25")

# Loop through each JDK version
for version in "${jdk_versions[@]}"; do
    case "$version" in
      25) pattern=" $version\.ea" ;;
      *)  pattern=" $version\.0.*-tem" ;;
    esac

    identifier=$(
      PAGER=cat sdk list java \
        | grep -E "$pattern" \
        | awk '{print $NF}' \
        | head -n1
    )

    if [ -n "$identifier" ]; then
        echo "Installing JDK version $version with identifier $identifier"
        # Install the JDK version using SDKMAN
        yes | sdk install java "$identifier"
    else
        echo "No suitable JDK version found for $version"
    fi
done

