import { describe, expect, it } from 'vitest';

import sourceText from './TagManagerDrawer.vue?raw';

describe('TagManagerDrawer', () => {
  it('keeps tag lifecycle operations separate from Image deletion', () => {
    expect(sourceText).toContain('untagDockerImage(image.value.id, { reference: selectedReference.value })');
    expect(sourceText).toContain("t('container.images.untag.lastTagWarning')");
    expect(sourceText).not.toContain('docker run');
    expect(sourceText).not.toContain('force');
  });

  it('offers only copyable image identity, digest, reference, and pull command values', () => {
    expect(sourceText).toContain('image.repository_digests?.length');
    expect(sourceText).toContain('`docker pull ${reference}`');
    expect(sourceText).toContain('copy(image.id)');
    expect(sourceText).toContain('copy(reference)');
  });

  it('centers a standalone loading indicator before the image record arrives', () => {
    expect(sourceText).toContain('class="tag-manager-loading-host"');
    expect(sourceText).toContain('.tag-manager-loading-host {\n  min-height: 240px;');
    expect(sourceText).toContain('class="tag-manager-loading-host__indicator"');
    expect(sourceText).toContain('place-items: center;');
    expect(sourceText).toContain('tag-manager-loading-spin 1s linear infinite');
  });
});
