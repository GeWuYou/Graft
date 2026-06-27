import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

import { afterEach, describe, expect, it } from 'vitest';

import { runNativeDialogGovernanceAudit } from './check-native-dialog-governance';

const tempRoots: string[] = [];
const SCRATCH_PARENT = join(process.cwd(), '.tmp/native-dialog-governance-tests');

function createScratchRoot() {
  mkdirSync(SCRATCH_PARENT, { recursive: true });
  const root = mkdtempSync(join(SCRATCH_PARENT, 'repo-'));
  tempRoots.push(root);
  mkdirSync(join(root, 'src/modules/demo'), { recursive: true });
  return root;
}

function runAuditWithSource(file: string, source: string) {
  const root = createScratchRoot();
  const filePath = join(root, file);
  mkdirSync(join(filePath, '..'), { recursive: true });
  writeFileSync(filePath, source);

  return runNativeDialogGovernanceAudit({
    rootDir: root,
    srcDir: join(root, 'src'),
  });
}

afterEach(() => {
  for (const root of tempRoots.splice(0)) {
    rmSync(root, { force: true, recursive: true });
  }
  rmSync(SCRATCH_PARENT, { force: true, recursive: true });
});

describe('check-native-dialog-governance blacklist scan', () => {
  it('allows runtime files without browser-native dialogs', () => {
    const result = runAuditWithSource(
      'src/modules/demo/OkDialog.vue',
      `
<script setup lang="ts">
const visible = ref(false);
</script>
<template>
  <t-dialog v-model:visible="visible" />
</template>
`,
    );

    expect(result.debt).toHaveLength(0);
    expect(result.output).toContain('Native dialog governance: no browser-native dialogs found.');
  });

  it('blocks window.confirm usage in runtime source', () => {
    const result = runAuditWithSource(
      'src/modules/demo/UnsafeConfirm.ts',
      `
export function run() {
  return window.confirm('danger');
}
`,
    );

    expect(result.debt).toHaveLength(1);
    expect(result.debt[0]?.callee).toBe('confirm');
    expect(result.output).toContain('Blacklisted browser-native dialogs found:');
  });

  it('does not flag TDesign DialogPlugin methods as native dialogs', () => {
    const result = runAuditWithSource(
      'src/modules/demo/DialogPluginOk.ts',
      `
import { DialogPlugin } from 'tdesign-vue-next';

export function run() {
  return DialogPlugin.confirm({ header: 'ok' });
}
`,
    );

    expect(result.debt).toHaveLength(0);
  });

  it('blocks bare alert and globalThis.prompt usage', () => {
    const result = runAuditWithSource(
      'src/modules/demo/UnsafePrompt.vue',
      `
<script setup lang="ts">
alert('notice');
globalThis.prompt('name');
</script>
`,
    );

    expect(result.debt).toHaveLength(2);
    expect(result.output).toContain('alert');
    expect(result.output).toContain('prompt');
  });

  it('ignores test files and generated runtime artifacts', () => {
    const root = createScratchRoot();
    mkdirSync(join(root, 'src/contracts/openapi/generated'), { recursive: true });
    writeFileSync(
      join(root, 'src/modules/demo/UnsafeConfirm.test.ts'),
      `
export function run() {
  return confirm('test only');
}
`,
    );
    writeFileSync(
      join(root, 'src/contracts/openapi/generated/schema.ts'),
      `
export const x = prompt('generated');
`,
    );

    const result = runNativeDialogGovernanceAudit({
      rootDir: root,
      srcDir: join(root, 'src'),
    });

    expect(result.debt).toHaveLength(0);
  });
});
