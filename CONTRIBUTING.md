<!-- omit in toc -->
# Contributing to k8gb

- [Getting started](#getting-started)
- [Getting help](#getting-help)
- [Reporting issues](#reporting-issues)
- [Contribution flow](#contribution-flow)
- [Local setup](#local-setup)
  - [Deploy k8gb locally](#deploy-k8gb-locally)
  - [Upgrade k8gb to local candidate](#upgrade-k8gb-to-local-candidate)
- [Overriding dev environment settings](#overriding-dev-environment-settings)
- [Testing](#testing)
  - [Testing against real k8s clusters](#testing-against-real-k8s-clusters)
- [Debugging](#debugging)
- [Metrics](#metrics)
- [Code style](#code-style)
- [Commit and Pull Request message](#commit-and-pull-request-message)
  - [Signature](#signature)
  - [Changelog](#changelog)
- [Documentation](#documentation)
- [k8gb.io website](#k8gbio-website)
  - [Local website authoring and testing](#local-website-authoring-and-testing)
- [End-to-end demo helper](#end-to-end-demo-helper)
- [Release process](#release-process)

k8gb is licensed under [Apache 2 License](./LICENSE) and accepts contributions via GitHub pull requests.
This document outlines the resources and guidelines necessary to follow by contributors to the k8gb project.

## Getting started

- Fork the repository on GitHub
- See the [local playground guide](/docs/local.md) for local dev environment setup

## Getting help

Feel free to ask for help and join the discussions at [k8gb community discussions forum](https://github.com/k8gb-io/k8gb/discussions).
We have [dedicated `#k8gb` channel on Cloud Native Computing Foundation (CNCF) Slack](https://cloud-native.slack.com/archives/C021P656HGB),
and we can also actively monitoring [`#sig-multicluster` channel on Kubernetes Slack](https://kubernetes.slack.com/archives/C09R1PJR3).

## Reporting issues

Reporting bugs is one of the best ways to contribute.
Feel free to open an issue describing your problem or question.

## Contribution flow

Following is a rough outline for the contributor's workflow:

- Create a topic branch from where to base the contribution.
- Make commits of logical units.
- Make sure commit messages are in the proper format (see below).
- Push changes in a topic branch to a personal fork of the repository.
- Submit a pull request to AbsaOSS/k8gb GitHub repository.
- Resolve review comments
- PR must receive an "LGTM" approval from at least one maintainer listed in the `CODEOWNERS` file.

## Local setup

### Deploy k8gb locally

```sh
make deploy-full-local-setup
```
deploys k8gb from scratch, including:

* 2 local clusters
* Stable k8gb helm chart
* Gslb Custom Resources examples
* Test applications

### Upgrade k8gb to local candidate
```sh
make upgrade-candidate
```
performs upgrade of k8gb helm chart and controller to the testing version built from your current development tree.

## Overriding dev environment settings

Sometimes there is a need to override environment variables used by `make` targets for local k8gb development.
This can be easily achieved by providing the list of environment variables with respective values in the `.env` file at the local repository root:
```sh
cat .env

# .env:
LOG_LEVEL=info
LOG_FORMAT=json
```
Overrides done this way can persist between terminal sessions and can be used as a single point of configuration for development in terminal and IDE of choice.

## Testing

- Any functional GSLB controller code change should be secured by the corresponding [unit tests](https://github.com/AbsaOSS/k8gb/tree/master/controllers/gslb_controller_test.go).
- Integration terratest suite is located [here](https://github.com/AbsaOSS/k8gb/tree/master/terratest).
  These tests are updated only if the change is substantial enough to affect the main end-to-end flow.
- See the [local playground guide](https://github.com/k8gb-io/k8gb/blob/master/docs/local.md) for local testing environment setup and integration test execution.

### Unit testing
- Include unit tests when you contribute new features, as they help to a) prove that your code works correctly, and b) guard against future breaking changes to lower the maintenance cost.
- Bug fixes also generally require unit tests, because the presence of bugs usually indicates insufficient test coverage.

Use `make test` to check your implementation changes.

### Testing against real k8s clusters

There is a possibility to execute the integration terratest suite over the real clusters.
For this, you need to override the set of test settings as in the example below.
```sh
PRIMARY_GEO_TAG=af-south-1 \
SECONDARY_GEO_TAG=eu-west-1 \
DNS_SERVER1=a377095726f1845fb85b95c2afef8ac0-9a1a10f24e634e28.elb.af-south-1.amazonaws.com \
DNS_SERVER1_PORT=53 \
DNS_SERVER2=a873f5c83be624a0a84c05a743d598a8-443f7e0285e4a28f.elb.eu-west-1.amazonaws.com \
DNS_SERVER2_PORT=53 \
GSLB_DOMAIN=test.k8gb.io \
K8GB_CLUSTER1=arn:aws:eks:af-south-1:<aws-account-id>:cluster/k8gb-cluster-af-south-1 \
K8GB_CLUSTER2=arn:aws:eks:eu-west-1:<aws-account-id>:cluster/k8gb-cluster-eu-west-1 \
make terratest
```

## Debugging

1. Install Delve debugger first. Follow the [installation instructions](https://github.com/go-delve/delve/tree/master/Documentation/installation) for specific platforms from Delve's website.

2. Run delve with options specific to IDE of choice.
There is a dedicated make target available for Goland:

    ```sh
    make debug-idea
    ```
    [This article](https://dev4devs.com/2019/05/04/operator-framework-how-to-debug-golang-operator-projects/) describes possible option examples for Goland and VS Code.

3. Attach debugger of your IDE to port `2345`.

## Metrics
More info about k8gb metrics can be found in the [metrics.md](/docs/metrics.md) document. 
If you need to check and query the k8gb metrics locally, you can install a Prometheus in the local clusters using the `make deploy-prometheus` command.

The deployed Prometheus scrapes metrics from the dedicated k8gb operator endpoint and makes them accessible via Prometheus web UI:

- http://127.0.0.1:8080
- http://127.0.0.1:8081

To uninstall Prometheus, run `make uninstall-prometheus`

## Code style

k8gb project is using the coding style suggested by the Golang community. See the [golang-style-doc](https://github.com/golang/go/wiki/CodeReviewComments) for details.

Please follow this style to make k8gb easy to review, maintain and develop.
Run `make check` to automatically check if your code is compliant.

## Commit and Pull Request message

We follow a rough convention for PR and commit messages, which is designed to answer two questions: what changed and why.
The subject line should feature the what, and the body of the message should describe the why.
The format can be described more formally as follows:

```
<what was changed>

<why this change was made>

<footer>
```

The first line is the subject and should be no longer than 70 characters.
The second line is always blank.
Consequent lines should be wrapped at 80 characters.
This way, the message is easier to read on GitHub as well as in various git tools.

```
scripts: add the test-cluster command

This command uses "k3d" to set up a test cluster for debugging.

Fixes #38
```

Commit message can be made lightweight unless it is the only commit forming the PR.
In that case, the message can follow the simplified convention:

```
<what was changed and why>
```
This convention is useful when several minimalistic commit messages are going to form PR descriptions as bullet points of what was done during the final squash and merge for PR.

### Signature

As a CNCF project, k8gb must comply with [Developer Certificate of Origin (DCO)](https://developercertificate.org/) requirement.
[DCO GitHub Check](https://github.com/apps/dco) automatically enforces DCO for all commits.
Contributors are required to ensure that every commit message contains the following signature:
```txt
Signed-off-by: NAME SURNAME <email@address.example.org>
```
The best way to achieve this automatically for local development is to create the following alias in the `~/.gitconfig` file:
```.gitconfig
[alias]
ci = commit -s
```
When a commit is created in GitHub UI as a result of [accepted suggested change](https://docs.github.com/en/github/collaborating-with-issues-and-pull-requests/incorporating-feedback-in-your-pull-request#applying-suggested-changes), the signature should be manually added to the "optional extended description" field.

### Changelog

The [CHANGELOG](CHANGELOG.md) is automatically generated from Github PRs and Issues during release.
Use dedicated [keywords](https://docs.github.com/en/github/managing-your-work-on-github/linking-a-pull-request-to-an-issue#linking-a-pull-request-to-an-issue-using-a-keyword) in PR message or [manual PR and Issue linking](https://docs.github.com/en/github/managing-your-work-on-github/linking-a-pull-request-to-an-issue#manually-linking-a-pull-request-to-an-issue) for clean changelog generation.
Issues and PRs should be also properly tagged with valid project tags ("bug", "enhancement", "wontfix", etc )

## Documentation

If contribution changes the existing APIs or user interface, it must include sufficient documentation explaining the use of the new or updated feature.

## k8gb.io website

k8gb.io website is a Jekyll-based static website generated from project markdown documentation and hosted by GitHub Pages.
`gh-pages` branch contains the website source, including configuration, website layout, and styling.
Markdown documents are automatically populated to `gh-pages` from the main branch and should be authored there.
Changes to the k8gb.io website layout and styling should be checked out from the `gh-pages` branch and  PRs should be created against `gh-pages`.

### Local website authoring and testing

These instructions will help you to set up and use local website authoring and testing environment:
- Check-out from the `gh-pages` branch
- Create dedicated [GitHub Personal Access Token](https://github.com/settings/tokens/new) with `public_repo` permission and assign it to the `JEKYLL_GITHUB_TOKEN` environment variable:
  ```sh
- Run the following `make` target to build and serve the local copy of the k8gb.io website.
  ```sh
  make serve
  ```
  *The target utilizes the `jekyll/jekyll` docker container to avoid unnecessary installation of local GitHub page authoring dependencies.*

- Open the `http://localhost:4000/` page in your browser.
- Website will automatically rebuild and refresh in the browser to accommodate the related code changes.

## End-to-end demo helper

The demo helper is designed to work with `podinfo` that was deployed by

```sh
make deploy-test-apps
```

It will configure `podinfo` to expose geotag as part of an HTTP response.

To test and/or demonstrate continuous query to GSLB enabled endpoint execute

```sh
make demo DEMO_URL=https://failover.test.exampledns.tk
```

The happy path will look like:

```sh
[Thu May 27 15:35:26 UTC 2021] ...

200  "message": "eu-west-1",

[Thu May 27 15:35:31 UTC 2021] ...

200
  "message": "eu-west-1",
[Thu May 27 15:35:36 UTC 2021] ...
```

The sources for demo helper images can be found [here](deploy/test-apps/curldemo/)

To enable verbose debug output declare `DEMO_DEBUG=1` like
```sh
make demo DEMO_URL=https://failover.test.exampledns.tk DEMO_DEBUG=1
```

## Release process

* Bump the version in `Chart.yaml`, see [example PR](https://github.com/k8gb-io/k8gb/pull/521).
* Merge the Pull Request after the review approval
* Create release tag (replace `v0.8.1` with an actual version)
```sh
git tag v0.8.1
```
* Push release tag
```sh
git push origin refs/tags/v0.8.1
```
* Watch the [release pipeline](https://github.com/k8gb-io/k8gb/actions/workflows/release.yaml) status
* Check the [offline changelog](https://github.com/k8gb-io/k8gb/actions/workflows/changelog_pr.yaml) creation

---
Thanks for contributing!
