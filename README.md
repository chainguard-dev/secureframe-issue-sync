# secureframe-issue-sync

Sync Secureframe tests to GitHub issues:

* Opens Github issues when a test fails
* Updates Github issues when test details change
* Closes Github issues when a test passes or is disabled

secureframe-issue-sync is designed to be used as a scheduled task, in particular GitHub Actions.

NOTE: This is using an undocumented Secureframe GraphQL API, so it may suddenly break. PR's welcome.

## Requirements

* A GitHub API token (preferably fine-grained for just issue management)
* A Secureframe API token (findable via browser headers)
* A Secureframe Company ID (findable via browser headers)

As Secureframe does not yet have a public API, you'll need to grab the latter two bits of information using your browser's Developer Tools functionality.

## Finding your Secureframe authentication data

1. Visit <https://app.secureframe.com/>
2. Enter your browser's "Developer Tools" feature
3. Click on the **Console** tab.
4. Type `sessionStorage.getItem("AUTH_TOKEN");` and press <enter>. This will show your auth token.
5. Type `sessionStorage.getItem("CURRENT_COMPANY_USER");` and press <enter>. This will show your company ID.

NOTE: Secureframe now invalidates authentication tokens every time you login. You may want to use a separate account for the authenticating this tool.

## Installation

```shell
go install github.com/chainguard-dev/secureframe-issue-sync@latest
```

## Usage: Command-Line

To build and install this tool, run:

```shell
secureframe-issue-sync --secureframe-token=<token> \
  --company=<company id> \
  --reports=soc2_alpha \
  --github-token=<token> \
  --github-repo=chainguard-dev/xyz`
```

There is a `--dry-run` flag available, which will pretend to make changes to GitHub instead of actually performing them.

You can also pass flags via environment variables, such as `SECUREFRAME_TOKEN=xyz`.

## Usage: GitHub Actions

In production, you're going to want to schedule the sync job to run every hour or so. Since you are already on Github, why not use GitHub Actions to do it?

See <https://github.com/chainguard-dev/secureframe-issue-sync/blob/main/github-action.yaml> for an example.
