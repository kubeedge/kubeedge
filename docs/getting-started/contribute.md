# How to contribute

Kubeedge is Apache 2.0 licensed and accepts contributions via GitHub pull requests. This document outlines some of the conventions on commit message formatting, contact points for developers, and other resources to help get contributions into kubeedge.

## Email and chat

- Email: [kubeedge](https://groups.google.com/forum/?hl=en#!forum/kubeedge)
- Slack: [kubeedge](https://join.slack.com/t/kubeedge/shared_invite/enQtNDg1MjAwMDI0MTgyLTQ1NzliNzYwNWU5MWYxOTdmNDZjZjI2YWE2NDRlYjdiZGYxZGUwYzkzZWI2NGZjZWRkZDVlZDQwZWI0MzM1Yzc)

## Getting started

- Fork the repository on GitHub
- Read the [setup](../setup/setup.md#build) for build instructions

## Reporting bugs and creating issues

Reporting bugs is one of the best ways to contribute. However, a good bug report has some very specific qualities, so please read over our short document on [reporting bugs](reporting_bugs.html) before submitting a bug report. This document might contain links to known issues, another good reason to take a look there before reporting a bug.

### Contribution flow

This is a rough outline of what a contributor's workflow looks like:

- Create a topic branch from where to base the contribution. This is usually master.
- Make commits of logical units.
- Make sure commit messages are in the proper format (see below).
- Push changes in a topic branch to a personal fork of the repository.
- Submit a pull request to kubeedge/kubeedge.
- The PR must receive an approval from two maintainers.

Thanks for contributing!

### Code style

The coding style suggested by the Golang community is used in kubeedge. See the [style doc](https://github.com/golang/go/wiki/CodeReviewComments) for details.

Please follow this style to make kubeedge easy to review, maintain and develop.

### Format of the commit message

We follow a rough convention for commit messages that is designed to answer two
questions: what changed and why. The subject line should feature the what and
the body of the commit should describe the why.

```
scripts: add test codes for metamanager

this add some unit test codes to imporve code coverage for metamanager

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
