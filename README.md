# secureframe-issue-sync

Sync Secureframe tests to GitHub issues:

* Opens Github issues when a test fails
* Updates Github issues when test details change
* Closes Github issues when a test passes or is disabled

secureframe-issue-sync is designed to be used as a scheduled task, and perfect for deploying via Github Actions. 

NOTE: This is using an undocumented Secureframe GraphQL API, so it may suddenly break. PR's welcome.

## Requirements

* A GitHub API token
* A Secureframe API token
* A Secureframe Company ID

As Secureframe does not yet have a public API, you'll need to grab the latter two bits of information using your browser's Developer Tools functionality. 

1. Visit <https://app.secureframe.com/dashboard/incomplete-tests/soc2-beta> or other dashboard page
2. Enter your browsers "Developer Tools"
3. Reload the page
4. Look for the `graphql` request that calls the `getDashboardTests` operation
5. Click the `Headers` tab 
6. Look for the `authorization` header, it will have a value in the form of `Bearer lr45laoeu21z4`: That is your Secureframe API token.
9. Click the `Payload` tab to capture the company ID and report keys to use.

## Usage: Github Actions

In production, your going to want to schedule the sync job to run every hour or so. Since you are already on Github, why not use Github Actions to do it?

See https://github.com/chainguard-dev/secureframe-issue-sync/blob/main/github-action.yaml for an example. 

## Usage: Command-line

`go run . --secureframe-token=<token> --reports=soc2_alpha --github-token=<token> --github-repo=chainguard-dev/xyz --github-label=soc2`

There is a `--dry-run` flag available.

You can also pass flags via environment variables, such as `SECUREFRAME_TOKEN=xyz`.
