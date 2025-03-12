#!/bin/bash

# Exit on error
set -e

# Enable debug output
# set -x

echo "Starting daily PR status update..."

# Check if consolidated files exist and create backups
for file in "all_prs.json" "open_prs.json" "failing_prs.json"; do
    if [ ! -f "data/consolidated/$file" ]; then
        echo "Error: data/consolidated/$file not found. Run collect-monthly.sh first."
        exit 1
    fi
    # Create backup with timestamp
    backup_file="data/consolidated/$file.$(date +%Y%m%d_%H%M%S).bak"
    echo "Creating backup: $backup_file"
    cp "data/consolidated/$file" "$backup_file"
done

# Create temporary files
TEMP_FILE=$(mktemp)
TEMP_VALID_JSON=$(mktemp)
PR_LIST_FILE=$(mktemp)  # Create as separate temp file

# We'll set the trap at the end of the script to prevent premature cleanup
cleanup() {
    rm -f "$TEMP_FILE" "$PR_LIST_FILE" "$TEMP_VALID_JSON"
    echo "Cleanup: Removed temporary files"
}

# Prepare for PR processing
echo "Updating PR statuses..."

echo "Preparing PR list..."
jq -r '.[] | select(.state == "OPEN") | "\(.repository) \(.number)"' "data/consolidated/open_prs.json" > "$PR_LIST_FILE"

# Verify PR list was created and has content
if [ ! -s "$PR_LIST_FILE" ]; then
    echo "Error: Failed to create PR list"
    exit 1
fi

# Make the PR processing script executable
chmod +x process_prs.sh

# Call the separate script to process PRs
echo "Calling PR processing script..."
if ! ./process_prs.sh "$PR_LIST_FILE" "$TEMP_FILE"; then
    echo "Error: PR processing script failed" >&2
    exit 1
fi

echo "PR processing completed successfully"

# Debug: Print the contents of the temporary file
echo "Validating collected PR data..."
if [ ! -s "$TEMP_FILE" ]; then
    echo "Error: No valid PR data collected. Keeping existing consolidated files."
    exit 1
fi

# Validate that temp file contains valid JSON array
echo "Converting PR data to JSON array..."
if ! jq -s '.' "$TEMP_FILE" > "$TEMP_VALID_JSON" 2>/dev/null; then
    echo "Error: Invalid JSON data collected. Keeping existing consolidated files."
    exit 1
fi

# Verify the JSON array has content
if [ ! -s "$TEMP_VALID_JSON" ] || ! jq -e '. | length > 0' "$TEMP_VALID_JSON" > /dev/null; then
    echo "Error: No valid PR data collected. Keeping existing consolidated files."
    exit 1
fi

# Function to safely update JSON files
update_json_file() {
    local source="$1"
    local temp="$2"
    local target="$3"
    local temp_output="${target}.tmp"
    
    # First verify the source file is valid JSON
    if ! jq '.' "$source" > /dev/null 2>&1; then
        echo "Error: Source file '$source' is not valid JSON or cannot be read" >&2
        return 1
    fi
    
    # Then verify the temp file is valid JSON
    if ! jq '.' "$temp" > /dev/null 2>&1; then
        echo "Error: Temp file '$temp' is not valid JSON or cannot be read" >&2
        return 1
    fi
    
    # Create a more robust update query that handles missing fields
    if ! jq -s '
        def update_pr(pr; new_pr):
            pr * {
                state: (new_pr.state // pr.state),
                statusCheckRollup: (new_pr.statusCheckRollup // pr.statusCheckRollup),
                checkStatus: (
                    if new_pr.statusCheckRollup and new_pr.statusCheckRollup[0] 
                    then new_pr.statusCheckRollup[0].state // "UNKNOWN"
                    else pr.checkStatus // "UNKNOWN"
                    end
                )
            };
        reduce (.[1] | .[]) as $new_pr (
            .[0];
            map(
                if .url == $new_pr.url
                then update_pr(.; $new_pr)
                else .
                end
            )
        )
    ' "$source" "$temp" > "$temp_output" 2>/dev/null; then
        echo "Error: Failed to update $target - jq merge operation failed" >&2
        return 1
    fi
    
    # Verify the new file is valid JSON and not empty
    if ! jq -e '. | length > 0' "$temp_output" > /dev/null 2>&1; then
        echo "Error: Generated file $target is empty or invalid"
        return 1
    fi
    
    mv "$temp_output" "$target"
    return 0
}

# Update the consolidated files with new status information
echo "Updating consolidated files with new status information..."

if ! update_json_file "data/consolidated/all_prs.json" "$TEMP_VALID_JSON" "data/consolidated/all_prs.json"; then
    # Restore from most recent backup
    latest_backup=$(ls -t data/consolidated/all_prs.json.*.bak | head -1)
    echo "Restoring all_prs.json from backup: $latest_backup"
    cp "$latest_backup" "data/consolidated/all_prs.json"
    exit 1
fi

# Update open_prs.json
echo "Updating open_prs.json..."
if ! jq '[.[] | select(.state == "OPEN")]' "data/consolidated/all_prs.json" > "data/consolidated/open_prs.json.tmp" && \
   jq -e '. | length >= 0' "data/consolidated/open_prs.json.tmp" > /dev/null; then
    echo "Error: Failed to update open_prs.json. Restoring from backup."
    latest_backup=$(ls -t data/consolidated/open_prs.json.*.bak | head -1)
    cp "$latest_backup" "data/consolidated/open_prs.json"
    exit 1
fi
mv "data/consolidated/open_prs.json.tmp" "data/consolidated/open_prs.json"

# Update failing_prs.json
echo "Updating failing_prs.json..."
if ! jq '[.[] | select(.state == "OPEN" and .checkStatus == "ERROR")]' "data/consolidated/all_prs.json" > "data/consolidated/failing_prs.json.tmp" && \
   jq -e '. | length >= 0' "data/consolidated/failing_prs.json.tmp" > /dev/null; then
    echo "Error: Failed to update failing_prs.json. Restoring from backup."
    latest_backup=$(ls -t data/consolidated/failing_prs.json.*.bak | head -1)
    cp "$latest_backup" "data/consolidated/failing_prs.json"
    exit 1
fi
mv "data/consolidated/failing_prs.json.tmp" "data/consolidated/failing_prs.json"

# Update Google Sheets with the latest data
echo "Updating Google Sheets..."
python3 upload_to_sheets.py "data/consolidated/all_prs.json" false

echo "Daily update completed successfully!"
echo "Updated files:"
echo "  - data/consolidated/all_prs.json"
echo "  - data/consolidated/open_prs.json"
echo "  - data/consolidated/failing_prs.json"
echo "Backups created:"
echo "  - data/consolidated/*.bak"
echo "Google Sheets dashboard has been updated."

# Now set the trap for cleanup only at the very end
trap cleanup EXIT INT TERM
