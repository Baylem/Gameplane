# Multicluster implementation references

These two PNGs are reviewed browser implementation snapshots, not Pencil exports. They record the intentional UI changes made under the user-authorized existing-components design exception in [the multicluster design decision](../../../docs/multicluster-ui.md). `design.pen` and `design-export/` remain the canonical Pencil sources and are untouched by this exception.

- `Wj0V4.png`: the Mod registries settings screen retains its registry controls and adds the central-management notice, Clusters navigation item and explicit cluster selector label. The existing Theme settings entry is retained.
- `t3IY3u.png`: the Edit user dialog adds cluster selection, scoped grant controls and explanations. Its increased height is intentional; the original profile fields and save/cancel actions remain visible.

The source capture run, commit, artifact and per-file SHA256 hashes are recorded in `manifest.json`. The PNGs are copied directly from the CI `browser-screenshot-captures` artifact. They are not resized, cropped from composites, retouched or exported from Pencil. `SHA256SUMS` verifies the exact committed files before use.

The visual workflow copies all Pencil references to a temporary directory, verifies these hashes, overlays only these two exact filenames, and passes that directory to the existing comparator through `--ref-dir`. The 4% global, 16% regional and scale gates remain unchanged, as do the captured/expected screen checks. Other screen references continue to come from Pencil. Missing or modified reference files fail verification; a failed comparison never automatically updates them.

A future reference change requires reviewing the complete original browser capture, recording why its changed UI is accepted and updating the provenance and hashes together. When these screens are reconciled with Pencil, export their designs through Pencil and remove this implementation-reference overlay.
