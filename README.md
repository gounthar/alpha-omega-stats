This helps create Markdown reports for specific users in the Jenkins organizations.

Syntax: 

`./compute-stats.sh handle1,handle2 2024-12-01 2024-12-31`

The Markdown report now includes a list of repositories that supplied a release during the chosen timeframe at the end of the report.

The new output format for the list of repositories with releases is as follows:

### Released plugins
1. Released the [claim plugin](https://github.com/jenkinsci/claim-plugin)
2. Released the [email-ext plugin](https://github.com/jenkinsci/email-ext-plugin)
3. Released the [last-changes plugin](https://github.com/jenkinsci/last-changes-plugin)
4. Released the [pipeline-aggregator-view plugin](https://github.com/jenkinsci/pipeline-aggregator-view-plugin)
5. Released the [pollscm plugin](https://github.com/jenkinsci/pollscm-plugin)
6. Released the [port-allocator plugin](https://github.com/jenkinsci/port-allocator-plugin)

The repository names in the new format are sorted alphabetically to maintain consistency and readability.


```
./count_prs.sh repos.txt 2024
./compute-stats.sh gounthar,jonesbusy 2024-12-01 2025-01-15
./group-prs.sh prs_gounthar_and_others_2024-12-01_to_2025-01-15.json plugins.json
```
