package github

// GraphQL queries for fetching user profile data

// UserProfileQuery fetches basic user information
const UserProfileQuery = `
query($username: String!) {
  user(login: $username) {
    id
    login
    name
    bio
    company
    location
    email
    websiteUrl
    twitterUsername
    createdAt
    updatedAt
    avatarUrl
    followers {
      totalCount
    }
    following {
      totalCount
    }
    repositories {
      totalCount
    }
    contributionsCollection {
      totalCommitContributions
      totalIssueContributions
      totalPullRequestContributions
      totalPullRequestReviewContributions
      contributionYears
    }
    organizations(first: 100) {
      nodes {
        login
        name
        description
        url
        avatarUrl
        createdAt
      }
    }
  }
}`

// UserRepositoriesQuery fetches user's repositories with pagination
const UserRepositoriesQuery = `
query($username: String!, $first: Int!, $after: String) {
  user(login: $username) {
    repositories(
      first: $first
      after: $after
      orderBy: {field: UPDATED_AT, direction: DESC}
      ownerAffiliations: [OWNER, COLLABORATOR, ORGANIZATION_MEMBER]
    ) {
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        name
        nameWithOwner
        description
        url
        isPrivate
        isFork
        isArchived
        createdAt
        updatedAt
        pushedAt
        stargazerCount
        forkCount
        watchers {
          totalCount
        }
        issues {
          totalCount
        }
        pullRequests {
          totalCount
        }
        primaryLanguage {
          name
          color
        }
        languages(first: 10, orderBy: {field: SIZE, direction: DESC}) {
          nodes {
            name
            color
          }
          edges {
            size
          }
        }
        repositoryTopics(first: 20) {
          nodes {
            topic {
              name
            }
          }
        }
        licenseInfo {
          name
          spdxId
        }
        diskUsage
        owner {
          login
          ... on Organization {
            name
            description
          }
        }
      }
    }
  }
}`

// UserContributionsQuery fetches contribution statistics
const UserContributionsQuery = `
query($username: String!, $from: DateTime!, $to: DateTime!) {
  user(login: $username) {
    contributionsCollection(from: $from, to: $to) {
      totalCommitContributions
      totalIssueContributions
      totalPullRequestContributions
      totalPullRequestReviewContributions
      contributionCalendar {
        totalContributions
        weeks {
          contributionDays {
            contributionCount
            date
          }
        }
      }
      commitContributionsByRepository(maxRepositories: 100) {
        repository {
          nameWithOwner
          primaryLanguage {
            name
          }
          owner {
            login
          }
        }
        contributions(first: 100) {
          nodes {
            commitCount
            user {
              login
            }
          }
        }
      }
      issueContributionsByRepository(maxRepositories: 100) {
        repository {
          nameWithOwner
        }
        contributions(first: 100) {
          nodes {
            issueCount
          }
        }
      }
      pullRequestContributionsByRepository(maxRepositories: 100) {
        repository {
          nameWithOwner
          owner {
            login
          }
        }
        contributions(first: 100) {
          nodes {
            pullRequestCount
          }
        }
      }
    }
  }
}`

// UserOrganizationsQuery fetches detailed organization information
const UserOrganizationsQuery = `
query($username: String!) {
  user(login: $username) {
    organizations(first: 100) {
      nodes {
        login
        name
        description
        url
        avatarUrl
        createdAt
        repositories(first: 100, affiliations: [OWNER, COLLABORATOR]) {
          nodes {
            nameWithOwner
            primaryLanguage {
              name
            }
            stargazerCount
            createdAt
            updatedAt
          }
        }
      }
    }
    repositoriesContributedTo(
      first: 100
      includeUserRepositories: false
      orderBy: {field: UPDATED_AT, direction: DESC}
    ) {
      nodes {
        nameWithOwner
        owner {
          login
          ... on Organization {
            name
            description
            url
          }
        }
        primaryLanguage {
          name
        }
        stargazerCount
        createdAt
        updatedAt
      }
    }
  }
}`

// RepositoryDetailsQuery fetches detailed information about a specific repository
const RepositoryDetailsQuery = `
query($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    id
    name
    nameWithOwner
    description
    url
    isPrivate
    isFork
    isArchived
    createdAt
    updatedAt
    pushedAt
    stargazerCount
    forkCount
    watchers {
      totalCount
    }
    issues(states: [OPEN]) {
      totalCount
    }
    pullRequests(states: [OPEN]) {
      totalCount
    }
    primaryLanguage {
      name
      color
    }
    languages(first: 20, orderBy: {field: SIZE, direction: DESC}) {
      totalSize
      nodes {
        name
        color
      }
      edges {
        size
      }
    }
    repositoryTopics(first: 20) {
      nodes {
        topic {
          name
        }
      }
    }
    licenseInfo {
      name
      spdxId
    }
    diskUsage
    collaborators(first: 100) {
      totalCount
      nodes {
        login
        name
      }
    }
    releases(first: 10, orderBy: {field: CREATED_AT, direction: DESC}) {
      nodes {
        name
        tagName
        createdAt
        isPrerelease
      }
    }
    defaultBranchRef {
      target {
        ... on Commit {
          history(first: 100, author: {id: $authorId}) {
            totalCount
            nodes {
              committedDate
              additions
              deletions
              message
            }
          }
        }
      }
    }
  }
}`

// UserPullRequestsQuery fetches user's pull request activity
const UserPullRequestsQuery = `
query($username: String!, $first: Int!, $after: String) {
  user(login: $username) {
    pullRequests(
      first: $first
      after: $after
      orderBy: {field: CREATED_AT, direction: DESC}
      states: [OPEN, CLOSED, MERGED]
    ) {
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        number
        title
        body
        state
        createdAt
        updatedAt
        closedAt
        mergedAt
        additions
        deletions
        changedFiles
        repository {
          nameWithOwner
          owner {
            login
          }
          primaryLanguage {
            name
          }
        }
        reviews(first: 10) {
          totalCount
          nodes {
            state
            submittedAt
          }
        }
        comments {
          totalCount
        }
        labels(first: 10) {
          nodes {
            name
          }
        }
      }
    }
  }
}`

// UserIssuesQuery fetches user's issue activity
const UserIssuesQuery = `
query($username: String!, $first: Int!, $after: String) {
  user(login: $username) {
    issues(
      first: $first
      after: $after
      orderBy: {field: CREATED_AT, direction: DESC}
      states: [OPEN, CLOSED]
    ) {
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        id
        number
        title
        body
        state
        createdAt
        updatedAt
        closedAt
        repository {
          nameWithOwner
          owner {
            login
          }
          primaryLanguage {
            name
          }
        }
        comments {
          totalCount
        }
        labels(first: 10) {
          nodes {
            name
          }
        }
      }
    }
  }
}`

// SearchUserRepositoriesQuery searches for repositories where user has contributed
const SearchUserRepositoriesQuery = `
query($searchQuery: String!, $first: Int!, $after: String) {
  search(
    query: $searchQuery
    type: REPOSITORY
    first: $first
    after: $after
  ) {
    pageInfo {
      hasNextPage
      endCursor
    }
    nodes {
      ... on Repository {
        nameWithOwner
        description
        url
        stargazerCount
        forkCount
        primaryLanguage {
          name
        }
        owner {
          login
          ... on Organization {
            name
          }
        }
        createdAt
        updatedAt
        repositoryTopics(first: 10) {
          nodes {
            topic {
              name
            }
          }
        }
      }
    }
  }
}`