# T045 .github/actions chunk: independent verification (opus)

**Method.** I tried to refute each candidate in `notes.md` (written by a sonnet reviewer) against branch `018-v0-3-release-readiness` at `1cd82d3c`.

What I read:
- `.github/actions/build-e2e-images/action.yml`, `dump-cluster-state/action.yml`, `e2e-images/action.yml` and `go-cache/action.yml`, in full.
- Every caller in `.github/workflows/ci.yaml`: the `build-e2e-images` calls (`:293`, `:307`), the `e2e-images` loads (`:1182`, `:1246`, `:1319`, `:1367`, `:1438`) and the seven `dump-cluster-state` calls (`:1208`, `:1273`, `:1278`, `:1333`, `:1397`, `:1457`, `:1527`), plus the `e2e-multicluster` job (`:1214-1281`).
- `.github/dependabot.yml:415-425` (the single `github-actions` entry), `docker-bake.hcl:1-30`, `.github/zizmor.yml`, and `deploy/kind/e2e.sh` (the image fallback at `:165-192` and the namespace at `:218-302`).
- The `uses:` pins of the same third-party actions in `.github/workflows/*.yaml`, for drift.

I ran no workflow, test or lint. The security-sensitive candidates for this component were verified separately and are not recorded in git (OD-019).

| Candidate | Verdict (kept/rejected) | Severity | Reason |
|---|---|---|---|
| C-github-actions-01 | kept | S3 | Confirmed. The only `github-actions` entry (`dependabot.yml:416-417`) has `directory: "/"`, which makes Dependabot scan `.github/workflows/` and a root `action.yml`, not composite actions under `.github/actions/<name>/`. Drift confirms it: `build-e2e-images/action.yml:18` pins `setup-buildx-action` v4.1.0 and `:59` pins `upload-artifact` v7.0.0, while every workflow pins v4.4.1 (`ci.yaml:349`, `images.yaml:60`, `release.yaml:57`, `publish-edge.yaml:101`) and v7.0.1 (`ci.yaml:694`). A workaround exists (bump by hand), so S3, the same rating as C-github-workflows-09. |
| C-github-actions-02 | rejected | n/a | The hard-coded `gameplane-system` is where the chart is always installed (`deploy/kind/e2e.sh:220-302`, on both multicluster clusters too), so the operator, API, helm and jobs steps look in the right place. The `namespaces` input is documented as the pod-describe list (`action.yml:11`), not a log location. The ingress-nginx job's pods are dumped: the last step (`:119-221`) prints current and previous logs for every container in each non-`gameplane-system` namespace passed in, which for `ci.yaml:1530` is `ingress-nginx`. |
| C-github-actions-03 | rejected | n/a | Confirmed that `--previous` runs for every container (`:187`, `:209`; `:74` only when `include-previous-logs` is set), but each one prints a single error line under a header that already says "(if restarted, …)" (`:186`, `:208`). That is one labelled line per container, not lost output. Style. |
| C-github-actions-04 | rejected | n/a | Confirmed that the six `redact()` copies (`:40`, `:63`, `:81`, `:95`, `:109`, `:124`) exist, and all six `-e` expressions are byte-identical today (`sort \| uniq -c` gives a count of 6 for each). Duplication with no current divergence is maintainability, not a defect. |
| C-github-actions-05 | rejected | n/a | The description (`e2e-images/action.yml:6-7`) talks about a missing *image*, and `deploy/kind/e2e.sh:168-170` and `:188-190` do rebuild any `gameplane-test/<img>:e2e` that `docker image inspect` can't find after the tar is loaded. A missing *artifact* is a different case. Every caller `needs:` the build jobs, so the artifact exists whenever the job starts, and failing fast there is the right outcome. |
| C-github-actions-06 | kept (narrowed) | S4 | Kept for the action name only. `docker-bake.hcl:1` says the file is "used by .github/actions/e2e-images", but the only bake caller is `build-e2e-images/action.yml:20,39`. `e2e-images` just downloads and loads the tar. The "six" half is rejected: `bake --load e2e` does build six targets (the `e2e` group at `:6-8`), and gameprobe is a separate bake call (`build-e2e-images/action.yml:38-46`), which is why the action saves seven. |
| C-github-actions-07 | rejected | n/a | Composite `shell: bash` steps run with `-eo pipefail`, so if `docker save` (`build-e2e-images/action.yml:47-58`) fails, the job stops before the upload step. The tar can only be missing at `:59` if the previous step already failed, so `if-no-files-found: warn` never applies. |

### C-github-actions-01

**Location:** `.github/dependabot.yml:415-425` (the single `github-actions` entry, `directory: "/"`), compared with the pins in `.github/actions/build-e2e-images/action.yml:18,20,39,59`, `.github/actions/e2e-images/action.yml:16` and `.github/actions/go-cache/action.yml:22,27`.

**Repro / observation:**
1. `grep -n 'package-ecosystem: "github-actions"' -A2 .github/dependabot.yml` gives one entry, `directory: "/"`.
2. For that ecosystem, Dependabot treats `/` as `.github/workflows/` plus a root `action.yml`/`action.yaml`. Composite actions in subdirectories are scanned only when they are listed (`directories:` or one entry per path).
3. `grep -rn 'setup-buildx-action\|upload-artifact' .github/` shows the drift. `build-e2e-images/action.yml:18` has `setup-buildx-action@d7f5e7f5… # v4.1.0`, and the workflows have `@f87e5991… # v4.4.1`. `build-e2e-images/action.yml:59` has `upload-artifact@bbbca2dd… # v7.0.0`, and the workflows have `@043fb46d… # v7.0.1`.
4. The workflow pins moved through Dependabot group bumps (for example `54b7a0d0`), and the composite-action pins did not.

**Expected:** Spec 008 US3 (`spec.md:55-59`) puts GitHub Actions under Dependabot because "stale actions expose the project to known CVEs", and SC-003 asks for no repository component to be left out. Every `uses:` in `.github/actions/*/action.yml` should get the same version-update PRs as the workflows.

**Actual:** The pins in the four composite actions are never bumped, and they already trail the workflows.

### C-github-actions-06

**Location:** `docker-bake.hcl:1`, compared with `.github/actions/build-e2e-images/action.yml:20,39` and `.github/actions/e2e-images/action.yml:1-22`.

**Repro / observation:**
1. Read `docker-bake.hcl:1`: "Bake definition for the e2e images (used by .github/actions/e2e-images)."
2. `grep -rn 'docker-bake.hcl\|bake-action' .github/` finds the bake calls only in `build-e2e-images/action.yml`.
3. `e2e-images/action.yml` has no bake step. It runs `download-artifact` and `docker load`.

**Expected:** The header names `.github/actions/build-e2e-images`.

**Actual:** It names the loader action, so a reader looking for the builder opens the wrong file.
