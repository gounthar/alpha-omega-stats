#!/bin/bash

# Exit on error
set -e

# Get the previous month and year
if [ -z "$1" ]; then
    # If no date provided, use last month
    YEAR=$(date -d "last month" +%Y)
    MONTH=$(date -d "last month" +%m)
else
    # Parse provided date (format: YYYY-MM)
    YEAR=${1%-*}
    MONTH=${1#*-}
fi

# Check if we should update Google Sheets (default: no)
UPDATE_SHEETS=${2:-false}

echo "Collecting PRs for $YEAR-$MONTH"

# Calculate start and end dates for the month
START_DATE="$YEAR-$MONTH-01"
# Calculate last day of the month
END_DATE=$(date -d "$YEAR-$MONTH-01 +1 month -1 day" +%Y-%m-%d)

echo "Date range: $START_DATE to $END_DATE"

# Set file names
BASE_NAME="prs_${YEAR}_${MONTH}"
RAW_FILE="data/monthly/${BASE_NAME}.json"
FILTERED_FILE="data/monthly/filtered_${BASE_NAME}.json"
GROUPED_FILE="data/monthly/grouped_${BASE_NAME}.json"

# Run the Go collector for the specified month
echo "Running PR collector..."
go run jenkins-pr-collector.go \
    -start "$START_DATE" \
    -end "$END_DATE" \
    -output "$RAW_FILE"

# Filter and group the PRs
echo "Filtering PRs..."
jq '.' "$RAW_FILE" > "$FILTERED_FILE"

echo "Grouping PRs..."
./group-prs.sh "$FILTERED_FILE" "plugins.json"

# Update consolidated files
echo "Updating consolidated files..."

# Function to merge JSON arrays
merge_json_arrays() {
    local files=("$@")
    local jq_args=()
    
    for file in "${files[@]}"; do
        if [ -f "$file" ]; then
            jq_args+=("--slurpfile" "arr$((${#jq_args[@]}/2))" "$file")
        fi
    done
    
    if [ ${#jq_args[@]} -eq 0 ]; then
        echo "[]"
        return
    fi
    
    jq "${jq_args[@]}" '
        reduce (inputs | .[]) as $item ([]; 
            . + if any(.[]; .url == $item.url) then [] else [$item] end
        )
    '
}

# Update all_prs.json with unique PRs from all monthly files
echo "Updating all_prs.json..."
find data/monthly -name "prs_*.json" -type f -print0 | xargs -0 cat | \
    jq -s 'add | unique_by(.url)' > "data/consolidated/all_prs.json" || {
        echo "Error: Failed to update all_prs.json" >&2
        exit 1
    }

# Extract open PRs
echo "Updating open_prs.json..."
jq '[.[] | select(.state == "OPEN")]' "data/consolidated/all_prs.json" > \
    "data/consolidated/open_prs.json"

# Extract failing PRs
echo "Extracting failing PRs..."
jq '[.[] | select(.state == "OPEN" and .checkStatus == "FAILURE")]' \
    "data/consolidated/all_prs.json" > "data/consolidated/failing_prs.json" || {
    echo "Error: Failed to extract failing PRs" >&2
    exit 1
}

# Group PRs by repository
echo "Grouping PRs by repository..."
jq -r 'group_by(.repository) | map({repository: .[0].repository, count: length}) | sort_by(-.count)' \
    "data/consolidated/all_prs.json" > "data/consolidated/prs_by_repo.json" || {
    echo "Error: Failed to group PRs by repository" >&2
    exit 1
}

# Group PRs by user
echo "Grouping PRs by user..."
jq -r 'group_by(.user) | map({user: .[0].user, count: length}) | sort_by(-.count)' \
    "data/consolidated/all_prs.json" > "data/consolidated/prs_by_user.json" || {
    echo "Error: Failed to group PRs by user" >&2
    exit 1
}

# Group PRs by plugin
echo "Grouping PRs by plugin..."
jq -r 'group_by(.pluginName) | map({plugin: .[0].pluginName, count: length}) | sort_by(-.count)' \
    "data/consolidated/all_prs.json" > "data/consolidated/prs_by_plugin.json" || {
    echo "Error: Failed to group PRs by plugin" >&2
    exit 1
}

# Group PRs by label
echo "Grouping PRs by label..."
jq -r '[.[].labels[]] | group_by(.) | map({label: .[0], count: length}) | sort_by(-.count)' \
    "data/consolidated/all_prs.json" > "data/consolidated/prs_by_label.json" || {
    echo "Error: Failed to group PRs by label" >&2
    exit 1
}

# Group PRs by check status
echo "Grouping PRs by check status..."
jq -r 'group_by(.checkStatus) | map({status: .[0].checkStatus, count: length}) | sort_by(-.count)' \
    "data/consolidated/all_prs.json" > "data/consolidated/prs_by_status.json" || {
    echo "Error: Failed to group PRs by check status" >&2
    exit 1
}

# Group PRs by state
echo "Grouping PRs by state..."
jq -r 'group_by(.state) | map({state: .[0].state, count: length}) | sort_by(-.count)' \
    "data/consolidated/all_prs.json" > "data/consolidated/prs_by_state.json" || {
    echo "Error: Failed to group PRs by state" >&2
    exit 1
}

# Archive old files (older than 6 months)
echo "Archiving old files..."
# Ensure archive directory exists and handle errors
if [ ! -d "data/archive" ]; then
    mkdir -p data/archive || {
        echo "Error: Failed to create archive directory" >&2
        exit 1
    }
fi

find data/monthly -name "prs_*.json" -type f -mtime +180 -exec mv {} data/archive/ \; || {
    echo "Warning: Some files could not be archived" >&2
}

# Update Google Sheets only if requested
if [ "$UPDATE_SHEETS" = "true" ]; then
    echo "Updating Google Sheets with consolidated data..."
    if [ -d "venv" ]; then
        source venv/bin/activate
        # Run the Python script with consolidated data
        python3 upload_to_sheets.py "data/consolidated/all_prs.json"
        deactivate
    else
        echo "Virtual environment not found. Skipping Google Sheets update."
    fi
else
    echo "Skipping Google Sheets update as requested."
fi

echo "Monthly collection completed successfully!"
echo "Files generated:"
echo "  - $RAW_FILE"
echo "  - $FILTERED_FILE"
echo "  - $GROUPED_FILE"
echo "Consolidated files updated:"
echo "  - data/consolidated/all_prs.json"
echo "  - data/consolidated/open_prs.json"
echo "  - data/consolidated/failing_prs.json"
echo "  - data/consolidated/prs_by_repo.json"
echo "  - data/consolidated/prs_by_user.json"
echo "  - data/consolidated/prs_by_plugin.json"
echo "  - data/consolidated/prs_by_label.json"
echo "  - data/consolidated/prs_by_status.json"
echo "  - data/consolidated/prs_by_state.json" 