#!/bin/sh

if [ -z "$GITHUB_TOKEN" ]; then
    echo "Error: GITHUB_TOKEN environment variable is required"
    exit 1
fi

END_DATE=$(date +%Y-%m-%d)
./jenkins-pr-collector -github-token "$GITHUB_TOKEN" -start-date "$START_DATE" -end-date "$END_DATE"
