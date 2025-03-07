#!/bin/bash

# Source the required scripts
source check-env.sh

# Function to create the GraphQL query with pagination
# Arguments:
#   $1 - The cursor for pagination (optional)
# Returns:
#   The GraphQL query string
create_query() {
    local after="$1"
    local cursor_param=""
    if [ ! -z "$after" ]; then
        cursor_param=", after: \"$after\""
    fi

    cat <<EOF
query {
    search(query: "is:pr is:open author:gounthar updated:2024-12-01T00:00:00Z..2025-02-25T23:59:59Z", type: ISSUE, first: 100${cursor_param}) {
        pageInfo {
            hasNextPage
            endCursor
        }
        nodes {
            ... on PullRequest {
                title
                url
                merged
                commits(last: 1) {
                    nodes {
                        commit {
                            statusCheckRollup {
                                state
                            }
                        }
                    }
                }
                createdAt
                updatedAt
                author {
                    login
                }
                repository {
                    name
                }
                labels(first: 10) {
                    nodes {
                        name
                    }
                }
                body
            }
        }
    }
}
EOF
}

# Function to check the rate limit status of the GitHub API
check_rate_limit() {
  # Make a request to the GitHub API to get the rate limit status
  rate_limit_response=$(curl -s -H "Authorization: token $GITHUB_TOKEN" "https://api.github.com/rate_limit")

  # Extract the remaining, reset, and limit values from the response using jq
  remaining=$(echo "$rate_limit_response" | jq '.resources.graphql.remaining')
  reset=$(echo "$rate_limit_response" | jq '.resources.graphql.reset')
  limit=$(echo "$rate_limit_response" | jq '.resources.graphql.limit')

  # Check if the remaining requests are less than 5% of the limit
  if [ "$remaining" -lt $((limit / 20)) ]; then
    # Calculate the current time and the wait time until the rate limit resets
    current_time=$(date +%s)
    wait_time=$((reset - current_time))
    end_time=$(date -d "@$reset" +"%H:%M")

    # Log an error message indicating the rate limit has been exceeded
    error "API rate limit exceeded for GraphQL. Please wait $wait_time seconds before retrying. Come back at $end_time."

    # Initialize progress bar
    start_time=$(date +%s)
    while [ $((current_time + wait_time)) -gt "$(date +%s)" ]; do
      # Calculate the elapsed time and progress percentage
      elapsed_time=$(( $(date +%s) - start_time ))
      progress=$(printf "%d" $(( 100 * elapsed_time / wait_time )))

      # Print the progress bar
      printf "\rProgress: [%-50s] %d%%" "$(printf "%0.s#" "$(seq 1 "$(( progress / 2 ))")")" "$progress"
      sleep 1
    done

    # Log a message indicating the wait time is completed
    info -e "\nWait time completed. Resuming..."
  fi
}

# Initialize an empty file for results
# Creates a JSON structure to store the results
echo "{ \"data\": { \"search\": { \"nodes\": [" > all_results.json
first_result=true

# Initialize cursor and pagination status
cursor=""
has_next_page="true"
# Initialize an empty array to collect all nodes
all_nodes="[]"

# Loop to fetch all pages of results
while [ "$has_next_page" = "true" ]; do
    check_rate_limit
    # Get the query result
    query=$(create_query "$cursor")
    result=$(gh api graphql -f query="$query")

    # Extract new cursor and hasNextPage status
    cursor=$(echo "$result" | jq -r '.data.search.pageInfo.endCursor')
    debug "cursor now is $cursor"
    has_next_page=$(echo "$result" | jq -r '.data.search.pageInfo.hasNextPage')
    debug "Do we have a next page? $has_next_page"

    # Extract and filter nodes from the GraphQL query result
    total_nodes=$(echo "$result" | jq '.data.search.nodes | length')
    nodes=$(echo "$result" | jq '.data.search.nodes | map(select(.commits.nodes[0].commit.statusCheckRollup.state == "FAILURE" and .merged == false))')
    filtered_nodes=$(echo "$nodes" | jq length)
    removed_nodes=$((total_nodes - filtered_nodes))
    info "Number of interesting nodes fetched: $filtered_nodes"
    info "Number of nodes removed: $removed_nodes"

    # Concatenate the new nodes with the existing ones using jq
    all_nodes=$(echo "$all_nodes" "$nodes" | jq -s '.[0] + .[1]')
done

# Create the final JSON structure and save it to the file
echo "$all_nodes" | jq '{"data": {"search": {"nodes": .}}}' > all_results.json
