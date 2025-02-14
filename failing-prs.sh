USER="gounthar"
START_DATE="2024-12-01T00:00:00Z"
END_DATE="2025-02-14T23:59:59Z"

gh api graphql -f query='
  query {
    search(query: "is:pr author:'$USER' updated:'$START_DATE'..'$END_DATE'", type: ISSUE, first: 100) {
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
  }
'

