#!/bin/bash

# Script to run the Go program for finding JUnit 5 migration PRs
# This script integrates with the existing workflow and handles environment setup

# Exit on error
set -e

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go before running this script."
    exit 1
fi

# Check if GITHUB_TOKEN is set
if [ -z "$GITHUB_TOKEN" ]; then
    # Try to use GitHub CLI token if available
    if command -v gh &> /dev/null; then
        echo "GITHUB_TOKEN not set, attempting to use GitHub CLI token..."
        export GITHUB_TOKEN=$(gh auth token)
        if [ -z "$GITHUB_TOKEN" ]; then
            echo "Error: Could not get token from GitHub CLI."
            echo "Please set GITHUB_TOKEN environment variable with: export GITHUB_TOKEN=your_github_token"
            exit 1
        fi
    else
        echo "Error: GITHUB_TOKEN environment variable is not set and GitHub CLI is not available."
        echo "Please set it with: export GITHUB_TOKEN=your_github_token"
        exit 1
    fi
fi

# Create required directories
mkdir -p data/junit5

# Set default parameters
OUTPUT_DIR="data/junit5"
CANDIDATE_FILE="junit5_candidate_prs.txt"
EXISTING_FILE="junit5_pr_urls.txt"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        --output-dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --candidate-file)
            CANDIDATE_FILE="$2"
            shift 2
            ;;
        --existing-file)
            EXISTING_FILE="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "Finding JUnit 5 migration PRs..."
echo "Output directory: $OUTPUT_DIR"
echo "Candidate file: $CANDIDATE_FILE"
echo "Existing file: $EXISTING_FILE"

# Navigate to the directory containing the Go program
cd "$(dirname "$0")/cmd/find-junit5-prs"

# Download dependencies if needed
echo "Downloading dependencies..."
go mod tidy

# Build and run the program
echo "Building and running the JUnit 5 PR finder..."
go run main.go --output-dir="../../$OUTPUT_DIR" --candidate-file="../../$CANDIDATE_FILE" --existing-file="../../$EXISTING_FILE"

# Return to the original directory
cd - > /dev/null

# Process the results
CANDIDATE_PATH="$OUTPUT_DIR/$CANDIDATE_FILE"
EXISTING_PATH="$OUTPUT_DIR/$EXISTING_FILE"

if [ -f "$CANDIDATE_PATH" ]; then
    NEW_PR_COUNT=$(grep -c "^https://" "$CANDIDATE_PATH" || true)
    echo " Found $NEW_PR_COUNT potential JUnit 5 migration PR candidates"
    
    # Check if there are new PRs not in the existing file
    if [ -f "$EXISTING_PATH" ]; then
        NEW_PRS=$(grep -v -f "$EXISTING_PATH" "$CANDIDATE_PATH" | grep "^https://" || true)
        NEW_PR_COUNT=$(echo "$NEW_PRS" | grep -c "^https://" || true)
        
        if [ "$NEW_PR_COUNT" -gt 0 ]; then
            echo " Found $NEW_PR_COUNT new PR candidates not already in $EXISTING_PATH"
            echo "To add all new PRs to $EXISTING_PATH, run:"
            echo "  grep -v -f \"$EXISTING_PATH\" \"$CANDIDATE_PATH\" | grep \"^https://\" >> \"$EXISTING_PATH\""
        else
            echo " No new PR candidates found"
        fi
    fi
else
    echo " Failed to generate candidate PR file"
fi

echo "Script completed successfully!"
