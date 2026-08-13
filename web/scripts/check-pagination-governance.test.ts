import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

import { afterEach, describe, expect, it } from 'vitest';

import { runPaginationGovernanceAudit } from './check-pagination-governance';

const tempRoots: string[] = [];
const SCRATCH_PARENT = join(process.cwd(), '.tmp/pagination-governance-tests');

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

  return runPaginationGovernanceAudit({ rootDir: root, srcDir: join(root, 'src') });
}

afterEach(() => {
  for (const root of tempRoots.splice(0)) {
    rmSync(root, { force: true, recursive: true });
  }
  rmSync(SCRATCH_PARENT, { force: true, recursive: true });
});

describe('check-pagination-governance management-list scan', () => {
  it('allows a management table bound directly to a server page', () => {
    const result = runAuditWithSource(
      'src/modules/demo/pages/list/index.vue',
      `<management-paged-table :rows="items" :total="total" />`,
    );

    expect(result.debt).toHaveLength(0);
  });

  it('blocks a filtered local list bound to a management table', () => {
    const result = runAuditWithSource(
      'src/modules/demo/pages/list/index.vue',
      `<template><management-paged-table :rows="filteredUsers" /></template>
<script setup lang="ts">
const filteredUsers = computed(() => users.value.filter((user) => user.name.includes(keyword.value)));
</script>`,
    );

    expect(result.debt).toEqual([expect.objectContaining({ method: 'filter', variable: 'filteredUsers' })]);
  });

  it('blocks a paged local list bound to a management table', () => {
    const result = runAuditWithSource(
      'src/modules/demo/pages/list/index.vue',
      `<template><advanced-query-paged-table :rows="pagedUsers" /></template>
<script setup lang="ts">
const pagedUsers = computed(() => users.value.slice(0, pageSize.value));
</script>`,
    );

    expect(result.debt).toEqual([expect.objectContaining({ method: 'slice', variable: 'pagedUsers' })]);
  });

  it('blocks direct filter and slice table bindings', () => {
    const result = runAuditWithSource(
      'src/modules/demo/pages/list/index.vue',
      `<management-paged-table :rows="items.filter(matchesQuery)" />
<advanced-query-paged-table :rows="items.slice(0, pageSize)" />`,
    );

    expect(result.debt).toEqual([
      expect.objectContaining({ method: 'filter', variable: '<table rows binding>' }),
      expect.objectContaining({ method: 'slice', variable: '<table rows binding>' }),
    ]);
  });

  it('does not flag derived values that are not bound to a management table', () => {
    const result = runAuditWithSource(
      'src/modules/demo/pages/list/index.vue',
      `<template><management-paged-table :rows="items" /></template>
<script setup lang="ts">
const filteredSummary = computed(() => items.value.filter((item) => item.enabled));
</script>`,
    );

    expect(result.debt).toHaveLength(0);
  });

  it('allows import-preview, detail-preview, and log-window local lists', () => {
    const root = createScratchRoot();
    const importFile = join(root, 'src/modules/demo/pages/import/InspectResources.vue');
    const detailFile = join(root, 'src/modules/demo/pages/detail/index.vue');
    const logFile = join(root, 'src/modules/app-log/components/LogWindow.vue');
    mkdirSync(join(importFile, '..'), { recursive: true });
    mkdirSync(join(detailFile, '..'), { recursive: true });
    mkdirSync(join(logFile, '..'), { recursive: true });
    const source = `<management-paged-table :rows="pagedRows" />
<script setup lang="ts">
const pagedRows = computed(() => rows.value.slice(0, 20));
</script>`;
    writeFileSync(importFile, source);
    writeFileSync(detailFile, source);
    writeFileSync(logFile, source);

    const result = runPaginationGovernanceAudit({ rootDir: root, srcDir: join(root, 'src') });
    expect(result.debt).toHaveLength(0);
  });

  it('ignores test and generated source artifacts', () => {
    const root = createScratchRoot();
    const testFile = join(root, 'src/modules/demo/pages/list/index.test.ts');
    const generatedFile = join(root, 'src/contracts/openapi/generated/schema.ts');
    mkdirSync(join(testFile, '..'), { recursive: true });
    mkdirSync(join(generatedFile, '..'), { recursive: true });
    const source = `<management-paged-table :rows="pagedRows" />
const pagedRows = computed(() => rows.value.slice(0, 20));`;
    writeFileSync(testFile, source);
    writeFileSync(generatedFile, source);

    const result = runPaginationGovernanceAudit({ rootDir: root, srcDir: join(root, 'src') });
    expect(result.debt).toHaveLength(0);
  });
});
