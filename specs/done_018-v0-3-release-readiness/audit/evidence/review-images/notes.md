# Review: images/

- **Date**: 2026-10-04
- **Reviewer tier**: sonnet (verification pending)
- **Checked against**: `images/README.md`, `.github/workflows/images.yaml`, `.github/dependabot.yml`, `specs/done_008-hardened-github-actions/spec.md`, `docs/module-authoring.md` (image pinning)

## Scope reviewed

Read in full:
- `images/README.md`
- `images/common/steamcmd/Dockerfile` (131 lines) and `steam-install.sh` (109 lines)
- `images/games/nuclear-option/Dockerfile` and `entrypoint.sh`
- `.github/workflows/images.yaml` (all 391 lines)

Read in part, to cross-check a claim:
- `.github/dependabot.yml` (`docker` entries for the two image directories)
- `Makefile` (any `docker build` of `images/`)

Not reviewed: the `modules/` submodule is empty in this worktree, so the README's claim that official templates pin the nuclear-option image by digest was not checked. No image was built or run and no registry was queried. The runtime claims below come from reading the scripts.

## Method

1. Read both Dockerfiles and both scripts top to bottom. Traced `UPDATE_ON_BOOT=true` and `false` through `entrypoint.sh` into `steam-install.sh`.
2. Compared every instruction in `README.md` (templates, build commands, trigger list, retry description) with the files they describe.
3. Traced `images.yaml` for a PR run, a master push and a `workflow_dispatch`: base build, digest hand-off to the game job, sign, verify, tag.

## Observations (no finding)

- The base `FROM` is pinned by digest (`steamcmd/steamcmd@sha256:…`, `Dockerfile:42`), and Dependabot has a `docker` entry for both image directories (`dependabot.yml:289-304`).
- On a master push, the game job builds `FROM` the base's pushed digest, and tags are attached only after `cosign verify` of the digest. A tag never names an unsigned image.
- The signing job takes the `release-signing` environment only when not a pull request. A `workflow_dispatch` from a non-master branch is bounded by the environment's deployment policy.
- The game image runs as the base's `gameserver` (UID 10000) and `entrypoint.sh` `exec`s the game, so SIGTERM reaches it.
- `STEAM_APPID` and the other values are passed as separate array elements to `steamcmd`, not through a shell string.

## Candidate findings

### C-images-01: `steam-install.sh` treats a failed update as success whenever the sentinel file already exists

- **Location**: `images/common/steamcmd/steam-install.sh:83-93`, reached from `images/games/nuclear-option/entrypoint.sh:32-50`
- **Category**: correctness
- **Suggested severity**: S3
- **Observation**:
  1. `UPDATE_ON_BOOT=true` (the default) sets `STEAM_SKIP_IF_INSTALLED=false`, so steamcmd runs on every start with the game binary already on the volume.
  2. After steamcmd returns, the script checks only that the sentinel file exists. On an existing install it always does. `steamcmd_exit` is read only on the failure path at `:98`.
  3. So a failed or half-finished update (CDN error, disk full, killed mid-download) prints "success" and `exit 0` on the first attempt, with no retry, and the server starts on a partly updated install.
- **Expected**: The README (`:110`) says steamcmd's exit code is not trusted "alone", which implies it is used too. On an update run, a non-zero exit should retry and, after the last attempt, fail or at least warn.
- **Actual**: The exit code is ignored once the sentinel exists. The retry loop only ever helps a first install.

### C-images-02: The README "Dockerfile template" fails to build on the non-root base

- **Location**: `images/README.md:60-61`, compared with `images/games/nuclear-option/Dockerfile:23-27` and `images/common/steamcmd/Dockerfile:117`
- **Category**: documentation / correctness
- **Suggested severity**: S3
- **Observation**: The template does `COPY entrypoint.sh /entrypoint.sh` then `RUN chmod +x /entrypoint.sh`. The base ends with `USER gameserver:gameserver`, so the `RUN` runs as UID 10000 on a root-owned file and `chmod` fails with "Operation not permitted". The real nuclear-option Dockerfile avoids it with `COPY --chmod=0755` and says why in its comment (`:23-26`). A contributor following "Adding a new game" will hit the failure.
- **Expected**: The template matches the working Dockerfile.
- **Actual**: The template and the shipped game disagree.

### C-images-03: The documented `docker run -v game-data:/data` cannot write to `/data` as the non-root user

- **Location**: `images/games/nuclear-option/Dockerfile:7-12`, `images/README.md:195-199`, `images/common/steamcmd/Dockerfile` (no `/data` directory is created)
- **Category**: correctness
- **Suggested severity**: S4
- **Observation**: Neither Dockerfile creates `/data` or sets its owner. Docker initialises a new named volume from the image's content at that path, and with no directory in the image it is created root-owned. `steam-install.sh:53` (`mkdir -p`) and steamcmd then run as UID 10000 and fail. In Kubernetes the `fsGroup` setting hides this (README `:157`), which is why only the plain `docker run` usage is affected.
- **Expected**: `mkdir /data && chown` in the base, or the `docker run` example documents `--user`/a `chown`'d volume.
- **Actual**: The documented local run fails on first install.

