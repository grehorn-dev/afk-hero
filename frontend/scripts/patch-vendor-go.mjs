// Some third-party npm packages (e.g. `flatted`) ship stray Go source files
// in their published tarball (under `golang/pkg/flatted/`). Those files
// become visible to the parent Go module's `go list ./...` and confuse
// tooling — the package shows up as `afk-hero/frontend/node_modules/...`.
//
// Go has no first-class way to ignore a subtree, but prepending a
// `//go:build ignore` directive to each such file keeps the Go toolchain
// from parsing them. This script runs from the `postinstall` hook of
// `frontend/package.json`, so the patch is reapplied on every `npm ci`.
//
// The script is deliberately dependency-free (uses only Node's built-in
// `fs`/`path`) so it can run in the CI container without extra packages.

import { existsSync, readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const BUILD_TAG = '//go:build ignore';

function walk(dir) {
  if (!existsSync(dir)) return [];
  const out = [];
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    const s = statSync(full);
    if (s.isDirectory()) {
      out.push(...walk(full));
    } else if (s.isFile() && full.endsWith('.go')) {
      out.push(full);
    }
  }
  return out;
}

function patch(file) {
  const content = readFileSync(file, 'utf8');
  if (content.startsWith(BUILD_TAG)) {
    return false;
  }
  writeFileSync(file, `${BUILD_TAG}\n\n${content}`, 'utf8');
  return true;
}

const here = fileURLToPath(new URL('.', import.meta.url));
const nodeModules = join(here, '..', 'node_modules');

const candidates = walk(nodeModules);
let patched = 0;
for (const file of candidates) {
  if (patch(file)) {
    patched += 1;
  }
}

if (patched > 0) {
  console.log(`patch-vendor-go: marked ${patched} stray Go file(s) as //go:build ignore`);
}
