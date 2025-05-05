#!/bin/bash

# Exit on error
set -e

# Create necessary directories if they don't exist
mkdir -p data/monthly data/consolidated data/archive data/backup

# Function to collect data for a specific month with retries
collect_month() {
    local year=$1
    local month=$2
    local max_retries=3
    local retry=0
    local success=false

    # Check if we're trying to collect data for a future month
    current_year=$(date +%Y)
    current_month=$(date +%m)
    if [ "$year" -gt "$current_year" ] || ([ "$year" -eq "$current_year" ] && [ "$month" -gt "$current_month" ]); then
        echo "Skipping future month $year-$month"
        return 0
    fi

    # Skip months before July 2024
    if [ "$year" -lt "2024" ] || ([ "$year" -eq "2024" ] && [ "$month" -lt "07" ]); then
        echo "Skipping month before July 2024: $year-$month"
        return 0
    fi

    while [ $retry -lt $max_retries ] && [ "$success" = false ]; do
        echo "Collecting data for $year-$month (attempt $((retry + 1))/$max_retries)"
        if ./collect-monthly.sh "$year-$month" false; then
            success=true
            echo "Successfully collected data for $year-$month"
        else
            retry=$((retry + 1))
            if [ $retry -lt $max_retries ]; then
                wait_time=$((retry * 120))  # Wait 2 minutes, then 4 minutes, then 6 minutes
                echo "Failed to collect data for $year-$month. Waiting ${wait_time} seconds before retry..."
                sleep $wait_time
            fi
        fi
    done

    if [ "$success" = false ]; then
        echo "Failed to collect data for $year-$month after $max_retries attempts"
        return 1
    fi

    # Wait between successful collections to avoid rate limits
    echo "Waiting 60 seconds before next collection..."
    sleep 60
}

# Backup existing data
echo "Backing up existing data..."
backup_dir="data/backup_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$backup_dir"
if [ -d "data/monthly" ]; then
    cp -r data/monthly/* "$backup_dir/" 2>/dev/null || true
fi
if [ -d "data/consolidated" ]; then
    cp -r data/consolidated/* "$backup_dir/" 2>/dev/null || true
fi

# Get current year and month
current_year=$(date +%Y)
current_month=$(date +%m)

# Start from July 2024
start_year=2024
start_month=7

# Process each month from July 2024 up to current month
year=$start_year
month=$start_month

while [ "$year" -lt "$current_year" ] || ([ "$year" -eq "$current_year" ] && [ "$month" -le "$current_month" ]); do
    # Format month with leading zero if needed
    month_padded=$(printf "%02d" "$month")
    
    collect_month "$year" "$month_padded" || {
        echo "Failed to collect data for $year-$month_padded, continuing with next month..."
    }

    # Move to next month
    month=$((month + 1))
    if [ "$month" -gt 12 ]; then
        month=1
        year=$((year + 1))
    fi
done

# Now that all data is collected, do one final collection for the current month
# to update Google Sheets with all consolidated data
echo "Updating Google Sheets with all consolidated data..."
if ! ./collect-monthly.sh "$current_year-$current_month" true; then
    echo "Warning: Failed to update Google Sheets with consolidated data"
fi

echo "Data collection completed. Check the logs for any errors." 