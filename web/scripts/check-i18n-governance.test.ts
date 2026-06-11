// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { spawnSync } from 'node:child_process';
import { cpSync, existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

import { afterEach, describe, expect, it } from 'vitest';

const tempRoots: string[] = [];
const FIXTURE_DIR = join(process.cwd(), 'scripts/i18n-governance/fixtures');
const SCRATCH_PARENT = join(process.cwd(), '.tmp/i18n-governance-tests');

function createScratchRepoRoot(prefix: string) {
  mkdirSync(SCRATCH_PARENT, { recursive: true });
  const repoRoot = mkdtempSync(join(SCRATCH_PARENT, prefix));
  tempRoots.push(repoRoot);

  return repoRoot;
}

function createTempWebRoot(source: string) {
  const repoRoot = createScratchRepoRoot('repo-');
  const root = join(repoRoot, 'web');

  mkdirSync(join(root, 'scripts'), { recursive: true });
  mkdirSync(join(root, 'src/modules/demo/locales'), { recursive: true });
  mkdirSync(join(repoRoot, 'server/internal'), { recursive: true });
  mkdirSync(join(repoRoot, 'server/modules'), { recursive: true });
  cpSync(join(process.cwd(), 'scripts/check-i18n-governance.ts'), join(root, 'scripts/check-i18n-governance.ts'));
  cpSync(join(process.cwd(), 'scripts/i18n-governance'), join(root, 'scripts/i18n-governance'), { recursive: true });
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
  const result = spawnSync('bun', ['run', 'scripts/check-i18n-governance.ts'], {
    cwd: root,
    encoding: 'utf8',
  });

  return {
    stdout: result.stdout,
    stderr: result.stderr,
    exitCode: result.status,
  };
}

async function runGovernanceScriptWithFixture(fixtureName: string, env: Record<string, string> = {}) {
  const repoRoot = createScratchRepoRoot('fixture-');
  const root = join(repoRoot, 'web');

  mkdirSync(join(root, 'scripts'), { recursive: true });
  mkdirSync(join(repoRoot, 'server/internal'), { recursive: true });
  mkdirSync(join(repoRoot, 'server/modules'), { recursive: true });
  cpSync(join(process.cwd(), 'scripts/check-i18n-governance.ts'), join(root, 'scripts/check-i18n-governance.ts'));
  cpSync(join(process.cwd(), 'scripts/i18n-governance'), join(root, 'scripts/i18n-governance'), { recursive: true });
  const fixtureRoot = join(FIXTURE_DIR, fixtureName);
  if (existsSync(join(fixtureRoot, 'src'))) {
    cpSync(join(fixtureRoot, 'src'), join(root, 'src'), { recursive: true });
  }
  if (existsSync(join(fixtureRoot, 'server'))) {
    cpSync(join(fixtureRoot, 'server'), join(repoRoot, 'server'), { recursive: true });
  }

  const result = spawnSync('bun', ['run', 'scripts/check-i18n-governance.ts'], {
    cwd: root,
    encoding: 'utf8',
    env: { ...process.env, ...env },
  });

  return {
    stdout: result.stdout,
    stderr: result.stderr,
    exitCode: result.status,
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
  rmSync(SCRATCH_PARENT, { force: true, recursive: true });
});

describe('check-i18n-governance datetime formatting scan', () => {
  it('blocks visible datetime formatting that depends on the host locale', async () => {
    const result = await runGovernanceScript(`
<template>{{ label }}</template>
<script setup lang="ts">
const label = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date());
const fallback = new Date().toLocaleString();
const voidLocale = Intl.DateTimeFormat(void 0, { timeStyle: 'short' }).format(new Date());
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
const count = 1234;
const numberLabel = count.toLocaleString();
</script>
`);

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('No hard-coded UI text or locale governance issues found.');
    expect(result.stderr).toBe('');
  });

  it('allows hidden text inside nested aria-hidden template ancestors', async () => {
    const result = await runGovernanceScript(`
<template>
  <button>
    <span aria-hidden="true">
      <span>Create report</span>
    </span>
  </button>
</template>
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
    expect(result.stdout).toContain('no-system-config-schema-fallback');
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

describe('check-i18n-governance fixture rules', () => {
  const invalidFixtures = [
    {
      fixture: 'invalid-hardcoded-template-text',
      expectation: 'blocks raw visible template text',
      expectedSnippets: ['no-hardcoded-template-text', 'Create report'],
    },
    {
      fixture: 'invalid-fallback-label-cjk',
      expectation: 'blocks fallbackLabel with Chinese literal copy',
      expectedSnippets: ['fallbackLabel', '编辑'],
    },
    {
      fixture: 'invalid-fallback-label-conditional-cjk',
      expectation: 'blocks conditional fallbackLabel with Chinese literal copy',
      expectedSnippets: ['fallbackLabel', '启用用户', '禁用用户'],
    },
    {
      fixture: 'invalid-bound-ui-prop-expression',
      expectation: 'blocks bound Vue UI props with conditional and template literal fallback copy',
      expectedSnippets: ['fallback-label', '启用用户', '禁用用户', '还有 ${count} 条任务'],
    },
    {
      fixture: 'invalid-more-label-fallback-cjk',
      expectation: 'blocks more-label-fallback with Chinese literal copy',
      expectedSnippets: ['more-label-fallback', '更多'],
    },
    {
      fixture: 'invalid-semantic-title-locale-object',
      expectation: 'blocks semanticTitle locale literal objects',
      expectedSnippets: ['semanticTitle', 'zh-CN', 'en-US'],
    },
    {
      fixture: 'invalid-cron-english-fallback',
      expectation: 'blocks cron-like English fallback copy',
      expectedSnippets: ['Advanced Cron expression'],
    },
    {
      fixture: 'invalid-template-literal-cjk',
      expectation: 'blocks template literals containing Chinese UI copy',
      expectedSnippets: ['每天', '执行'],
    },
    {
      fixture: 'invalid-fallback-message-english',
      expectation: 'blocks English fallbackMessage string fallback',
      expectedSnippets: ['fallbackMessage', 'Request failed'],
    },
    {
      fixture: 'invalid-raw-notification-technical-display',
      expectation: 'blocks raw notification technical values and fallback copy',
      expectedSnippets: [
        'no-raw-notification-technical-display',
        'task_succeeded',
        'scheduled_task_run',
        'USER',
        'Scheduled task succeeded',
      ],
    },
    {
      fixture: 'invalid-notification-required-keyset',
      expectation: 'blocks missing Notification Center presenter key catalogs',
      expectedSnippets: [
        'notification-required-keyset',
        'missing required notification key notification.title.scheduler.runSucceeded',
      ],
    },
  ];

  it.each(invalidFixtures)('$fixture: $expectation', async ({ expectedSnippets, fixture }) => {
    const result = await runGovernanceScriptWithFixture(fixture);

    expect(result.exitCode).toBe(1);
    for (const snippet of expectedSnippets) {
      expect(result.stdout).toContain(snippet);
    }
    expect(result.stderr).toBe('');
  });

  it('valid-i18n-keyed-copy: allows keyed UI copy without hardcoded fallback text', async () => {
    const result = await runGovernanceScriptWithFixture('valid-i18n-keyed-copy');

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('No hard-coded UI text or locale governance issues found.');
    expect(result.stderr).toBe('');
  });

  it('valid-notification-resolver-display: allows notification presenter usage', async () => {
    const result = await runGovernanceScriptWithFixture('valid-notification-resolver-display');

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('No hard-coded UI text or locale governance issues found.');
    expect(result.stderr).toBe('');
  });

  it('warns on server fallback-only key-first copy by default', async () => {
    const result = await runGovernanceScriptWithFixture('fallback-only-key-first-warning');

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('[warning]');
    expect(result.stdout).toContain('no-fallback-only-key-first');
    expect(result.stdout).toContain('Dashboard title');
    expect(result.stderr).toBe('');
  });

  it('blocks server fallback-only key-first copy in strict mode', async () => {
    const result = await runGovernanceScriptWithFixture('fallback-only-key-first-warning', {
      STRICT_I18N_KEY_FIRST: 'true',
    });

    expect(result.exitCode).toBe(1);
    expect(result.stdout).toContain('[error]');
    expect(result.stdout).toContain('no-fallback-only-key-first');
    expect(result.stdout).toContain('Dashboard title');
    expect(result.stderr).toBe('');
  });

  it('allows lowerCamel Go key-first fields next to fallback copy in strict mode', async () => {
    const result = await runGovernanceScriptWithServerSourceAndLocales(
      '',
      `
package demo

type auditLabels struct {
  messageKey string
  message string
}

var labels = auditLabels{
  messageKey: "demo.audit.saved",
  message: "record saved",
}

type configGroup struct {
  descriptionKey string
  description string
}

var group = configGroup{
  descriptionKey: "systemConfig.demo.description",
  description: "Controls demo behavior.",
}
`,
      JSON.stringify({
        demo: { audit: { saved: 'Record saved' } },
        systemConfig: { demo: { description: 'Controls demo behavior.' } },
      }),
    );

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('No hard-coded UI text or locale governance issues found.');
    expect(result.stderr).toBe('');
  });
});

describe('check-i18n-governance split legacy rules', () => {
  const splitRuleFixtures = [
    {
      invalid: 'invalid-missing-locale-key',
      ruleId: 'no-missing-locale-key',
      snippet: 'referenced locale key demo.missing.title is missing',
      valid: 'valid-missing-locale-key',
    },
    {
      invalid: 'invalid-locale-catalog-drift',
      ruleId: 'no-locale-catalog-drift',
      snippet: 'split locale ownership for demo.shared.title',
      valid: 'valid-locale-catalog-drift',
    },
    {
      invalid: 'invalid-unused-locale-key',
      ruleId: 'no-unused-locale-key',
      snippet: 'unused locale key demo.unused.title',
      valid: 'valid-unused-locale-key',
    },
    {
      invalid: 'invalid-duplicate-locale-key',
      ruleId: 'no-duplicate-locale-key',
      snippet: 'duplicate locale key demo.duplicate.title',
      valid: 'valid-duplicate-locale-key',
    },
    {
      invalid: 'invalid-unsafe-datetime-locale',
      ruleId: 'no-unsafe-datetime-locale',
      snippet: 'visible datetime formatting must pass the active locale instead of undefined',
      valid: 'valid-unsafe-datetime-locale',
    },
    {
      invalid: 'invalid-unsafe-locale-value',
      ruleId: 'no-unsafe-locale-value',
      snippet: 'locale key demo.self.title resolves to itself',
      valid: 'valid-unsafe-locale-value',
    },
    {
      invalid: 'invalid-hardcoded-template-text',
      ruleId: 'no-hardcoded-template-text',
      snippet: 'Create report',
      valid: 'valid-hardcoded-template-text',
    },
  ];

  it.each(splitRuleFixtures)('$invalid: reports $ruleId', async ({ invalid, ruleId, snippet }) => {
    const result = await runGovernanceScriptWithFixture(invalid);

    expect(result.exitCode).toBe(1);
    expect(result.stdout).toContain(ruleId);
    expect(result.stdout).toContain(snippet);
    expect(result.stderr).toBe('');
  });

  it.each(splitRuleFixtures)('$valid: passes $ruleId valid fixture', async ({ valid }) => {
    const result = await runGovernanceScriptWithFixture(valid);

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('No hard-coded UI text or locale governance issues found.');
    expect(result.stderr).toBe('');
  });
});
