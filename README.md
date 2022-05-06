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
7. Run: `go run . --secureframe-token=<token> --reports=soc2_alpha --github-token=<token> --github-repo=chainguard-dev/xyz --github-label=soc2`

There is a `--dry-run` flag available.

You can also pass flags via environment variables, such as `SECUREFRAME_TOKEN=xyz`.
