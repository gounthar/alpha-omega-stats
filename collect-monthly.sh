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
find data/monthly -name "prs_*.json" -type f | xargs cat | \
    jq -s 'add | unique_by(.url)' > "data/consolidated/all_prs.json"

# Extract open PRs
echo "Updating open_prs.json..."
jq '[.[] | select(.state == "OPEN")]' "data/consolidated/all_prs.json" > \
    "data/consolidated/open_prs.json"

# Extract failing PRs
echo "Updating failing_prs.json..."
jq '[.[] | select(.state == "OPEN" and .checkStatus == "FAILURE")]' \
    "data/consolidated/all_prs.json" > "data/consolidated/failing_prs.json"

# Archive old files (older than 6 months)
echo "Archiving old files..."
find data/monthly -name "prs_*.json" -type f -mtime +180 -exec mv {} data/archive/ \;

# Update Google Sheets only if requested
if [ "$UPDATE_SHEETS" = "true" ]; then
    echo "Updating Google Sheets with consolidated data..."
    if [ -d "venv" ]; then
        source venv/bin/activate
        # Run the Python script with consolidated data
        python3 upload_to_sheets.py "data/consolidated/all_prs.json" "$FAILING_PRS_ERROR"
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