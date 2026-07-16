import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { describe, expect, it } from 'vitest';

const sourceText = readFileSync(join(process.cwd(), 'src/modules/container/pages/resources/index.vue'), 'utf8');

describe('container resources page', () => {
  it('derives static resource snapshots from the module Query helper', () => {
    expect(sourceText).toContain('useDockerResourceQueries(active)');
    expect(sourceText).not.toContain('async function load()');
    expect(sourceText).not.toContain('onMounted');
    expect(sourceText).not.toContain('watch(active');
    expect(sourceText).not.toContain('ref<any[]>');
  });
});
