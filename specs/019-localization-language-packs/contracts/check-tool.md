# Contract: `i18n-check` CI tool

**Feature**: `019-localization-language-packs` | **Binding modules**: `i18n/cmd/i18n-check`, `.github/workflows/ci.yaml`, module repo CI, `gp-module validate` | **Status**: Binding

## Invocation

```text
go run ./i18n/cmd/i18n-check [flags] <locales-dir>
  --reference en            reference tag (default en)
  --base <dir>              base-branch copy for before/after completeness
  --extracted <file>        formatjs extract output to compare with the reference (drift)
  --sync-server             write api/internal/messages registry into the reference
  --check-server            fail if the registry and the reference differ
  --used-ids <file>         ids referenced by code (from extract + Go registry) → unused report
  --manifest <out.json>     write languages.json
  --stamp                   write sourceHash for translated entries
  --module <template.yaml>  module mode: ids derived from the template
  --summary <file>          GitHub job summary markdown
```

Exit 0 = pass, 1 = findings, 2 = usage/IO error.

## Findings (each names file, id and a plain-language problem — FR-015)

| Code | Severity | Meaning |
|---|---|---|
| `malformed-pack` | error | invalid JSON / wrong shape |
| `invalid-meta` | error | `meta.json` missing field, bad tag, tag ≠ directory, unknown key |
| `icu-syntax` | error | message does not parse |
| `missing-placeholder` / `extra-placeholder` | error | argument set differs from English |
| `markup` | error | tag/entity/script outside `<b>`, `<link>` |
| `unknown-id` | error | translation for an id not in the reference (stale) |
| `missing-description` | error | reference entry without translator note |
| `extract-drift` | error | committed English reference ≠ extracted |
| `server-drift` | error | Go registry ≠ `server.*` entries |
| `unused-id` | error | reference id used nowhere (FR-023) |
| `needs-review` | info | sourceHash stale, placeholders still match |

Summary table: tag, completeness (base → head), listed yes/no.

Seeded fixtures `i18n/testdata/defective/<code>/` MUST each produce exactly their finding (SC-007).
