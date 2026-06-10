// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { spawnSync } from 'node:child_process';
import { cpSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

import { afterEach, describe, expect, it } from 'vitest';

const tempRoots: string[] = [];

function createTempWebRoot(source: string) {
  const repoRoot = join(tmpdir(), `graft-i18n-governance-${process.pid}-${tempRoots.length}`);
  const root = join(repoRoot, 'web');
  tempRoots.push(repoRoot);

  mkdirSync(join(root, 'scripts'), { recursive: true });
  mkdirSync(join(root, 'src/modules/demo/locales'), { recursive: true });
  mkdirSync(join(repoRoot, 'server/internal'), { recursive: true });
  mkdirSync(join(repoRoot, 'server/modules'), { recursive: true });
  cpSync(join(process.cwd(), 'scripts/check-i18n-governance.ts'), join(root, 'scripts/check-i18n-governance.ts'));
  writeFileSync(join(root, 'src/modules/demo/UnsafeTime.vue'), source);
  writeFileSync(join(root, 'src/modules/demo/locales/en-US.json'), '{}');
  writeFileSync(join(root, 'src/modules/demo/locales/zh-CN.json'), '{}');

  return root;
}

function writeLocaleCatalogs(root: string, messages: string) {
  writeFileSync(join(root, 'src/modules/demo/locales/en-US.json'), messages);
  writeFileSync(join(root, 'src/modules/demo/locales/zh-CN.json'), messages);
}

function writeServerModule(root: string, file: string, source: string) {
  const filePath = join(root, '..', 'server/modules/demo', file);
  mkdirSync(join(root, '..', 'server/modules/demo'), { recursive: true });
  writeFileSync(filePath, source);
}

async function runGovernanceScript(source: string) {
  const root = createTempWebRoot(source);
  const process = spawnSync('bun', ['run', 'scripts/check-i18n-governance.ts'], {
    cwd: root,
    encoding: 'utf8',
  });

  return {
    stdout: process.stdout,
    stderr: process.stderr,
    exitCode: process.status,
  };
}

async function runGovernanceScriptWithServerSource(source: string, serverSource: string) {
  const root = createTempWebRoot(source);
  writeServerModule(root, 'config.go', serverSource);
  const process = spawnSync('bun', ['run', 'scripts/check-i18n-governance.ts'], {
    cwd: root,
    encoding: 'utf8',
  });

  return {
    stdout: process.stdout,
    stderr: process.stderr,
    exitCode: process.status,
  };
}

async function runGovernanceScriptWithServerSourceAndLocales(
  source: string,
  serverSource: string,
  localeMessages: string,
) {
  const root = createTempWebRoot(source);
  writeLocaleCatalogs(root, localeMessages);
  writeServerModule(root, 'config.go', serverSource);
  const process = spawnSync('bun', ['run', 'scripts/check-i18n-governance.ts'], {
    cwd: root,
    encoding: 'utf8',
  });

  return {
    stdout: process.stdout,
    stderr: process.stderr,
    exitCode: process.status,
  };
}

afterEach(() => {
  for (const root of tempRoots.splice(0)) {
    rmSync(root, { force: true, recursive: true });
  }
});

describe('check-i18n-governance datetime formatting scan', () => {
  it('blocks visible datetime formatting that depends on the host locale', async () => {
    const result = await runGovernanceScript(`
<template>{{ label }}</template>
<script setup lang="ts">
const label = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date());
const fallback = new Date().toLocaleString();
</script>
`);

    expect(result.exitCode).toBe(1);
    expect(result.stdout).toContain('visible datetime formatting must pass the active locale instead of undefined');
    expect(result.stdout).toContain('toLocaleString must pass the active locale');
    expect(result.stderr).toBe('');
  });

  it('allows visible datetime formatting through an explicit locale-aware formatter', async () => {
    const result = await runGovernanceScript(`
<template>{{ label }}</template>
<script setup lang="ts">
import { formatLocaleDateTime } from '@/shared/observability';
const label = formatLocaleDateTime('2026-06-10T02:38:00Z', 'en-US');
</script>
`);

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('No hard-coded UI text or locale governance issues found.');
    expect(result.stderr).toBe('');
  });
});

describe('check-i18n-governance server system config scan', () => {
  it('requires web locale catalogs for backend system config dynamic display keys', async () => {
    const result = await runGovernanceScriptWithServerSource(
      `
<template><span /></template>
`,
      `
package demo

const demoEnabledKey = "demo.enabled"

func demoConfigTitleKey(key string) string {
  return "systemConfig.demo." + key + ".title"
}

func demoConfigDescriptionKey(key string) string {
  return "systemConfig.demo." + key + ".description"
}

func demoConfigDefinitions() []configregistry.Definition {
  return []configregistry.Definition{
    booleanDemoDefinition(demoEnabledKey, "Demo enabled", "Whether demo is enabled."),
  }
}

func booleanDemoDefinition(key string, title string, description string) configregistry.Definition {
  return configregistry.Definition{
    Key: key,
    DomainKey: "systemConfig.domains.demo",
    GroupKey: "systemConfig.groups.demo.general",
    GroupDescriptionKey: "systemConfig.groups.demo.general.description",
    TitleKey: demoConfigTitleKey(key),
    DescriptionKey: demoConfigDescriptionKey(key),
    Schema: []byte("{\\"type\\":\\"boolean\\",\\"x-i18n\\":{\\"titleKey\\":\\"systemConfig.demo.demo.enabled.title\\",\\"descriptionKey\\":\\"systemConfig.demo.demo.enabled.description\\"}}"),
  }
}
`,
    );

    expect(result.exitCode).toBe(1);
    expect(result.stdout).toContain('referenced locale key systemConfig.demo.demo.enabled.title is missing');
    expect(result.stdout).toContain('referenced locale key systemConfig.demo.demo.enabled.description is missing');
    expect(result.stdout).toContain('referenced locale key systemConfig.domains.demo is missing');
    expect(result.stdout).toContain('referenced locale key systemConfig.groups.demo.general is missing');
    expect(result.stderr).toBe('');
  });

  it('blocks server system config schema fallback copy without x-i18n keys', async () => {
    const result = await runGovernanceScriptWithServerSource(
      `
<template><span /></template>
`,
      `
package demo

const demoConfigSchema = \`{"type":"object","properties":{"maxQuickActions":{"type":"integer","title":"Maximum quick actions","description":"Maximum personalized entries shown on the dashboard home page."}}}\`
`,
    );

    expect(result.exitCode).toBe(1);
    expect(result.stdout).toContain(
      'system config schema schema.properties.maxQuickActions.title has visible fallback "Maximum quick actions" without x-i18n.titleKey',
    );
    expect(result.stdout).toContain(
      'system config schema schema.properties.maxQuickActions.description has visible fallback "Maximum personalized entries shown on the dashboard home page." without x-i18n.descriptionKey',
    );
    expect(result.stderr).toBe('');
  });

  it('allows server system config schema fallback copy when matching x-i18n keys exist', async () => {
    const result = await runGovernanceScriptWithServerSourceAndLocales(
      `
<template><span /></template>
`,
      `
package demo

const demoConfigSchema = \`{"type":"object","properties":{"maxQuickActions":{"type":"integer","title":"Maximum quick actions","description":"Maximum personalized entries shown on the dashboard home page.","x-i18n":{"titleKey":"systemConfig.demo.maxQuickActions.title","descriptionKey":"systemConfig.demo.maxQuickActions.description"}}}}\`
`,
      `{
  "systemConfig": {
    "demo": {
      "maxQuickActions": {
        "title": "Maximum quick actions",
        "description": "Maximum personalized entries shown on the dashboard home page."
      }
    }
  }
}`,
    );

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('No hard-coded UI text or locale governance issues found.');
    expect(result.stderr).toBe('');
  });
});
