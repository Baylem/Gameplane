# Contract: per-PR move brief

`specs/028-split-big-packages/briefs/<NN>-<slug>.md`, written by the scout, read by the
move agent (blind, rule 18), checked by the reviewer and by `hack/move_parity.py test-edits`.

## Sections

1. **Scope**: module, package or screen; task IDs; base SHA.
2. **Open PRs on these paths** (number, title) or "none".
3. **Moves**: table `source path | destination path | whole file or declarations (names)`.
4. **Exports**: table `old identifier | new identifier | package` for every name that crosses a new package line. This is the identifier map the test-edit check uses.
5. **New files**: helper package files, `suite_envtest_test.go` copies (with the exact relative-path change, OD-9), `pages/<Screen>.tsx` composers.
6. **Path references**: every config, Dockerfile, Makefile, chart, docs, `specs.md` and 023 mapping line to change, with before/after text.
7. **Test files touched**: list, and for each the edit kinds used (OD-6 only).
8. **Expected checks**: `go build ./...` / `npx tsc --noEmit` targets; `move_parity.py` commands.

## Rules

- Facts are verified once, here (rule 18). The move agent does not re-read sources.
- Anything that does not fit the R1 change kinds stops the PR and goes to valgul.
