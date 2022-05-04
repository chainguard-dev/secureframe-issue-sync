# secureframe-issue-sync

Sync Secureframe tests to GitHub issues

This is experimental, but as a proof-of-concept it halfway works. PR's welcome.

# Usage

1. Visit <https://app.secureframe.com/dashboard/incomplete-tests/soc2-beta> or other dashboard page
2. Enter your browsers "Developer Tools"
3. Reload the page
4. Look for the `graphql` request that calls the `getDashboardTests` operation
5. Click the `Headers` tab to capture the bearer token
6. Click the `Payload` tab to capture the company ID and report keys to use.
7. Run:

`go run . --dry-run --bearer-token=<bearer token> --report-keys=soc2_alpha --company-id=079b854c`

The dry-run mode will indicate which issues it finds.
