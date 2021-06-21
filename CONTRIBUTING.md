# Contributing to KubeDirector

First - thank you for taking the time to look up how to contribute to KubeDirector! We know your time is valuable, and we want to make sure that you're able to work easily and effectively with the code that we're providing.

## Development environment

FYI most of the developers working on KubeDirector use macOS, so there may be unforeseen issues in Linux and Windows development. Please make a Pull Request against this file if something here is out of date or inaccurate.

Currently, we use the following tools and versions:

* go version 1.13 or later
* version 0.15.2 of the [Operator SDK](https://github.com/operator-framework/operator-sdk)
* Docker (any recent version should do)

The [KubeDirector development doc](https://github.com/bluek8s/kubedirector/blob/master/doc/kubedirector-development.md) goes into greater detail about setting up your environment, building KubeDirector, and deploying. It is essential reading.

Many people have also set up Travis CI on their personal forks, to run basic sanity checks when code changes are pushed to the fork. If you would like to do this, follow these instructions:

1. Log in to GitHub.
1. Sign up for Travis CI ([link](https://travis-ci.com/)).
1. In Github, go to the GitHub Marketplace.
1. Search for and then click on "Travis CI".
1. Scroll to the bottom, highlight "Open Source", and click "Install it for free".
1. Click "Complete Order".
1. Under "Install on your personal account", select "Only select repositories".
1. From the dropdown below the radio button, select your kubedirector fork.
1. Click "Install".

## The roadmap

Current work items are tracked in the [GitHub issues list](https://github.com/bluek8s/kubedirector/issues), but for a more organized look at the KubeDirector development priorities see the [roadmap](https://github.com/bluek8s/kubedirector/blob/master/ROADMAP.md).

## Submitting changes

Before making a Pull Request (PR), it is almost always best to ensure that you are addressing an existing issue that has been assigned to you. This consideration doesn't necessarily apply to very minor PRs such as typo fixes, but usually you should be able to point to the issue that you are solving with your PR. If an issue doesn't exist yet, please file one! If an issue exists but is unassigned, feel free to ask in the issue comments if you can take it.

Once you have made and tested your change, do a final build before creating a PR. Make sure all of your code is pushed to your fork of KubeDirector. (If you have Travis CI integration for your fork as described above, you can have some confidence that the PR build checks will pass.) Then go ahead and create your PR.

We don't currently have a PR template that must be adhered to. If you identify the issue number that you are solving, describe your changes, and describe the testing you have done, then your PR is off to a great start. We will do our best to get to your PR in a timely fashion.