### C-images-04: `README.md` says retries use exponential backoff; the script uses a fixed delay

- **Location**: `images/README.md:111`, `images/common/steamcmd/steam-install.sh:102-104`
- **Category**: documentation
- **Suggested severity**: S4
- **Observation**: `sleep "$STEAM_RETRY_DELAY"` waits the same 5 seconds every time. The README says "with exponential backoff".
- **Expected / Actual**: Docs and code disagree. Either one fixes it.

### C-images-05: Stale comments in `steamcmd/Dockerfile` about pinning, and in `entrypoint.sh` about `set -e`

- **Location**: `images/common/steamcmd/Dockerfile:32-41` (and `:16` for the `FROM` example), `images/games/nuclear-option/entrypoint.sh:52-54`
- **Category**: documentation
- **Suggested severity**: S4
- **Observation**:
  1. `Dockerfile:32` says "the tag above is not pinned to a digest" and tells the reader to replace the `FROM` line by hand with a digest. The `FROM` at `:42` is already a digest, and Dependabot maintains it (F-017 / PR #380).
  2. `entrypoint.sh:52-54` says that if `steam-install.sh` fails "the binary check below will catch it". With `set -euo pipefail` (`:2`), the script exits at the failing `steam-install.sh` line, so the check below is only reached on success. The check is harmless but never the thing that catches a failure.
- **Expected**: Comments match the code.
- **Actual**: Comments describe an earlier state.

### C-images-05b: `UPDATE_ON_BOOT` accepts only the exact string `true`

- **Location**: `images/games/nuclear-option/entrypoint.sh:32-38`
- **Category**: correctness
- **Suggested severity**: S4
- **Observation**: `True`, `1`, `yes` and `TRUE` all fall into the `else` branch and silently disable updates. Operators setting the variable from a template or the dashboard have no validation.
- **Expected**: Normalise the value or reject unknown ones.
- **Actual**: Silent fall-through to "skip updates".

### C-images-06: Published base and game images carry no release version and are never tagged by `release.yaml`

- **Location**: `.github/workflows/images.yaml:79-81,95-96,277-283,297-299` (tags and `VERSION=dev`), `.github/workflows/images.yaml:3-14` (no `tags:` trigger)
- **Category**: release-pipeline
- **Suggested severity**: S3
- **Observation**:
  1. The workflow runs on master pushes and manual dispatch only. Its tags are the branch name and `latest`. No `v0.3.0-rc.N` or `v0.3.0` tag is ever pushed for `common-steamcmd` or `nuclear-option`.
  2. `VERSION=dev` is hard-coded, so every published image's `org.opencontainers.image.version` label says `dev`.
  3. A consumer can pin by digest (the README's recommended path), but nothing ties a digest to a release, and the release page and `docs/` cannot name a version. The 12 chart images get this from `release.yaml`; these two do not.
- **Expected**: Either these images take release tags and a real `VERSION`, or the docs say plainly that they are digest-only and unversioned.
- **Actual**: The README says "Official module templates pin game images by digest", which is true, but the label `dev` on a release-pipeline image is misleading in `docker inspect`.

### C-images-07: README "Workflow triggers" omits the pull-request trigger and the manual-build command lines do not run from the repo root

- **Location**: `images/README.md:220-228`, `images/README.md:189-193`
- **Category**: documentation
- **Suggested severity**: S4
- **Observation**:
  1. The trigger list names `workflow_dispatch` and pushes only. `images.yaml:5-8` also builds on `pull_request` for `images/**` (without publishing).
  2. `docker build -t … -f Dockerfile images/common/steamcmd` takes `-f` relative to the current directory, not the context. The repo root has no `Dockerfile`, so the command as written fails. It works only from inside the image directory, with the context as `.`.
- **Expected / Actual**: README and workflow/behaviour disagree.

### C-images-08: A comment in `images.yaml` still says the 10-minute timeout is "pending maintainer confirmation"

- **Location**: `.github/workflows/images.yaml:29`
- **Category**: documentation
- **Suggested severity**: S4
- **Observation**: `timeout-minutes: 10 # extension per D-A, pending maintainer confirmation`. Spec 008 is archived as `done_008-…` and D-A is recorded there (`OPEN-DECISIONS.md:53`). The base job builds two architectures under QEMU, so whether 10 minutes is enough on a cold cache is also unmeasured. I did not check the D-A measurements for this job.
- **Expected**: A resolved comment, or a measured number.
- **Actual**: Leftover comment.
