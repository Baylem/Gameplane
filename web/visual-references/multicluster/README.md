# Historical multicluster implementation references

These two PNGs preserve the reviewed browser implementation snapshots from the user-authorized existing-components design exception in [the multicluster design decision](../../../docs/multicluster-ui.md). They are historical provenance, not active CI baselines or Pencil exports. `design.pen` and its exports in `design-export/` are the canonical design sources.

- `Wj0V4.png`: the Mod registries settings screen retains its registry controls and adds the central-management notice, Clusters navigation item and explicit cluster selector label. The existing Theme settings entry is retained.
- `t3IY3u.png`: the Edit user dialog adds cluster selection, scoped grant controls and explanations. Its increased height is intentional; the original profile fields and save/cancel actions remain visible.

The source capture run, commit, artifact and per-file SHA256 hashes are recorded in `manifest.json`. The PNGs were copied directly from the CI `browser-screenshot-captures` artifact. They were not resized, cropped from composites, retouched or exported from Pencil. `SHA256SUMS` is retained to verify the exact historical files.

The visual workflow previously overlaid these two captures on a temporary copy of the Pencil references. That overlay has been removed. CI now compares browser captures directly against `design-export/screenshots` with `npm run diff:screenshots -- --check-expected`. The original browser captures artifact is retained for diagnostics only. The comparison thresholds, masks, scale gates and captured/expected screen checks remain unchanged.

Keep these PNGs, manifests and hashes as evidence of the earlier review. Future baseline changes must be made through Pencil and exported to `design-export/`; a failed comparison must not replace the design baseline with a browser capture.
