# Contributing to KubeDirector

First - thank you for taking the time to look up how to contribute to KubeDirector! We know your time is valuable, and we want to make sure that you're able to work easily and effectively with the code that we're providing.

## Tools that we use

Most of the developers working on KubeDirector use MacOS, so there may be some unforseen issues in Linux and Windows development. Please make a Pull Request against this file if something here is out of date or inaccurate. Currently, we use the following tools and versions:

* go version 1.11 or 1.11.1
* dep version 0.5.0

Many of us have also set up travis-ci on our personal forks. To set up travis-ci on your personal fork, follow these instructions:

1. Log in to GitHub
1. Sign up for Travis CI ([link](https://travis-ci.com/))
1. In Github, go to the GitHub Marketplace
1. Click "Continuous Integration" in the left "Categories" tree.
1. Click "Travis CI"
1. Scroll to the bottom, Highlight "Open Source", and click "Install it for free"
1. Click "Complete Order"
1. Under "Install on your personal account", select "Only select repositories"
1. From the dropdown below the radio button, select kubedirector
1. Click Install

Before making a Pull Request, please ensure that the code still builds. If you're on Windows, this might not work for you, but if you're on MacOS or Linux, this process should be pretty seamless.

Attempt a build with the following steps:

1. `make dep`
1. `make compile`
1. `make build`

If everything works in that sequence, please push to your own fork and make the Pull Request. We will do our best to get to your Pull Request in a timely fashion. It may take a while to get to any given Pull Request - please be patient. Please note that having a travis-ci build set up on a personal fork will allow you to run checks before opening your Pull Request.