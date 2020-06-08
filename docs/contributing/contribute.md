# Contributing

## Code of Conduct

Please make sure to read and observe our [Code of Conduct](https://github.com/kubeedge/community/blob/master/CODE_OF_CONDUCT.md).

## Community Expectations

KubeEdge is a community project driven by its community which strives to promote a healthy, friendly and productive environment.
The goal of the community is to develop a cloud native edge computing platform built on top of Kubernetes to manage edge nodes and devices at scale and demonstrate resiliency, reliability in offline scenarios. To build a platform at such scale requires the support of a community with similar aspirations.

- See [Community Membership](community.md) for a list of various community roles. With gradual contributions, one can move up in the chain.


## Prepartion

- Choose matched [golang](https://golang.org/dl) version and install:

|                         | Golang 1.10    | Golang 1.11     | Golang 1.12     | Golang 1.13     |
|-------------------------|----------------|-----------------|-----------------|-----------------|
| KubeEdge 1.1            | ✗              | ✗               | ✓               | ✗               |
| KubeEdge 1.2            | ✗              | ✗               | ✓               | ✓               |
| KubeEdge 1.3            | ✗              | ✗               | ✓               | ✓               |
| KubeEdge HEAD (master)  | ✗              | ✗               | ✓               | ✓               |

- Fork the repository on GitHub
- Download the repository

## Your First Contribution

We will help you to contribute in different areas like filing issues, developing features, fixing critical bugs and getting your work reviewed and merged.


### Find something to work on

We are always in need of help, be it fixing documentation, reporting bugs or writing some code.
Look at places where you feel best coding practices aren't followed, code refactoring is needed or tests are missing.
Here is how you get started.

#### Find a good first topic

There are [multiple repositories](https://github.com/kubeedge/) within the KubeEdge organization.
Each repository has beginner-friendly issues that provide a good first issue.
For example, [kubeedge/kubeedge](https://github.com/kubeedge/kubeedge) has [help wanted](https://github.com/kubeedge/kubeedge/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22) and [good first issue](https://github.com/kubeedge/kubeedge/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22) labels for issues that should not need deep knowledge of the system.
We can help new contributors who wish to work on such issues.

Another good way to contribute is to find a documentation improvement, such as a missing/broken link. Please see [Contributing](#Contributor Workflow) below for the workflow.

#### Work on an issue

When you are willing to take on an issue, you can assign it to yourself. Just reply with `/assign` or `/assign @yourself` on an issue,
then the robot will assign the issue to you and your name will present at `Assignees` list.

## File an Issue

While we encourage everyone to contribute code, it is also appreciated when someone reports an issue.

Issues should be filed under the appropriate KubeEdge sub-repository.

*Example:* a KubeEdge issue should be opened to [kubeedge/kubeedge](https://github.com/kubeedge/kubeedge/issues).

Please follow the prompted submission guidelines while opening an issue:

- Specific. Include as much details as possible: which version, what environment, what configuration, etc. If the bug is related to running the kubeedge server, please attach the kubeedge log (the starting log with kubeedge configuration is especially important).

- Reproducible. Include the steps to reproduce the problem. We understand some issues might be hard to reproduce, please includes the steps that might lead to the problem.

- Isolated. Please try to isolate and reproduce the bug with minimum dependencies. It would significantly slow down the speed to fix a bug if too many dependencies are involved in a bug report.

- Unique. Do not duplicate existing bug report.

- Scoped. One bug per report. Do not follow up with another bug inside one report.

We might ask for further information about the issue. Any duplicated report will be closed.

## Contributor Workflow

Please do not ever hesitate to ask a question or send a pull request.

This is a rough outline of what a contributor's workflow looks like:

- Create a topic branch from where to base the contribution. This is usually master.
- Make commits of logical units.
- Make sure commit messages are in the proper format (see below).
- Push changes in a topic branch to a personal fork of the repository.
- Submit a pull request to [kubeedge/kubeedge](https://github.com/kubeedge/kubeedge).
- The PR must receive an approval from two maintainers.

### Creating Pull Requests

Pull requests are often called simply "PR".
KubeEdge generally follows the standard [github pull request](https://help.github.com/articles/about-pull-requests/) process.

In addition to the above process, a bot will begin applying structured labels to your PR.

The bot may also make some helpful suggestions for commands to run in your PR to facilitate review.
These `/command` options can be entered in comments to trigger auto-labeling and notifications.
Refer to its [command reference documentation](https://go.k8s.io/bot-commands).

### Code Review

To make it easier for your PR to receive reviews, consider the reviewers will need you to:

* follow [good coding guidelines](https://github.com/golang/go/wiki/CodeReviewComments).
* write [good commit messages](https://chris.beams.io/posts/git-commit/).
* break large changes into a logical series of smaller patches which individually make easily understandable changes, and in aggregate solve a broader issue.
* label PRs with appropriate reviewers: to do this read the messages the bot sends you to guide you through the PR process.

### Format of the commit message

We follow a rough convention for commit messages that is designed to answer two questions: what changed and why.
The subject line should feature the what and the body of the commit should describe the why.

```
scripts: add test codes for metamanager

this add some unit test codes to improve code coverage for metamanager

Fixes #12
```

The format can be described more formally as follows:

```
<subsystem>: <what changed>
<BLANK LINE>
<why this change was made>
<BLANK LINE>
<footer>
```

The first line is the subject and should be no longer than 70 characters, the second line is always blank, and other lines should be wrapped at 80 characters. This allows the message to be easier to read on GitHub as well as in various git tools.

Note: if your pull request isn't getting enough attention, you can use the reach out on Slack to get help finding reviewers.

### Testing

There are multiple types of tests.
The location of the test code varies with type, as do the specifics of the environment needed to successfully run the test:

* Unit: These confirm that a particular function behaves as intended. Unit test source code can be found adjacent to the corresponding source code within a given package. These are easily run locally by any developer.
* Integration: These tests cover interactions of package components or interactions between KubeEdge components and Kubernetes control plane components like API server.  An example would be testing whether the device controller is able to create config maps when device CRDs are created in the API server.
* End-to-end ("e2e"): These are broad tests of overall system behavior and coherence. The e2e tests are in [kubeedge e2e](https://github.com/kubeedge/kubeedge/tree/master/tests/e2e).

Continuous integration will run these tests on PRs.
