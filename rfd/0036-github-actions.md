
---
author: jane quintero (jane@goteleport.com)   
state: implemented
---


# Bot 

## What 

This RFD proposes the implementation of using Github Actions to better manage the Teleport repository's pull requests. The first iteration will include:  
- Auto assigning reviewers to pull requests. 
- Checking approvals for pull requests. 
- Dismissing stale workflow runs. 

## Why 

To improve the speed and quality of the current development workflow process.

## Implementation 

### Getting Data 

Pull request metadata will be obtained via the [execution context](https://docs.github.com/en/enterprise-server@3.0/actions/reference/context-and-expression-syntax-for-github-actions). This execution context contains `GITHUB_EVENT_PATH` as an environment variable which is the path to a JSON file with the complete event payload. With this, the pull request metadata will be unmarshaled into a `PullRequestMetadata` struct and will be used to make the necessary API calls. 

```go
  // Example PullRequestMetadata struct
type PullRequestMetadata struct {
	Author     string
	RepoName   string
	RepoOwner  string
	Number     int
	HeadSHA    string
	BaseSHA    string
	Reviewer   string
	BranchName string
}
```


### Workflows
### Assigning Reviewers 

Reviewers will be assigned when a pull request is opened, ready for review, or reopened. 

```yaml
# Example workflow configuration 
name: Assign
on: 
  pull_request_target:
    types: [assigned, opened, reopened, ready_for_review]
jobs:
  auto-request-review:
    name: Auto Request Review
    runs-on: ubuntu-latest
    steps:
      # Check out the master branch of the Teleport repository. This is to prevent an
      # attacker from submitting their review assignment logic.
      - name: Checkout master branch
        uses: actions/checkout@master        
      - name: Installing the latest version of Go.
        uses: actions/setup-go@v2
      # Run "assign-reviewers" subcommand on bot.
      - name: Assigning reviewers 
        run: go run cmd/bot.go --token=${{ secrets.GITHUB_TOKEN }} --default-reviewers=${{ secrets.defaultreviewers }} --reviewers=${{ secrets.reviewers }} assign-reviewers

```

### Checking Reviews 

Every time pull request and pull request review events occur, the bot will check if all the required reviewers have approved. 

```yaml
# Example Check workflow
name: Check
on: 
  pull_request_review:
    type: [submitted, edited, dismissed]
  pull_request_target: 
    types: [assigned, opened, reopened, ready_for_review, synchronize]

jobs: 
  check-reviews:
    name: Checking reviewers 
    runs-on: ubuntu-latest
    steps:
      # Check out the master branch of the Teleport repository. This is to prevent an
      # attacker from submitting their review assignment logic. 
      - name: Check out the master branch 
        uses: actions/checkout@master
      - name: Installing the latest version of Go.
        uses: actions/setup-go@v2
        # Run "check-reviewers" subcommand on bot.
      - name: Checking reviewers
        run: go run cmd/bot.go --token=${{ secrets.GITHUB_TOKEN }} --default-reviewers=${{ secrets.defaultreviewers }} --reviewers=${{ secrets.reviewers }} check-reviewers
```

#### Secrets 

To know which reviewers to assign and check for, a hardcoded JSON object will be used as a Github secret. Usernames will be the name of the key and the value will be a list of required reviewers' usernames. 


```json
    // Example `reviewers` secret
    {
        "author1": ["reviewer0", "reviewer1"],
        "author2": ["reviewer2", "reviewer3", "reviewer4"]
    }
```

If an author is external or is not in `reviewers`, default reviewers will be assigned. Default reviewers will also be stored as a secret that is a string representing a list. 

```json
  // Example `defaultreviewers` secret 

  ["defaultreviewer1", "defaultreviewer2", "defaultreviewer3"]
```

### Dismissing Stale Runs 

This workflow dismisses stale workflow runs every 30 minutes for every open pull request in the Teleport repository. There is a separate workflow for this because when a review event occurs on an external contributor's PR, the token in that context does not have the correct permissions. 

*There doesn't seem to be a way to work around this without using CRON job and not creating an additional access token.* 

```yaml
  // Example dismissing stale runs workflow 
  name: Dismiss Stale Workflows Runs
on:
  schedule:
    - cron:  '0,30 * * * *' 

jobs: 
  dismiss-stale-runs:
    name: Dismiss Stale Workflow Runs
    runs-on: ubuntu-latest
    steps:
      - name: Check out the master branch 
        uses: actions/checkout@master
      - name: Installing the latest version of Go.
        uses: actions/setup-go@v2
        # Run "dismiss-runs" subcommand on bot.
      - name: Dismiss
        run: cd .github/workflows/teleport-ci && go run cmd/bot.go --token=${{ secrets.GITHUB_TOKEN }} dismiss-runs

```
### Authentication & Permissions

For authentication, Github Actions provides a token to use in workflow, saved as `GITHUB_TOKEN` in the `secrets` context, to authenticate on behalf of Github actions. The token expires when the job is finished. 

### Bot Edits and Failures 

The [CODEOWNERS](https://docs.github.com/en/github/creating-cloning-and-archiving-repositories/creating-a-repository-on-github/about-code-owners) feature will be used to assign reviewers who can approve edits to the `.github/workflows` directory if there is a change to `.github/workflows`.

__CODEOWNERS need to approve these changes before the edits get merged.__ 

In the event that the bot has a bug or fails, repository admins can override the failed checks and force merge any changes. 
### Security 

To prevent edits to the contents of the workflow directory after CODEOWNERS have approved for external contributors, we need to invalidate approvals for the following commits. This can be done via hitting the [dismiss a review](https://docs.github.com/en/rest/reference/pulls#dismiss-a-review-for-a-pull-request) endpoint for all reviews in an approved state. This will occur on the `synchronize` (commit push to a pull request) event type. 

### Dependencies
This bot will use the [go-github](https://github.com/google/go-github) client library to access the Github API to assign, check reviewers, and dismiss stale workflow runs. 


### Scenarios Listed

Internal contributors:

- Reviewers will be assigned when a pull request event triggers.
- PR can't be merged until it has all required approvals.
- Each review event will trigger the check workflow and the PR will be checked for approvals.
- Each time the `Check` workflow triggers, stale workflow runs will be dismissed.
- New commits will not invalidate approvals.

External contributors:

- Reviewers will be assigned when a pull request event triggers.
- PR can't be merged until it has all required approvals.
- Each review event will trigger the check workflow and the PR will be checked for approvals.
- A cron job will invalidate stale `Check` workflow runs every 30 minutes.
- New commits *will* invalidate approvals unless they are committed and verified by Github.
  - Reviewers will be tagged in a comment on the pull request to rereview upon invalidation.

