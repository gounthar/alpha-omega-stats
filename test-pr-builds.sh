#!/bin/bash

set -e

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Constants
WORK_DIR="/tmp/pr-build-tests"
SUCCESS_FILE="$SCRIPT_DIR/data/consolidated/successful_builds.txt"
FAILING_PRS_FILE="$SCRIPT_DIR/data/consolidated/failing_prs.json"

# Create working directory and ensure data directory exists
mkdir -p "$WORK_DIR"
mkdir -p "$(dirname "$SUCCESS_FILE")"
rm -f "$SUCCESS_FILE"

# First ensure JDK versions are installed
./install-jdk-versions.sh

# Initialize SDKMAN
source "$HOME/.sdkman/bin/sdkman-init.sh"

# Define major Java versions we want to test (in descending order)
MAJOR_VERSIONS=("17" "11" "8")

# Get the installed JDK versions for each major version
JDK_VERSIONS=()
for major in "${MAJOR_VERSIONS[@]}"; do
    version=$("./get-jdk-versions.sh" "$major")
    if [ -n "$version" ]; then
        JDK_VERSIONS+=("$version")
    fi
done

# Function to switch JDK version
switch_jdk() {
    local version=$1
    echo "Switching to JDK $version"
    sdk use java "$version"
}

# Function to test PR
test_pr() {
    local pr_number=$1
    local repo=$2
    local branch=$3

    # Extract just the repository name (without organization)
    repo_name=$(echo "$repo" | sed 's#.*/##')

    # Clone and checkout PR
    echo "Testing PR #$pr_number in $repo"
    cd "$WORK_DIR"
    rm -rf "$repo_name"
    git clone "https://github.com/$repo.git"
    cd "$repo_name"

    # Try JDK versions in order
    for jdk in "${JDK_VERSIONS[@]}"; do
        switch_jdk "$jdk"
        echo "Attempting build with JDK $jdk"

        # Try to build
        if mvn clean verify -B; then
            echo "✓ Build successful with JDK $jdk"
            echo "https://github.com/$repo/pull/$pr_number;$jdk" >> "$SUCCESS_FILE"
            return 0
        else
            echo "✗ Build failed with JDK $jdk"
        fi
    done

    echo "✗ All JDK versions failed for PR #$pr_number"
    return 1
}

# Main processing
echo "Reading failing PRs..."
while IFS= read -r line; do
    echo "Processing line: $line"
    pr_number=$(echo "$line" | jq -r '.url' | sed 's#.*/pull/##')
    repo=$(echo "$line" | jq -r '.repository')

    echo "PR Number: $pr_number, Repository: $repo"

    if [ "$pr_number" != "null" ] && [ "$repo" != "null" ]; then
        test_pr "$pr_number" "$repo"
    else
        echo "Invalid PR data: $line"
    fi
done < <(jq -c '.[]' "$FAILING_PRS_FILE")

echo "Done. Results saved to $SUCCESS_FILE"
