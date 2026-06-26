import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';

import StreamViewportStateSurface from './StreamViewportStateSurface.vue';

describe('StreamViewportStateSurface', () => {
  it('renders console-style prop-driven copy for busy stream states', () => {
    const wrapper = mount(StreamViewportStateSurface, {
      props: {
        state: 'connecting',
        badgeLabel: 'Stream Handshake',
        busyLabel: 'Negotiating upstream transport',
        title: 'Attaching to live output',
        description: 'The viewport will lock to fresh events as soon as the stream opens.',
        hint: 'Keep this panel open while the upstream endpoint finishes negotiation.',
      },
    });

    expect(wrapper.classes()).toContain('stream-viewport-state-surface--connecting');
    expect(wrapper.text()).toContain('Stream Handshake');
    expect(wrapper.text()).toContain('Negotiating upstream transport');
    expect(wrapper.text()).toContain('Attaching to live output');
    expect(wrapper.text()).toContain('The viewport will lock to fresh events as soon as the stream opens.');
    expect(wrapper.text()).toContain('Keep this panel open while the upstream endpoint finishes negotiation.');
    expect(wrapper.find('.stream-viewport-state-surface__busy-indicator').exists()).toBe(true);
    expect(wrapper.find('.stream-viewport-state-surface__cursor').exists()).toBe(true);
    expect(wrapper.findAll('.stream-viewport-state-surface__faux-line')).toHaveLength(7);
  });

  it('keeps copy empty when optional props are omitted and still renders the console scaffold', () => {
    const wrapper = mount(StreamViewportStateSurface, {
      props: {
        state: 'empty',
        fauxLineCount: 5,
      },
    });

    expect(wrapper.classes()).toContain('stream-viewport-state-surface--empty');
    expect(wrapper.text()).toBe('');
    expect(wrapper.find('.stream-viewport-state-surface__busy-indicator').exists()).toBe(false);
    expect(wrapper.find('.stream-viewport-state-surface__cursor').exists()).toBe(false);
    expect(wrapper.findAll('.stream-viewport-state-surface__faux-line')).toHaveLength(5);
  });

  it('allows explicit cursor and busy overrides for paused snapshots', () => {
    const wrapper = mount(StreamViewportStateSurface, {
      props: {
        state: 'paused',
        title: 'Stream paused at tail',
        showBusy: false,
        showCursor: true,
      },
    });

    expect(wrapper.classes()).toContain('stream-viewport-state-surface--paused');
    expect(wrapper.attributes('aria-label')).toBe('Stream paused at tail');
    expect(wrapper.find('.stream-viewport-state-surface__busy-indicator').exists()).toBe(false);
    expect(wrapper.find('.stream-viewport-state-surface__cursor').exists()).toBe(true);
  });
});
