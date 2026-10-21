# Contract: `hack/ci_scope.py` modes and outputs

`ci_scope.py` stays Python stdlib only and never installs Go or Node. Existing modes
(`modules`, `images`, `bots`, `dockerfiles`) keep their current outputs; this feature
adds keys and modes, it does not rename any.

## Inputs (all modes that diff)

| Input | Source |
|---|---|
| `--base`, `--head` | `github.event.pull_request.base.sha \|\| github.event.before`, `github.sha` |
| `GITHUB_REF`, `GITHUB_EVENT_NAME` | Actions env |
| `CI_FULL_RUN` | `"true"` when the PR has the `ci: full-run` label (`contains(github.event.pull_request.labels.*.name, 'ci: full-run')`) |

Full scope when: `GITHUB_REF == refs/heads/master`, `GITHUB_EVENT_NAME ==
workflow_dispatch`, `CI_FULL_RUN == true`, the diff is `None`, a path is unmapped, or a
path forces everything (research R8).

## Mode `modules` — added keys

| Key | Type | Meaning |
|---|---|---|
| `scope` | `"full"` \| `"partial"` | Run-level scope (FR-009 switch). Emitted by mode `summary`, not `modules` (see below) |
| `go-packages` | JSON object: module → list of package patterns | `["./..."]` on full scope or when a module's own `go.mod`/`go.sum` changed |
| `web-tests` | JSON: `"all"` or list of `web/src/...` source paths for `vitest related` | |
| `e2e-exclude` | Not emitted (decision 2). Unselected legs skip through `RUN_REGEX` instead | Would have been read by `e2e-go` and `e2e-go-arm64` |
| `e2e-run` | JSON object: bucket → anchored `-run` regex of the selected tests | Emitted by mode `e2e`. Replaces `buckets.sh regex <bucket>` in the e2e jobs (OD-6); equals it on full scope |
| `e2e-multicluster`, `e2e-upgrade`, `e2e-web-live`, `e2e-capture-overhead` | `"true"`/`"false"` | Emitted by mode `e2e`. Folded into the existing `combine` step booleans |
| `ci-jobs` | JSON list of job ids selected by the `ci.yaml` line mapper | Folded into each job's gate |
| `unmapped` | JSON list of paths | Emitted by mode `e2e`. Non-empty forces full scope and fails `selection-rules` |
| `selection` | JSON list of SelectionResult | Emitted by mode `summary`; read by the summary table and `report` |

## Mode `summary` — new

Last step of the `changes` job, after `combine`. It reads the real gate values, so it
never re-implements paths-filter in Python. Its failure never blocks CI (the step is
`continue-on-error: true`).

Env inputs (JSON strings from `toJSON(steps.*.outputs)` unless noted):

| Env | Source |
|---|---|
| `SCOPE_OUTPUTS` | `steps.scope.outputs` (mode `modules` keys) |
| `FILTER_OUTPUTS` | `steps.filter.outputs` (paths-filter with `list-files: json`, giving `<filter>_files`) |
| `GATE_OUTPUTS` | `steps.combine.outputs` (the final gate booleans) |
| `BOT_MATRIX` | `steps.bots.outputs.matrix` |
| `CI_FULL_RUN` | As in Inputs above |

Outputs (written to `$GITHUB_OUTPUT`):

| Key | Type | Meaning |
|---|---|---|
| `scope` | `"full"` \| `"partial"` | Run-level scope, same value as the summary header |
| `reason` | string | The full-scope reason (`master push`, `manual run`, `label ci: full-run`, `unknown base`, `selection-rule change`, `error: <exception>`, `selection fallback (see detect changes log)`), or empty on partial scope |
| `selection` | JSON list of SelectionResult | One record per suite and job gate |

## Mode `e2e` (new)

Emits the e2e selection that `e2e_outputs()` computes from
[suite-components.md](./suite-components.md) and the changed paths. The `e2e` step of the
`changes` job runs right after `bots`. Outputs (written to `$GITHUB_OUTPUT`):

| Key | Type | Meaning |
|---|---|---|
| `all` | boolean | True when `reason` is set; `combine` folds it into `CI` |
| `reason` | string | Why every e2e test runs (a full-scope reason or `unmapped path <p>`), or empty |
| `unmapped` | JSON list of paths | Changed paths that no rule maps; non-empty forces full scope |
| `e2e-run` | JSON object: bucket → `-run` regex | Selected tests of each bucket |
| `e2e-why` | JSON object: bucket or test → string | "N of M tests" and the changed paths that chose them, for the summary |
| `e2e-multicluster`, `e2e-upgrade`, `e2e-web-live`, `e2e-capture-overhead` | boolean | Gates for the matching jobs |
| `bot-matrix` | JSON object `{"test": [...]}` | Selected `bot-fast` tests, read by `e2e-game-bot` |
| `bots-any` | boolean | False skips `e2e-game-bot` |

## Mode `bots` — changed

Emits only the selected `bot-fast` tests. When none is selected it emits an empty
matrix and `bots-any=false`. The matrix that `e2e-game-bot` reads comes from mode `e2e`, which emits its own `bot-matrix` and `bots-any`.

## Mode `verify-suites` — new

Exit 0 when [suite-components.md](./suite-components.md) rules 1, 2 and 5 hold; otherwise
exit 1 and print each violation. No diff needed.

## Step summary

Appended to `$GITHUB_STEP_SUMMARY` by mode `summary`:

```markdown
### Test selection (partial)
| Suite | Decision | Why |
|---|---|---|
| e2e operator (amd64, arm64) | run | changed: agent/internal/console/pty.go |
| e2e telemetry (amd64, arm64) | skip | not affected |
| go api | run | changed: api/internal/handlers/servers.go (+2) |
```

A full-scope run prints one line stating why (`full: master push`, `full: label ci:
full-run`, `full: unmapped path tools/new.sh`, …) followed by the table with every row
`run`.

## Failure behaviour

Any exception in diffing or parsing selects full scope and prints the exception in the
summary. It never selects less than today.
