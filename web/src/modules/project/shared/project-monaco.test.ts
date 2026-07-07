import { readFileSync } from 'node:fs';

import { describe, expect, it } from 'vitest';

import { toMonacoColor } from './project-monaco-color';
import { isProjectMonacoDebugEnabled } from './project-monaco-debug';
import { buildProjectMonacoWorker } from './project-monaco-worker';

describe('project-monaco color normalization', () => {
  it('converts srgb browser output into Monaco-safe hex', () => {
    expect(toMonacoColor('color(srgb 0.1379 0.197 0.2236 / 1)', '#1b2230')).toBe('#233239');
  });

  it('preserves alpha as rrggbbaa when the browser returns slash alpha rgb', () => {
    expect(toMonacoColor('rgb(95 155 255 / 0.28)', '#1b2230')).toBe('#5f9bff47');
  });

  it('falls back to the parsed fallback color when the computed value is invalid', () => {
    expect(toMonacoColor('definitely-not-a-color', '#1b2230')).toBe('#1b2230');
  });

  it('never returns raw css color syntax fragments', () => {
    const normalized = toMonacoColor('color(srgb 0.1379 0.197 0.2236 / 1)', '#1b2230');

    expect(normalized.startsWith('#')).toBe(true);
    expect(normalized.includes('color(')).toBe(false);
    expect(normalized.includes('#0.')).toBe(false);
  });
});

describe('project-monaco worker routing', () => {
  it('routes yaml models to the yaml worker factory', () => {
    const editorWorker = { kind: 'editor-worker' } as unknown as Worker;
    const yamlWorker = { kind: 'yaml-worker' } as unknown as Worker;

    const worker = buildProjectMonacoWorker('yaml', {
      createEditorWorker: () => editorWorker,
      createYamlWorker: () => yamlWorker,
    });

    expect(worker).toBe(yamlWorker);
  });

  it('routes non-yaml models to the editor worker factory', () => {
    const editorWorker = { kind: 'editor-worker' } as unknown as Worker;
    const yamlWorker = { kind: 'yaml-worker' } as unknown as Worker;

    const worker = buildProjectMonacoWorker('editorWorkerService', {
      createEditorWorker: () => editorWorker,
      createYamlWorker: () => yamlWorker,
    });

    expect(worker).toBe(editorWorker);
  });
});

describe('project-monaco language registration', () => {
  it('keeps yaml contribution import wired into the shared monaco setup', () => {
    const source = readFileSync('src/modules/project/shared/project-monaco.ts', 'utf8');

    expect(source).toContain('monaco-editor/esm/vs/basic-languages/yaml/yaml.contribution');
  });
});

describe('project-monaco debug toggle', () => {
  it('reads the explicit global debug flag before localStorage', () => {
    const previousValue = (globalThis as typeof globalThis & Record<string, unknown>).__GRAFT_MONACO_DEBUG__;

    (globalThis as typeof globalThis & Record<string, unknown>).__GRAFT_MONACO_DEBUG__ = true;

    expect(isProjectMonacoDebugEnabled()).toBe(true);

    if (typeof previousValue === 'undefined') {
      delete (globalThis as typeof globalThis & Record<string, unknown>).__GRAFT_MONACO_DEBUG__;
      return;
    }

    (globalThis as typeof globalThis & Record<string, unknown>).__GRAFT_MONACO_DEBUG__ = previousValue;
  });
});
