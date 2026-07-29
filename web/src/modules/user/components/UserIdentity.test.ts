import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';

import UserIdentity from './UserIdentity.vue';

describe('UserIdentity', () => {
  it('falls back to the username when display contains only whitespace', () => {
    const wrapper = mount(UserIdentity, {
      props: {
        user: {
          display: '   ',
          username: 'alice',
        },
      },
    });

    expect(wrapper.get('.user-cell__display').text()).toBe('alice');
    expect(wrapper.get('.user-cell__avatar').text()).toBe('A');
  });

  it('keeps an emoji display initial intact', () => {
    const wrapper = mount(UserIdentity, {
      props: {
        user: {
          display: '😀 Alice',
          username: 'alice',
        },
      },
    });

    expect(wrapper.get('.user-cell__display').text()).toBe('😀 Alice');
    expect(wrapper.get('.user-cell__avatar').text()).toBe('😀');
  });
});
