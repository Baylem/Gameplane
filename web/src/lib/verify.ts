import type { CatalogEntry, ModuleSource, ModuleVerifySpec } from "@/types";

export type VerifyMode = "keyed" | "keyless" | "none";

// verifyMode classifies a source's cosign policy, mirroring
// ModuleSource.spec.verify: keyless (Fulcio) takes precedence over a keyed
// public key if both are somehow present; an absent verify block means the
// source does not require signatures.
export function verifyMode(spec?: ModuleVerifySpec): VerifyMode {
  if (spec?.keyless) return "keyless";
  if (spec?.key) return "keyed";
  return "none";
}

// EntryVerify is the catalog-card view of a module's verification posture.
// ModuleCard's VerifyBadge maps it: enforced -> solid "verified"; mode!=none
// && !enforced && !mixed -> outline "policy" (declared, not yet checked);
// mixed || none -> no badge (suppressed, never over-claims).
export interface EntryVerify {
  // mode is the representative policy to badge.
  mode: VerifyMode;
  // enforced is true only for an installed entry whose Module status records a verification (verifiedDigest), i.e. the operator actually signature-checked the installed digest. The source's current policy alone never sets it. It is false before install.
  enforced: boolean;
  // mixed flags a not-yet-installed entry whose candidate sources disagree on
  // policy, so a single badge would over-claim.
  mixed: boolean;
}

// verifyForEntry derives the verification posture for one catalog row by
// joining it against the live ModuleSource list (already fetched by the
// Modules page). The join lives client-side on purpose: the operator records
// whether a verification ran (CatalogEntry.verifiedDigest), but not the
// policy of sibling sources; the per-source policy join stays client-side so
// the API remains a pure read/map (rule 10). For an installed entry the
// authoritative source is the one it was pulled from; otherwise we summarise
// across the candidate sources.
export function verifyForEntry(
  entry: CatalogEntry,
  sources: ModuleSource[],
): EntryVerify {
  const byName = new Map<string, ModuleSource>();
  for (const s of sources) {
    if (s.metadata?.name) byName.set(s.metadata.name, s);
  }

  if (entry.installed && entry.installedFrom) {
    const enforced = !!entry.verifiedDigest;
    // enforced reflects a recorded verification (the operator actually ran
    // cosign.Verify at install time), not the source's current policy: a
    // policy added after install must not retroactively claim bytes that
    // were never checked. When enforced, mode must come from the *recorded*
    // verifyPolicy too — the source's current policy can have changed since
    // (e.g. keyed -> keyless), which would otherwise mislabel what was
    // actually checked. Only the unenforced, declared-but-unchecked badge
    // reads the live source policy.
    // The recorded policy is narrowed rather than cast: an unexpected value
    // falls back to "none" (no badge) instead of rendering a verified chip.
    const recorded = entry.verifyPolicy;
    const mode: VerifyMode = enforced
      ? recorded === "keyless" || recorded === "keyed"
        ? recorded
        : "none"
      : verifyMode(byName.get(entry.installedFrom)?.spec.verify);
    return { mode, enforced, mixed: false };
  }

  const modes = (entry.sources ?? []).map((ref) =>
    verifyMode(byName.get(ref.name)?.spec.verify),
  );
  const mixed = modes.some((m) => m !== modes[0]);
  const mode = modes.find((m) => m !== "none") ?? "none";
  return { mode, enforced: false, mixed };
}
