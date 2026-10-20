# Quickstart: reviewing a spec 028 move PR

1. Read the brief in `specs/028-split-big-packages/briefs/` for the PR.
2. `git diff -M --stat origin/master...HEAD`: most files should show as renames.
3. Check the PR body's Parity section: test names equal, test-edit check pass, generated outputs unchanged.
4. Check CI ran full (`ci: full-run` label) and is green on amd64 and arm64, coverage gates included.
5. Web PRs: visual-diff and screenshot jobs green; the sonnet browser smoke test notes in the PR.
6. Merge, delete the branch (rule 12); the next PR in the phase starts from the new master.
