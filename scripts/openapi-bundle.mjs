// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mkdirSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { createRequire } from 'node:module';

const repositoryRoot = resolve(import.meta.dirname, '..');
const sourceSpecPath = resolve(repositoryRoot, 'openapi/openapi.yaml');
const bundleSpecPath = resolve(repositoryRoot, 'openapi/dist/openapi.bundle.json');

const requireFromWeb = createRequire(resolve(repositoryRoot, 'web/package.json'));
const { bundle, createConfig } = requireFromWeb('@redocly/openapi-core');

const config = await createConfig({}, { extends: ['minimal'] });
const result = await bundle({
  config,
  ref: sourceSpecPath,
});

const blockingProblems = result.problems.filter((problem) => problem.severity === 'error');
if (blockingProblems.length > 0) {
  const message = blockingProblems.map((problem) => problem.message).join('\n');
  throw new Error(`bundle openapi spec failed:\n${message}`);
}

mkdirSync(dirname(bundleSpecPath), { recursive: true });
writeFileSync(bundleSpecPath, `${JSON.stringify(result.bundle.parsed, null, 2)}\n`);
