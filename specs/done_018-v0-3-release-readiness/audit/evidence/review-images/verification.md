# T045 images chunk: independent verification (opus)

**Method.** I tried to refute each candidate in `notes.md` (written by a sonnet reviewer) against branch `018-v0-3-release-readiness` at `1cd82d3c`.

What I read:
- `images/README.md`, `images/common/steamcmd/Dockerfile`, `images/common/steamcmd/steam-install.sh`, `images/games/nuclear-option/Dockerfile` and `images/games/nuclear-option/entrypoint.sh`, in full.
- `.github/workflows/images.yaml:1-120` and `:180-300` (triggers, `common-base`, `game-images`, tag metadata and build args).
- Spec 008 `OPEN-DECISIONS.md` (the measurement table and D-A, `:20-80`), and a grep of `release.yaml`, `docs/` and the spec 018 docs for any requirement to version-tag the game images.

The `modules/` submodule is empty in this worktree, so no module template was checked for `UPDATE_ON_BOOT` or for the image pin. I built and ran no image and ran no test, lint or workflow. This component has no security-sensitive candidates.

| Candidate | Verdict (kept/rejected) | Severity | Reason |
|---|---|---|---|
| C-images-01 | kept | S3 | Confirmed. `UPDATE_ON_BOOT=true` (the default, `entrypoint.sh:13`) sets `STEAM_SKIP_IF_INSTALLED=false` (`:32-33`). On every boot after the first, the sentinel `NuclearOptionServer.x86_64` already exists. `steam-install.sh:83-93` records a non-zero steamcmd exit and then exits 0 with "success" because the sentinel is there, so a failed or partial update is reported as success on attempt 1, and the retry loop never runs. The server then starts on whatever files are on the volume. A restart retries, so S3. |
| C-images-02 | kept | S3 | Confirmed. The base ends with `USER gameserver:gameserver` (`steamcmd/Dockerfile:117`). In the README template (`README.md:60-61`), `COPY` without `--chown` creates a root-owned `/entrypoint.sh`, and `RUN chmod +x` runs as `gameserver`, so `chmod` fails with "Operation not permitted" and the build stops. The shipped Dockerfile avoids this with `COPY --chmod=0755` (`nuclear-option/Dockerfile:23-27`), and its comment states this exact failure. That Dockerfile is the workaround, so S3. |
| C-images-03 | kept | S4 | Confirmed. Neither Dockerfile creates `/data`. With `docker run -v game-data:/data` (`README.md:195-199`, `nuclear-option/Dockerfile:9-12`), Docker creates the mount point root-owned in a fresh named volume, and the entrypoint runs as UID 10000. `mkdir -p` (`steam-install.sh:53`) succeeds because the directory exists, but steamcmd can't write into it. In Kubernetes, `fsGroup` (`README.md:157`) hides this. It affects only the documented local `docker run`, so S4. |
| C-images-04 | kept | S4 | Confirmed. `README.md:111` says "with exponential backoff". `steam-install.sh:102-104` sleeps a fixed `STEAM_RETRY_DELAY` each time, and the README's own table (`:126`) describes it as "Seconds to wait between retries". The script comment at `:102` ("Retry with backoff") is loose in the same way. Documentation only. |
| C-images-05 | rejected | n/a | Both comments are stale. `steamcmd/Dockerfile:32` says "the tag above is not pinned", but `:42` is a digest. `entrypoint.sh:52-54` says the binary check "will catch" a failed `steam-install.sh`, but under `set -e` the script exits first. Neither changes behaviour: the image is pinned, and a failed install still stops the container with a non-zero exit. These are stale internal comments, which is style (the same call as C-hack-03). |
| C-images-05b | rejected | n/a | `UPDATE_ON_BOOT` uses the same exact-`true` test as `STEAM_VALIDATE` and `STEAM_SKIP_IF_INSTALLED` (`steam-install.sh:47,69`), and the README's variable table documents only `true`/`false`. No caller that passes `True`, `1` or `yes` exists in the tree. This is speculative input. |
| C-images-06 | rejected | n/a | `images.yaml:79-81` and `:280-283` do tag only the branch and `latest`, and `:95-96` and `:297-299` do pass `VERSION=dev`. But the documented way to use these images is by digest (`README.md:11`, `:48`, `:93-102`), and they version independently of the Gameplane release. Nothing in `release.yaml`, `docs/` or spec 018 asks for release tags. The `dev` value in the OCI version label is cosmetic metadata, so S3 is overstated and nothing here is a defect against a stated requirement. |
| C-images-07 | kept | S4 | Confirmed, both halves. The README trigger list (`README.md:220-228`) leaves out the `pull_request` trigger (`images.yaml:5-8`, which builds without publishing), and it doesn't say the `push` trigger is limited to `master` (`:9-11`). In `docker build -f Dockerfile images/common/steamcmd` (`README.md:190`, `:193`), `-f` resolves against the current directory, not the context. Run from the repo root, where there is no `Dockerfile`, the build fails. Documentation only. |
| C-images-08 | rejected | n/a | The comment at `images.yaml:29` ("pending maintainer confirmation") is stale. D-A is **RESOLVED** (spec 008 `OPEN-DECISIONS.md`, 2026-08-30), and the measured `common-base` duration is 1m median and 1m max (`:34-48`), well inside 10 minutes. A stale comment with no effect on behaviour is style. |

