# Release Process

- [Release Process](#release-process)
    - [Breaking Changes](#breaking-changes)
  - [Major Release Procedure](#major-release-procedure)
    - [Release Notes](#release-notes)
    - [Tagging Procedure](#tagging-procedure)
      - [Test building artifacts](#test-building-artifacts)
      - [Installing goreleaser](#installing-goreleaser)
  - [Non-major Release Procedure](#non-major-release-procedure)

This document outlines the release process for AtomOne.

AtomOne follows [semantic versioning](https://semver.org), but with the following deviations to account for state-machine and API breaking changes: 

- State-machine breaking changes will result in an increase of the major version X (X.y.z).
- Emergency releases & API breaking changes will result in an increase of the minor version Y (x.Y.z | x > 0).
- All other changes will result in an increase of the patch version Z (x.y.Z | x > 0).

**Note:** In case a major release is deprecated before ending up on the network (due to potential bugs), 
it is replaced by a minor release (eg: `v14.0.0` → `v14.1.0`). 
As a result, this minor release is considered state-machine breaking.

### Breaking Changes

A change is considered to be ***state-machine breaking*** if it requires a coordinated upgrade for the network to preserve [state compatibility](./STATE-COMPATIBILITY.md). 
Note that when bumping the dependencies of [Cosmos SDK](https://github.com/cosmos/cosmos-sdk), [IBC](https://github.com/cosmos/ibc-go), and [ICS](https://github.com/cosmos/interchain-security) we will only treat patch releases as non state-machine breaking.

A change is considered to be ***API breaking*** if it modifies the provided API. This includes events, queries, CLI interfaces. 

## Major Release Procedure

A _major release_ is an increment of the first number (eg: `v9.1.0` → `v10.0.0`).

**Note**: Generally, PRs should target either `main` or a long-lived feature branch (see [CONTRIBUTING.md](./CONTRIBUTING.md#pull-requests)).
An exception are PRs open via the Github mergify integration (i.e., backported PRs). 

* Once the team feels that `main` is _**feature complete**_, we create a `release/vY` branch (going forward known as release branch), 
  where `Y` is the version number, with the minor and patch part substituted to `x` (eg: 11.x). 
  * Update the [GitHub mergify integration](./.mergify.yml) by adding instructions for automatically backporting commits from `main` to the `release/vY` using the `A:backport/vY` label.
  * **PRs targeting directly a release branch can be merged _only_ when exceptional circumstances arise**.
* In the release branch 
  * Create a new version section in the `CHANGELOG.md` (follow the procedure described [below](#changelog))
  * Additionally verify that the `UPGRADING.md` file is up to date and contains all the necessary information for upgrading to the new version.
* We freeze the release branch from receiving any new features and focus on releasing a release candidate.
  * Finish audits and reviews.
  * Add more tests.
  * Fix bugs as they are discovered.
* After the team feels that the release branch works fine (i.e., has `~90%` chance of reaching mainnet), we cut a release candidate.
  * Create a new annotated git tag for a release candidate in the release branch (follow the [Tagging Procedure](#tagging-procedure)).
  * The release verification on public testnets must pass. 
  * When bugs are found, create a PR for `main`, and backport fixes to the release branch.
  * Create new release candidate tags after bugs are fixed.
* After the team feels the release candidate is mainnet ready, create a full release:
  * **Note:** The final release MUST have the same commit hash as the latest corresponding release candidate.
  * Create a new annotated git tag in the release branch (follow the [Tagging Procedure](#tagging-procedure)). This will trigger the automated release process (which will also create the release artifacts).
  * Once the release process completes, modify release notes if needed.

### Release Notes

Release notes should be appended to the `CHANGELOG.md` file from the release
branch. 

With every release, the `goreleaser` tool will create a file with all the build artifact checksums and upload it alongside the artifacts.
The file is called `SHA256SUMS-{{.version}}.txt` and contains the following:
```
098b00ed78ca01456c388d7f1f22d09a93927d7a234429681071b45d94730a05  atomoned_0.0.4_windows_arm64.exe
15b2b9146d99426a64c19d219234cd0fa725589c7dc84e9d4dc4d531ccc58bec  atomoned_0.0.4_darwin_amd64
604912ee7800055b0a1ac36ed31021d2161d7404cea8db8776287eb512cd67a9  atomoned_0.0.4_darwin_arm64
76e5ff7751d66807ee85bc5301484d0f0bcc5c90582d4ba1692acefc189392be  atomoned_0.0.4_linux_arm64
bcbca82da2cb2387ad6d24c1f6401b229a9b4752156573327250d37e5cc9bb1c  atomoned_0.0.4_windows_amd64.exe
f39552cbfcfb2b06f1bd66fd324af54ac9ee06625cfa652b71eba1869efe8670  atomoned_0.0.4_linux_amd64
```

For security reason the content of this file is also duplicated in the
`RELEASES.md` file.

### Tagging Procedure

**Important**: _**Always create tags from your local machine**_ since all release tags should be signed and annotated.
Using Github UI will create a `lightweight` tag, so it's possible that `atomoned version` returns a commit hash, instead of a tag.
This is important because most operators build from source, and having incorrect information when you run `make install && atomoned version` raises confusion.

The following steps are the default for tagging a specific branch commit using git on your local machine. Usually, release branches are labeled `release/v*`:

Ensure you have checked out the commit you wish to tag and then do:
```bash
git pull --tags

# test tag creation and releasing using goreleaser
make create-release-dry-run TAG=v11.0.0

# after successful test push the tag
make create-release TAG=v11.0.0
```

To re-create a tag:
```bash
# delete a tag locally
git tag -d v11.0.0  

# push the deletion to the remote
git push --delete origin v11.0.0 

# redo create-release
make create-release-dry-run TAG=v11.0.0
make create-release TAG=v11.0.0
```

#### Test building artifacts

Before tagging a new version, please test the building releasing artifacts by running:

```bash
# TODO run with appropriate go version
TM_VERSION=$(make print_tm_version) goreleaser release --snapshot --clean --debug
```

#### Installing goreleaser

<!-- TODO: fix version of goreleaser to avoid using different version that
might break build reproducibility -->
Check the instructions for installing goreleaser locally for your platform
* https://goreleaser.com/install/


## Non-major Release Procedure

A minor release_ is an increment of the _point number_ (eg: `v9.0.0 → v9.1.0`, also called _point release_). 
A _patch release_ is an increment of the patch number (eg: `v10.0.0` → `v10.0.1`).

**Important**: _**Non-major releases must not break consensus.**_

Updates to the release branch should come from `main` by backporting PRs 
(usually done by automatic cherry pick followed by a PRs to the release branch). 
The backports must be marked using `backport/Y` label in PR for main.
It is the PR author's responsibility to fix merge conflicts, update changelog entries, and
ensure CI passes. If a PR originates from an external contributor, a member of the codeowners assumes
responsibility to perform this process instead of the original author.

After the release branch has all commits required for the next patch release:

* Update the `CHANGELOG.md` and the [release notes](#release-notes).
* Create a new annotated git tag in the release branch (follow the [Tagging Procedure](#tagging-procedure)). This will trigger the automated release process (which will also create the release artifacts).
* Once the release process completes, modify release notes if needed.
