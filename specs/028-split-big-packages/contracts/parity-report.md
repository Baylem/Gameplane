# Contract: parity report (PR body section)

Each move PR body has a "Parity" section:

| Check | Result |
|---|---|
| Test names before/after (`move_parity.py names`) | equal, N tests |
| Test-edit check (`move_parity.py test-edits`) | pass, M test files touched, kinds: … |
| Rename stats (`git diff -M --stat`) | X files R100, Y split |
| Generated outputs (`make generate manifests` diff, `openapi.yaml`) | no diff |
| Coverage, full run | module: before → after (minimum) |
| envtest job time (Go PRs adding a `TestMain`) | before → after |
| Visual-diff / screenshots (web PRs) | unchanged |

Files whose content changed beyond R1 kinds: must be "split files only", each listed.
