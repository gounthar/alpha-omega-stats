#!/bin/bash

# Function to create the GraphQL query with pagination
create_query() {
    local after="$1"
    local cursor_param=""
    if [ ! -z "$after" ]; then
        cursor_param=", after: \"$after\""
    fi
    
    echo "query {
        search(query: \"is:pr author:gounthar updated:2024-12-01T00:00:00Z..2025-02-14T23:59:59Z status:failure\", type: ISSUE, first: 100${cursor_param}) {
            pageInfo {
                hasNextPage
                endCursor
            }
            nodes {
                ... on PullRequest {
                    title
                    url
                    commits(last: 1) {
                        nodes {
                            commit {
                                statusCheckRollup {
                                    state
                                }
                            }
                        }
                    }
                }
            }
        }
    }"
}

# Initialize an empty file for results
echo "{ \"data\": { \"search\": { \"nodes\": [" > all_results.json
first_result=true

# Initialize cursor
cursor=""
has_next_page="true"

while [ "$has_next_page" = "true" ]; do
    # Get the query result
    query=$(create_query "$cursor")
    result=$(gh api graphql -f query="$query")
    
    # Extract new cursor and hasNextPage status
    cursor=$(echo "$result" | jq -r '.data.search.pageInfo.endCursor')
    has_next_page=$(echo "$result" | jq -r '.data.search.pageInfo.hasNextPage')
    
    # Extract and append nodes
    nodes=$(echo "$result" | jq -r '.data.search.nodes')
    if [ "$first_result" = "true" ]; then
        echo "${nodes:1:${#nodes}-2}" >> all_results.json
        first_result=false
    else
        echo "," >> all_results.json
        echo "${nodes:1:${#nodes}-2}" >> all_results.json
    fi
done

# Close the JSON structure
echo "]} } }" >> all_results.json

# Pretty print the results
jq '.' all_results.json
