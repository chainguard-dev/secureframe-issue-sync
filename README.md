# secureframe-issue-sync

Sync Secureframe tests to GitHub issues

* Opens Github issues when a test fails
* Updates Github issues when test details change
* Closes Github issues when a test passes or is disabled

This is using an undocumented Secureframe GraphQL API, so it may suddenly break.

PR's welcome.

## Usage

1. Visit <https://app.secureframe.com/dashboard/incomplete-tests/soc2-beta> or other dashboard page
2. Enter your browsers "Developer Tools"
3. Reload the page
4. Look for the `graphql` request that calls the `getDashboardTests` operation
5. Click the `Headers` tab to capture the bearer token
6. Click the `Payload` tab to capture the company ID and report keys to use.
7. Run: `go run . -bearer-token=<bearer token> --report-keys=soc2_alpha --company-id=079b854c --github-repo=somewhere/project --github-label=soc2`

NOTE: There is a `--dry-run` flag available.
