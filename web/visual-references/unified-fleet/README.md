# Historical unified fleet implementation reference

`DWztv.png` preserves the reviewed original browser capture of the light mobile server list. It records the compact Filter trigger and the cluster/namespace line on each server card described in [the unified dashboard design](../../../docs/unified-dashboard.md), under the approved existing-components design exception. It is historical provenance, not an active CI baseline.

Location now belongs inside the Filter popover, so the separate location row is removed and Search and the server cards move upward. The identity lines remain visible. Search, status badges and visible server cards remain readable without overlap or horizontal clipping. Further cards remain reachable by normal vertical scrolling. The desktop and narrow mobile browser checks separately verify overflow and resource identity.

This is a browser implementation reference, not a Pencil export. Its source commit, workflow run, artifact and SHA256 are recorded in `manifest.json`. The original PNG bytes were copied directly from the CI artifact without resizing, cropping or retouching. `SHA256SUMS` is retained to verify the exact historical file. `design.pen` and its exports in `design-export/` are the canonical design sources.

The visual workflow previously overlaid this capture on a temporary copy of the Pencil references. That overlay has been removed. CI now compares browser captures directly against `design-export/screenshots` with `npm run diff:screenshots -- --check-expected`. The original browser captures artifact is retained for diagnostics only. The comparison thresholds, regional checks, masks, scale checks and expected capture requirements remain unchanged.

Keep this PNG, manifest and hashes as evidence of the earlier review. Future baseline changes must be made through Pencil and exported to `design-export/`; a failed comparison must not replace the design baseline with a browser capture.
