#### CAVEATS

The KubeDirector release process is documented here for benefit of the maintainers of the main KubeDirector repo. A KubeDirector user, or even a contributing developer, should not need to read this doc.

The below process could be reduced to a smaller number of steps that reach the same end result. However, we take a few extra steps to honor a couple of principles:
* Don't add commits directly to the main repo. Commits come in through reviewed PR merges. The only situation where this restriction doesn't apply is a trivial doc edit made by a maintainer.
* Don't leave the main repo's master branch in a broken or misleading state at any point. We should not be relying on all of this release process happening quickly -- ideally it should, but in practice it could be paused or interrupted at some point.

In the below text, change "x.y.z" to whatever the version-to-be-released is, such as "0.2.0".

#### ABOUT BUGFIXES

It's possible that some other change may need to be made to KubeDirector, such as fixing a bug found in the regression testing. This may happen during the below process, after creating release-related branches but before the release is finalized.

If it's important that the fix gets into this release, then the fix must eventually make it into the x.y.z-release branch before the release is tagged. The safest general approach is for any fix to go into master branch (from a "fix branch" taken from master). Once the fix is merged to master, make sure your own repo has master synced and then rebase your own repo's x.y.z-release-info branch onto master.

An exception: if the fix is purely related to the new info added in the x.y.z-release-info branch, e.g. changing the release date, it can go directly into that branch instead.

If a fix happens even after the x.y.z-release branch has been created and modified, then that branch will need to be rebased as well (onto the updated x.y.z-release-info branch).

If you make any functional changes to master, remember to build and push the latest unstable KD image (modify Local.mk to enable push_default if necessary).

#### GENERAL PREP

If deps have not been updated recently, commit the results of "make dep" and "make modules" and merge that to master.

Build and push the latest unstable KD image (modify Local.mk to enable push_default if necessary).

Regression test this image.

#### PREP CHANGES IN DOCS

Make sure your own repo, both on GitHub and in your local clone, has its master branch synced with the main kubedirector repo.

In your local clone of your own repo, create the x.y.z-release-info branch from master.

Working on your local x.y.z-release-info branch:
* Change references to the previous KD version to x.y.z in doc/quickstart.md - for example changing from "v0.1.0" to "v0.2.0".
* Update/finalize HISTORY.md (i.e. release date and changes for version x.y.z).

Push your local x.y.z-release-info branch to your own GitHub repo.

Do NOT merge x.y.z-release-info to the main kubedirector repo yet!

#### SNAPSHOT DATA STRUCTURES ON WIKI

Create x.y.z-versioned pages of wiki docs for CRs (app, cluster, config) as snapshots of current content. Make sure to change each page's initial text appropriately, to describe how it is a spec for a particular released version.

#### CREATE RELEASE TAG POINT

In your local clone of your own repo, create the x.y.z-release branch from x.y.z-release-info.

Working on your local x.y.z-release branch:
* Search docs for links that include "kubedirector/wiki/Type-Definitions" (i.e. CR docs) and replace each with a link to the appropriate version-snapshot page.
* Change image version from unstable to x.y.z in Makefile and deployment-prebuilt.yaml.
* Build and push that KD image (modify Local.mk to enable push_default if necessary).
* Regression test this image.

Push your local x.y.z-release branch to your own GitHub repo.

Do NOT merge x.y.z-release to the main kubedirector repo yet!

#### PROMOTE CONTENT TO MAIN REPO

On GitHub for the main kubedirector repo, create the x.y.z-release-info branch from master.

Do a GitHub PR to merge your x.y.z-release-info branch to the main x.y.z-release-info branch. Don't proceed until that PR is approved and merged.

Do NOT merge x.y.z-release-info to master yet!

On GitHub for the main kubedirector repo, create the x.y.z-release branch from x.y.z-release-info.

Do a GitHub PR to merge your x.y.z-release branch to the main x.y.z-release branch. Don't proceed until that PR is approved and merged.

Do NOT merge x.y.z-release to master!

Don't proceed to subsequent steps until you are ready to make the release public. If you need to delay the release, don't forget to change the release date in HISTORY.md in both the x.y.z-release-info and x.y.z-release branches.

#### CREATE THE RELEASE

On GitHub, go to the releases page and click "Draft a new release". Name the tag as "vx.y.z" (for example "v0.2.0") and select the x.y.z-release branch as the tag's location.

The release title should be in the form "KubeDirector vx.y.z". The release description needs some boilerplate text (about checking for latest release etc.); also copy the version's information from HISTORY.md into the release description. Note that any relative links from HISTORY.md will have to be changed to plaintext or absolute links.

Create this tag/release but do NOT merge x.y.z-release to master!

Delete the x.y.z-release branch everywhere (local, your GitHub, main GitHub).

#### ADVERTISE THE RELEASE

Modify the "current" wiki page documenting each CR to add the x.y.z-versioned page to its bullet list of released versions.

Do a GitHub PR to merge the main repo's x.y.z-release-info branch to master.

Delete the x.y.z-release-info branch everywhere (local, your GitHub, main GitHub).
