import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

import { describe, expect, it } from 'vitest';

const PAGE_SOURCE = readFileSync(join(dirname(fileURLToPath(import.meta.url)), 'index.vue'), 'utf8');

describe('security overview theme contracts', () => {
  it('keeps recent security event text on theme-aware primary and secondary tokens', () => {
    expect(PAGE_SOURCE).toMatch(/\.security-overview__event\s*\{[\s\S]*?color:\s*var\(--td-text-color-primary\)/);
    expect(PAGE_SOURCE).toMatch(
      /\.security-overview__event-copy\s+small,[\s\S]*?color:\s*var\(--td-text-color-secondary\)/,
    );
    expect(PAGE_SOURCE).not.toContain("t('security.overview.events.unknownResource')");
    expect(PAGE_SOURCE).toContain('function eventLabel(action: string)');
    expect(PAGE_SOURCE).not.toMatch(/\.security-overview__event[^{}]*\{[^{}]*color:\s*(?:black|#000)/i);
  });
});
