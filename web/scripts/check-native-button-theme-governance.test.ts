import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

import { afterEach, describe, expect, it } from 'vitest';

import { runNativeButtonThemeGovernanceAudit } from './check-native-button-theme-governance';

const tempRoots: string[] = [];
const SCRATCH_PARENT = join(process.cwd(), '.tmp/native-button-theme-governance-tests');

function runAuditWithSources(sources: Record<string, string>) {
  mkdirSync(SCRATCH_PARENT, { recursive: true });
  const root = mkdtempSync(join(SCRATCH_PARENT, 'repo-'));
  tempRoots.push(root);
  mkdirSync(join(root, 'src/style'), { recursive: true });

  for (const [file, source] of Object.entries(sources)) {
    const filePath = join(root, file);
    mkdirSync(join(filePath, '..'), { recursive: true });
    writeFileSync(filePath, source);
  }

  return runNativeButtonThemeGovernanceAudit({ rootDir: root, srcDir: join(root, 'src') });
}

afterEach(() => {
  for (const root of tempRoots.splice(0)) {
    rmSync(root, { force: true, recursive: true });
  }
  rmSync(SCRATCH_PARENT, { force: true, recursive: true });
});

describe('check-native-button-theme-governance', () => {
  it('accepts the global inheritance baseline and tokenized native button colors', () => {
    const result = runAuditWithSources({
      'src/style/reset.less': 'button { color: inherit; font: inherit; }',
      'src/modules/demo/page.vue': `
<template><button class="demo-action">Action</button></template>
<style lang="less">
.demo-action { color: var(--td-text-color-primary); }
</style>
`,
    });

    expect(result.findings).toHaveLength(0);
  });

  it('blocks missing inheritance and hard-coded native button colors', () => {
    const result = runAuditWithSources({
      'src/style/reset.less': 'button { color: inherit; }',
      'src/modules/demo/page.vue': `
<template><button class="demo-action">Action</button></template>
<style lang="less">
.demo-action button { color: #000; }
</style>
`,
    });

    expect(result.findings).toHaveLength(2);
    expect(result.output).toContain(
      'global native button theme baseline must declare color: inherit and font: inherit',
    );
    expect(result.output).toContain('found #000');
  });

  it('does not flag TDesign button styles or test/generated files', () => {
    const result = runAuditWithSources({
      'src/style/reset.less': 'button { color: inherit; font: inherit; }',
      'src/modules/demo/page.vue': `
<style lang="less">
:deep(.t-button) { color: #000; }
</style>
`,
      'src/modules/demo/page.test.vue': '<style>button { color: #000; }</style>',
      'src/contracts/openapi/generated/schema.ts': 'button { color: #000; }',
    });

    expect(result.findings).toHaveLength(0);
  });
});
