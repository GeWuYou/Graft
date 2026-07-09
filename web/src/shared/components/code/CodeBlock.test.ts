import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';

import CodeBlock from './CodeBlock.vue';

const renderHighlightedCodeBlock = vi.hoisted(() => vi.fn());

vi.mock('@/shared/code/shiki', () => ({
  renderHighlightedCodeBlock,
}));

describe('CodeBlock', () => {
  it('keeps the latest highlighted result when earlier rendering resolves later', async () => {
    let resolveFirst: ((value: string) => void) | undefined;
    let resolveSecond: ((value: string) => void) | undefined;
    renderHighlightedCodeBlock
      .mockImplementationOnce(
        () =>
          new Promise<string>((resolve) => {
            resolveFirst = resolve;
          }),
      )
      .mockImplementationOnce(
        () =>
          new Promise<string>((resolve) => {
            resolveSecond = resolve;
          }),
      );

    const wrapper = mount(CodeBlock, { props: { code: 'first', lang: 'shell' } });
    await wrapper.setProps({ code: 'second' });

    resolveSecond?.('<pre><code>second</code></pre>');
    await flushPromises();
    resolveFirst?.('<pre><code>first</code></pre>');
    await flushPromises();

    expect(wrapper.html()).toContain('second');
    expect(wrapper.html()).not.toContain('first');
  });
});
