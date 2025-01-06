This helps create Markdown reports for specific users in the Jenkins organizations.

Syntax: 

`./compute-stats.sh yaroslavafenkin,shlomomdahan 2024-12-01 2024-12-31`

The Markdown report now includes a list of repositories that supplied a release during the chosen timeframe at the end of the report.

The new output format for the list of repositories with releases is as follows:

### Released plugins
1. Released the [claim plugin](https://github.com/jenkinsci/jenkinsci/claim-plugin)
2. Released the [email-ext plugin](https://github.com/jenkinsci/jenkinsci/email-ext-plugin)
3. Released the [last-changes plugin](https://github.com/jenkinsci/jenkinsci/last-changes-plugin)
4. Released the [pipeline-aggregator-view plugin](https://github.com/jenkinsci/jenkinsci/pipeline-aggregator-view-plugin)
5. Released the [pollscm plugin](https://github.com/jenkinsci/jenkinsci/pollscm-plugin)
6. Released the [port-allocator plugin](https://github.com/jenkinsci/jenkinsci/port-allocator-plugin)

The repository names in the new format are sorted alphabetically to maintain consistency and readability.
