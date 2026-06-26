import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

import { afterEach, describe, expect, it } from 'vitest';

import { runScrollbarGovernanceAudit } from './check-scrollbar-governance';

const tempRoots: string[] = [];
const SCRATCH_PARENT = join(process.cwd(), '.tmp/scrollbar-governance-tests');

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

  return runScrollbarGovernanceAudit({
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

describe('check-scrollbar-governance blacklist scan', () => {
  it('allows the shared graft-scrollbar utility selector', () => {
    const result = runAuditWithSource(
      'src/style/scrollbar.less',
      `
.graft-scrollbar {
  scrollbar-color: var(--td-scrollbar-color) transparent;
  scrollbar-gutter: stable;
  scrollbar-width: thin;
}

.graft-scrollbar::-webkit-scrollbar {
  height: 10px;
  width: 10px;
}
`,
    );

    expect(result.debt).toHaveLength(0);
    expect(result.exceptions).toHaveLength(2);
    expect(result.output).toContain('Scrollbar governance: no blacklisted native scrollbar styles found.');
  });

  it('allows explicit allowlist entries such as SafeMarkdown', () => {
    const result = runAuditWithSource(
      'src/shared/components/markdown/SafeMarkdown.vue',
      `
<template><div class="markdown-viewer" /></template>
<style scoped lang="less">
.markdown-viewer :deep(pre) {
  scrollbar-color: var(--td-scrollbar-color) transparent;
  scrollbar-width: thin;
}
</style>
`,
    );

    expect(result.debt).toHaveLength(0);
    expect(result.exceptions).toHaveLength(1);
    expect(result.output).toContain('Allowed exceptions:');
  });

  it('blocks page-local native scrollbar rules', () => {
    const result = runAuditWithSource(
      'src/modules/demo/UnsafePanel.vue',
      `
<template><div class="unsafe-panel" /></template>
<style scoped lang="less">
.unsafe-panel {
  overflow: auto;
  scrollbar-width: thin;
}

.unsafe-panel::-webkit-scrollbar {
  width: 8px;
}
</style>
`,
    );

    expect(result.debt).toHaveLength(2);
    expect(result.output).toContain('Blacklisted native scrollbar styling found:');
    expect(result.output).toContain(
      'Only the shared `graft-scrollbar` utility and explicit allowlist entries may define native scrollbar styling.',
    );
  });

  it('blocks page-local graft-scrollbar pseudo-element rules outside the shared utility', () => {
    const result = runAuditWithSource(
      'src/modules/demo/LooksSharedButIsLocal.vue',
      `
<template><div class="graft-scrollbar local-panel" /></template>
<style scoped lang="less">
.local-panel.graft-scrollbar::-webkit-scrollbar {
  width: 8px;
}
</style>
`,
    );

    expect(result.debt).toHaveLength(1);
    expect(result.debt[0]?.selector).toContain('.graft-scrollbar::-webkit-scrollbar');
  });
});
