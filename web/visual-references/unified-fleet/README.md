# Unified fleet implementation reference

`DWztv.png` is the reviewed original browser capture of the light mobile server list. It records the location filter and the cluster/namespace line on each server card described in [the unified dashboard design](../../../docs/unified-dashboard.md), under the approved existing-components design exception.

The filter and identity lines intentionally increase the vertical space. Search, status badges and visible server cards remain readable without overlap or horizontal clipping. Further cards remain reachable by normal vertical scrolling. The desktop and narrow mobile browser checks separately verify overflow and resource identity.

This is a browser implementation reference, not a Pencil export. Its source commit, workflow run, artifact and SHA256 are recorded in `manifest.json`. The original PNG bytes are copied directly from the CI artifact without resizing, cropping or retouching. `design.pen` and `design-export/` remain unchanged.

The visual workflow verifies `SHA256SUMS` and overlays only this named image in its temporary reference directory. All other references, comparison thresholds, regional checks, masks, scale checks and expected capture requirements remain unchanged. A future change requires reviewing the complete original capture and updating the provenance and hash together. Remove this overlay once the design is reconciled through Pencil.
