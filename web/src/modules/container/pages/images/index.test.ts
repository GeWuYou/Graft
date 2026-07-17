import { describe, expect, it } from 'vitest';

import sourceText from './index.vue?raw';

describe('docker image list page', () => {
  it('keeps image actions and pull logs inside the container module page', () => {
    expect(sourceText).toContain('useDockerImageQuery');
    expect(sourceText).toContain('pullDockerImage');
    expect(sourceText).toContain('LogBatchBuffer');
    expect(sourceText).toContain('LogRingBuffer');
    expect(sourceText).toContain('<log-viewer');
  });
});
