import type { MessageFormatElement } from "react-intl";

// LanguageInfo is one row of the language manifest computed by the shared Go
// checker (`i18n-check --manifest`); the dashboard never derives listing rules
// itself.
export interface LanguageInfo {
  tag: string;
  nativeName: string;
  englishName: string;
  direction: "ltr" | "rtl";
  completeness: number;
  listed: boolean;
}

export const ENGLISH: LanguageInfo = {
  tag: "en",
  nativeName: "English",
  englishName: "English",
  direction: "ltr",
  completeness: 100,
  listed: true,
};

// CompiledCatalog maps each message id to its compiled AST (formatjs --ast).
export type CompiledCatalog = Record<string, MessageFormatElement[]>;

const manifests = import.meta.glob<LanguageInfo[]>("/src/generated/i18n/languages.json", {
  eager: true,
  import: "default",
});

const catalogs = import.meta.glob<Record<string, unknown>>(
  [
    "/src/generated/i18n/*.json",
    "!/src/generated/i18n/languages.json",
    "!/src/generated/i18n/extracted.json",
  ],
  { import: "default" },
);

function isLanguageInfo(value: unknown): value is LanguageInfo {
  if (typeof value !== "object" || value === null) return false;
  const row = value as Partial<LanguageInfo>;
  return (
    typeof row.tag === "string" &&
    typeof row.nativeName === "string" &&
    typeof row.englishName === "string" &&
    (row.direction === "ltr" || row.direction === "rtl") &&
    typeof row.completeness === "number" &&
    typeof row.listed === "boolean"
  );
}

// parseManifests flattens the eagerly loaded manifest modules into one list
// that always contains English. Pure so tests can pass fixtures.
export function parseManifests(records: Record<string, LanguageInfo[] | undefined>): LanguageInfo[] {
  const rows: LanguageInfo[] = [];
  for (const value of Object.values(records)) {
    if (!Array.isArray(value)) continue;
    for (const row of value) {
      if (isLanguageInfo(row)) rows.push(row);
    }
  }
  if (rows.some((row) => row.tag === ENGLISH.tag)) return rows;
  return [ENGLISH, ...rows];
}

let allCache: LanguageInfo[] | null = null;

// allLanguages returns the manifest entries (or just English when the
// generated manifest is absent, e.g. before `npm run build`).
export function allLanguages(): LanguageInfo[] {
  allCache ??= parseManifests(manifests);
  return allCache;
}

export function listedLanguages(): LanguageInfo[] {
  return allLanguages().filter((row) => row.listed);
}

export function isListed(tag: string): boolean {
  return listedLanguages().some((row) => row.tag === tag);
}

// directionFor returns the text direction for a tag; unknown tags are ltr.
export function directionFor(tag: string, languages: readonly LanguageInfo[] = allLanguages()): "ltr" | "rtl" {
  return languages.find((row) => row.tag === tag)?.direction ?? "ltr";
}

// CatalogLoaders maps a glob path to a lazy loader, as import.meta.glob returns.
export type CatalogLoaders = Record<string, () => Promise<unknown>>;

// loadCatalogFrom resolves one locale's compiled chunk from a loader map, or
// null when no chunk was generated for the tag.
export async function loadCatalogFrom(loaders: CatalogLoaders, tag: string): Promise<CompiledCatalog | null> {
  const loader = loaders[`/src/generated/i18n/${tag}.json`];
  if (!loader) return null;
  const data = await loader();
  if (typeof data !== "object" || data === null || Array.isArray(data)) return null;
  // The chunk is produced by `formatjs compile --ast` at build time, so every
  // value is a MessageFormatElement[]; the runtime cannot check that cheaply.
  return data as CompiledCatalog;
}

export function loadMessages(tag: string): Promise<CompiledCatalog | null> {
  return loadCatalogFrom(catalogs, tag);
}
