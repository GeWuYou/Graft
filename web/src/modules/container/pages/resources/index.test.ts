import { describe, expect, it } from 'vitest';

import sourceText from './index.vue?raw';

describe('container resources page', () => {
  it('derives static resource snapshots from the module Query helper', () => {
    expect(sourceText).toContain('useDockerResourceQueries(active)');
    expect(sourceText).not.toContain('async function load()');
    expect(sourceText).not.toContain('onMounted');
    expect(sourceText).not.toContain('watch(active');
    expect(sourceText).not.toContain('ref<any[]>');
    expect(sourceText).not.toContain('value="volumes"');
    expect(sourceText).not.toContain('getDockerVolumes');
  });
});
