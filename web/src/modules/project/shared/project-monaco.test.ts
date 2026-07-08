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
  it('routes json models to the json worker factory', () => {
    const editorWorker = { kind: 'editor-worker' } as unknown as Worker;
    const jsonWorker = { kind: 'json-worker' } as unknown as Worker;
    const yamlWorker = { kind: 'yaml-worker' } as unknown as Worker;

    const worker = buildProjectMonacoWorker('monaco-editor/esm/vs/language/json/json.worker', 'json', {
      createEditorWorker: () => editorWorker,
      createJsonWorker: () => jsonWorker,
      createYamlWorker: () => yamlWorker,
    });

    expect(worker).toBe(jsonWorker);
  });

  it('routes yaml models to the yaml worker factory', () => {
    const editorWorker = { kind: 'editor-worker' } as unknown as Worker;
    const jsonWorker = { kind: 'json-worker' } as unknown as Worker;
    const yamlWorker = { kind: 'yaml-worker' } as unknown as Worker;

    const worker = buildProjectMonacoWorker('monaco-yaml/yaml.worker', 'yaml', {
      createEditorWorker: () => editorWorker,
      createJsonWorker: () => jsonWorker,
      createYamlWorker: () => yamlWorker,
    });

    expect(worker).toBe(yamlWorker);
  });

  it('routes non-yaml models to the editor worker factory', () => {
    const editorWorker = { kind: 'editor-worker' } as unknown as Worker;
    const jsonWorker = { kind: 'json-worker' } as unknown as Worker;
    const yamlWorker = { kind: 'yaml-worker' } as unknown as Worker;

    const worker = buildProjectMonacoWorker('monaco-editor/esm/vs/editor/editor.worker', 'editorWorkerService', {
      createEditorWorker: () => editorWorker,
      createJsonWorker: () => jsonWorker,
      createYamlWorker: () => yamlWorker,
    });

    expect(worker).toBe(editorWorker);
  });

  it('routes yaml workers by moduleId even when the label is missing', () => {
    const editorWorker = { kind: 'editor-worker' } as unknown as Worker;
    const jsonWorker = { kind: 'json-worker' } as unknown as Worker;
    const yamlWorker = { kind: 'yaml-worker' } as unknown as Worker;

    const worker = buildProjectMonacoWorker('monaco-yaml/yaml.worker', '', {
      createEditorWorker: () => editorWorker,
      createJsonWorker: () => jsonWorker,
      createYamlWorker: () => yamlWorker,
    });

    expect(worker).toBe(yamlWorker);
  });
});

describe('project-monaco debug toggle', () => {
  it('enables debug logging from the explicit env flag in production', () => {
    const previousValue = import.meta.env.VITE_PROJECT_MONACO_DEBUG;

    try {
      import.meta.env.VITE_PROJECT_MONACO_DEBUG = 'true';

      expect(isProjectMonacoDebugEnabled()).toBe(true);
    } finally {
      import.meta.env.VITE_PROJECT_MONACO_DEBUG = previousValue;
    }
  });

  it('reads the explicit global debug flag before localStorage', () => {
    const previousValue = (globalThis as typeof globalThis & Record<string, unknown>).__GRAFT_MONACO_DEBUG__;

    try {
      (globalThis as typeof globalThis & Record<string, unknown>).__GRAFT_MONACO_DEBUG__ = true;

      expect(isProjectMonacoDebugEnabled()).toBe(true);
    } finally {
      if (typeof previousValue === 'undefined') {
        delete (globalThis as typeof globalThis & Record<string, unknown>).__GRAFT_MONACO_DEBUG__;
      } else {
        (globalThis as typeof globalThis & Record<string, unknown>).__GRAFT_MONACO_DEBUG__ = previousValue;
      }
    }
  });
});
