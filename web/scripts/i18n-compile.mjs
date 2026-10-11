// Compiles every shared locale catalog (../i18n/locales/<tag>/app.json) into
// an AST-format JSON chunk under src/generated/i18n/<tag>.json. The dashboard
// loads these chunks at runtime via import.meta.glob (src/lib/i18n/languages.ts).
//
// Node ESM, no dependencies beyond the local @formatjs/cli binary. Skips
// silently when the shared locales directory is absent.
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, readdirSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const webRoot = path.resolve(here, "..");
const localesDir = path.resolve(webRoot, "../i18n/locales");
const outDir = path.join(webRoot, "src/generated/i18n");
const formatjsBin = path.join(webRoot, "node_modules/.bin/formatjs");

if (!existsSync(localesDir)) {
  process.exit(0);
}

mkdirSync(outDir, { recursive: true });

for (const entry of readdirSync(localesDir, { withFileTypes: true })) {
  if (!entry.isDirectory()) continue;
  const tag = entry.name;
  const source = path.join(localesDir, tag, "app.json");
  if (!existsSync(source)) continue;
  const target = path.join(outDir, `${tag}.json`);
  execFileSync(formatjsBin, ["compile", source, "--ast", "--out-file", target], { stdio: "inherit" });
  console.log(`i18n:compile ${tag} -> src/generated/i18n/${tag}.json`);
}
