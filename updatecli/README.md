# Updatecli Configuration

This directory contains [Updatecli](https://www.updatecli.io/) manifests used to automate updates in the alpha-omega-stats repository.

## Current Automations

### top-250-plugins.csv

The `top-250-plugins.yaml` manifest checks for changes in the `top-250-plugins.csv` file from the upstream source at:
[Upstream top-250-plugins.csv](https://raw.githubusercontent.com/gounthar/jdk8-removal/refs/heads/main/top-250-plugins.csv)

When changes are detected, Updatecli will:
1. Download the latest version of the file
2. Compare it with the local version
3. If different, create a pull request with the updated content

## Running Locally

To run Updatecli locally, you need to:

1. [Install Updatecli](https://www.updatecli.io/docs/intro/installation/)
2. Set the required environment variables:
   ```bash
   export UPDATECLI_GITHUB_TOKEN=your_github_token
   ```
3. Run the diff command to see what would change:
   ```bash
   updatecli diff --config ./updatecli/updatecli.d/top-250-plugins.yaml --values ./updatecli/values/default.yaml
   ```
4. Apply the changes:
   ```bash
   updatecli apply --config ./updatecli/updatecli.d/top-250-plugins.yaml --values ./updatecli/values/default.yaml
   ```

## Automation Schedule

The Updatecli workflow runs:
- Daily at midnight (UTC)
- On manual trigger via GitHub Actions workflow_dispatch