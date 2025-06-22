#!/bin/bash

set -e

CSV_FILE="top-250-plugins.csv"

# Check file exists
if [ ! -f "$CSV_FILE" ]; then
  echo "Error: $CSV_FILE does not exist."
  exit 1
fi

# Check file is not empty
if [ ! -s "$CSV_FILE" ]; then
  echo "Error: $CSV_FILE is empty."
  exit 1
fi

# Check header
header=$(head -n 1 "$CSV_FILE")
if [ "$header" != "name,popularity" ]; then
  echo "Error: CSV header is incorrect. Found: $header"
  exit 1
fi

# Check row count (should be 251: 1 header + 250 plugins)
row_count=$(wc -l < "$CSV_FILE")
if [ "$row_count" -ne 251 ]; then
  echo "Error: Expected 251 rows (1 header + 250 plugins), found $row_count."
  exit 1
fi

# Check all popularity values are numeric
tail -n +2 "$CSV_FILE" | awk -F, '{if ($2 !~ /^[0-9]+$/) {print "Non-numeric popularity in line: " NR+1; exit 2}}'

echo "Validation passed: $CSV_FILE is well-formed."
