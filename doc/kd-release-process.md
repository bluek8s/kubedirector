#### CAVEATS

The KubeDirector release process is documented here for benefit of the maintainers of the main KubeDirector repo. A KubeDirector user, or even a contributing developer, should not need to read this doc.

The below process could be reduced to a smaller number of steps that reach the same end result. However, we take a few extra steps to honor a couple of principles:
* Don't add commits directly to the main repo. Commits come in through reviewed PR merges. The only situation where this restriction doesn't apply is a trivial doc edit made by a maintainer.
* Don't leave a main repo branch in a broken or misleading state at any point. We should not be relying on all of this release process happening quickly -- ideally it should, but in practice it could be paused or interrupted at some point.

#### BRANCHES

In current KubeDirector development where simultaneous work on multiple releases is rare, the branching arrangement is simple. Leading-edge new release development happens on the master branch. If a patch for a previous release is required, that will be handled on a new branch created on demand for that patch release.

If a new branch needs to be created to do a patch release, then that branch should be created at the common ancestor commit between master and the relevant release tag that the patch release will be based on. E.g. if you needed to develop an 0.4.3 release based on 0.4.2, you could create the 0.4.3 branch like so:
```bash
    branchpoint=$(git merge-base master v0.4.2)
    git branch 0.4.3 $branchpoint
```

Note that by convention a tag for a release starts with "v", and any non-master branch for a release in progress is just the bare version string without a leading "v".

A "dev branch" will be referred to in the steps below. This will be the branch that is collecting the changes for the build of this release, whether master or some patch release branch like "0.4.3". To help make the release process mistake-free, copy the below text into a new document and make the following substitutions:
* Change "x.y.z" to whatever the version-to-be-released is, such as "0.5.0".
* Change "the dev branch" to swap out "dev" for the actual name of the dev branch. So e.g. you should replace "the dev branch" with "the master branch" or "the 0.4.3 branch".

Then follow the process using your modified copy of the text.

#### ABOUT BUGFIXES

It's possible that some other change may need to be made to KubeDirector, such as fixing a bug found in the regression testing. This may happen during the below process, after creating release-related branches but before the release is finalized.

If it's important that the fix gets into this release, then the fix must eventually make it into the x.y.z-release branch before the release is tagged. The safest general approach is for any fix to go into the dev branch. Once the fix is merged to the dev branch, make sure your own repo is synced with the dev branch, and then rebase your own repo's x.y.z-release-info branch onto the dev branch.

An exception: if the fix is purely related to the new info added in the x.y.z-release-info branch, e.g. changing the release date, it can go directly into that branch instead.

If a fix happens even after the x.y.z-release branch has been created and modified, then that branch will need to be rebased as well (onto the updated x.y.z-release-info branch).

If you make any functional changes to the dev branch, and ONLY IF the dev branch is the master branch, also remember to build and push the latest unstable KD image (modify Local.mk to enable push_default if necessary).

#### GENERAL PREP

If the dev branch is the master branch, build and push the latest unstable KD image (modify Local.mk to enable push_default if necessary).

Regression test this image.

#### PREP CHANGES IN DOCS

Make sure your own repo, both on GitHub and in your local clone, has its copy of the dev branch synced with the main kubedirector repo.

In your local clone of your own repo, create the x.y.z-release-info branch from the dev branch.

Working on your local x.y.z-release-info branch:
* Change references to the previous KD version to x.y.z in doc/quickstart.md.
* Update/finalize HISTORY.md (i.e. release date and changes for version x.y.z).
* Change the version string to "x.y.z-unstable" in version.go.

Push your local x.y.z-release-info branch to your own GitHub repo.

Do NOT merge x.y.z-release-info to the main kubedirector repo yet!

Finally, it is a good idea at this point to prepare any changes that will need to be made to the CRD definitions on the wiki (as described below in "ADVERTISE THE RELEASE"). If the release may be delayed, you could save these in local docs as opposed to updating the wiki. Note that you can choose to work with the wiki as a git repo (bluek8s/kubedirector.wiki.git) rather than using the web UI if you want.

#### CREATE RELEASE TAG POINT

In your local clone of your own repo, create the x.y.z-release branch from x.y.z-release-info.

Working on your local x.y.z-release branch:
* Change image version from unstable to x.y.z in Makefile and deployment-prebuilt.yaml.
* Change the version string to "x.y.z" in version.go.
* Build and push that KD image (modify Local.mk to enable push_default if necessary).
* Regression test this image.

Push your local x.y.z-release branch to your own GitHub repo.

Do NOT merge x.y.z-release to the main kubedirector repo yet!

#### PROMOTE CONTENT TO MAIN REPO

On GitHub for the main kubedirector repo, create the x.y.z-release-info branch from the dev branch.

Do a GitHub PR to merge your x.y.z-release-info branch to the main x.y.z-release-info branch. Don't proceed until that PR is approved and merged.

Do NOT merge x.y.z-release-info to the dev branch yet!

On GitHub for the main kubedirector repo, create the x.y.z-release branch from x.y.z-release-info.

Do a GitHub PR to merge your x.y.z-release branch to the main x.y.z-release branch. Don't proceed until that PR is approved and merged.

Do NOT merge x.y.z-release to the dev branch!

Don't proceed to subsequent steps until you are ready to make the release public. If you need to delay the release, don't forget to change the release date in HISTORY.md in both the x.y.z-release-info and x.y.z-release branches.

#### CREATE THE RELEASE

On GitHub, go to the releases page and click "Draft a new release". Name the tag as "vx.y.z" and select the x.y.z-release branch as the tag's location.

The release title should be in the form "KubeDirector vx.y.z". The release description needs some boilerplate text (about checking for latest release etc.); also copy the version's information from HISTORY.md into the release description. Note that any relative links from HISTORY.md will have to be changed to plaintext or absolute links.

Create this tag/release but do NOT merge x.y.z-release to the dev branch!

Delete the x.y.z-release branch everywhere (local, your GitHub, main GitHub).

#### ADVERTISE THE RELEASE

Modify the wiki page documenting each CRD so that it includes documentation of the support for any new properties added in this release. (Reference the existing tables to see how properties for a new version are added and marked.) Note that this is assuming the K8s API versioning practice where new properties are added to an existing API version while maintaining backwards compatibility; if we reach the point where a new API version is required then the CRD documentation will have to be further restructured.

Do a GitHub PR to merge the main repo's x.y.z-release-info branch to the dev branch.

Delete the x.y.z-release-info branch everywhere (local, your GitHub, main GitHub).