### C-images-01

**Location:** `images/games/nuclear-option/entrypoint.sh:13`, `:32-38` and `:43-50`, and `images/common/steamcmd/steam-install.sh:47-50`, `:78-106`.

**Repro / observation:**
1. Start a pod once so the game installs, and `/data/NuclearOptionServer.x86_64` exists on the volume.
2. Restart it with the default `UPDATE_ON_BOOT=true`. `entrypoint.sh:32-33` passes `STEAM_SKIP_IF_INSTALLED=false`, so `steam-install.sh:47` doesn't skip.
3. Attempt 1 runs steamcmd. Suppose it fails partway (a CDN error, or the volume fills up) and exits non-zero. `:83-87` stores `steamcmd_exit`, which nothing reads unless the last attempt fails.
4. `:90-92` finds the sentinel left by the earlier install, prints "success — sentinel file … found" and exits 0. Attempts 2 and 3 never run, and the server starts on the old or partly updated files.

**Expected:** `README.md:110-112` says the script verifies success and retries on transient failure. When the sentinel existed before the run, a non-zero steamcmd exit should count as a failed attempt.

**Actual:** After the first install the sentinel check is always true, so update failures are reported as success and are never retried.

### C-images-02

**Location:** `images/README.md:52-65` (the Dockerfile template; the failing lines are `:60-61`), compared with `images/common/steamcmd/Dockerfile:117` and `images/games/nuclear-option/Dockerfile:23-27`.

**Repro / observation:**
1. Follow "Adding a new game" (`README.md:37-48`) and copy the Dockerfile template.
2. `FROM ${STEAMCMD_BASE_IMAGE}` inherits `USER gameserver:gameserver` from the base (`steamcmd/Dockerfile:117`).
3. `COPY entrypoint.sh /entrypoint.sh` creates the file owned by `root:root`.
4. `RUN chmod +x /entrypoint.sh` runs as `gameserver`. Only the owner or root can `chmod`, so it fails with `Operation not permitted`, and `docker build` stops.

**Expected:** The template builds. For example `COPY --chmod=0755 entrypoint.sh /entrypoint.sh`, as `nuclear-option/Dockerfile:27` does.

**Actual:** The documented template fails to build on top of the documented base.

### C-images-03

**Location:** `images/README.md:195-199` and `images/games/nuclear-option/Dockerfile:9-12` (the documented `docker run -v game-data:/data`). Neither `images/common/steamcmd/Dockerfile` nor `images/games/nuclear-option/Dockerfile` creates `/data`.

**Repro / observation:**
1. Build both images as documented, then run `docker run -it -v game-data:/data gameplane-nuclear-option:latest` with a new named volume.
2. `/data` doesn't exist in the image, so Docker creates the mount point and the volume root is `root:root 0755`.
3. The entrypoint runs as UID 10000 (`steamcmd/Dockerfile:117`). `mkdir -p /data` (`steam-install.sh:53`) succeeds, but steamcmd can't write the install into `/data`. Every attempt leaves no sentinel, and the script exits 1 after three tries.
4. In Kubernetes, `fsGroup: 10000` (`README.md:157`) makes the PVC group-writable, so the problem doesn't show there.

**Expected:** The documented local run works. For example, the base creates `/data` owned by `gameserver` (so Docker seeds a new named volume with that ownership), or the README says to pre-create the volume with the right owner.

**Actual:** The documented local `docker run` fails to install the game.

### C-images-04

**Location:** `images/README.md:111`, compared with `images/common/steamcmd/steam-install.sh:28`, `:102-104` and `README.md:126`.

**Repro / observation:**
1. Read `README.md:111`: "up to `STEAM_RETRY_COUNT` times (default 3) with exponential backoff."
2. Read `steam-install.sh:103-104`: `sleep "$STEAM_RETRY_DELAY"`. The delay is the same fixed value (default 5 s, `:28`) between every pair of attempts.

**Expected:** The README describes a fixed delay, as its own variable table does (`:126`), or the script implements backoff.

**Actual:** The README promises exponential backoff, and the script uses a fixed delay.

### C-images-07

**Location:** `images/README.md:220-228` (trigger list) and `:188-193` (manual build commands), compared with `.github/workflows/images.yaml:3-14`.

**Repro / observation:**
1. Read `images.yaml:3-14`. The triggers are `workflow_dispatch`, `pull_request` on `images/**` and the workflow file, and `push` to `master` on the same paths.
2. Read `README.md:222-226`. The list has `workflow_dispatch` and two `push` lines, with no `pull_request` and no `master` limit.
3. From the repo root, run `docker build -t gameplane-steamcmd-base:latest -f Dockerfile images/common/steamcmd` (`README.md:190`). Docker resolves `-f Dockerfile` against the current directory, there is no `./Dockerfile`, and the build fails before it starts. `:193` has the same form.

**Expected:** The trigger list matches `images.yaml`. The manual commands either drop `-f` (the context's `Dockerfile` is the default) or pass `-f images/common/steamcmd/Dockerfile`.

**Actual:** The list leaves out the PR build, and the copy-paste build commands fail from the repo root.
