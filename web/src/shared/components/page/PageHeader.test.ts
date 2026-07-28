import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';

import PageHeader from './PageHeader.vue';

const PAGE_HEADER_SOURCE = readFileSync(resolve(process.cwd(), 'src/shared/components/page/PageHeader.vue'), 'utf8');

describe('PageHeader', () => {
  it('does not render breadcrumb markup', () => {
    const wrapper = mount(PageHeader, {
      props: {
        titleFallback: 'Server status',
        descriptionFallback: 'Health and runtime overview',
      },
    });

    expect(wrapper.find('.page-header__title').text()).toBe('Server status');
    expect(wrapper.find('.page-header__description').text()).toBe('Health and runtime overview');
    expect(wrapper.find('.page-header__breadcrumb').exists()).toBe(false);
    expect(wrapper.find('.t-breadcrumb').exists()).toBe(false);
  });

  it('keeps action slots in the header side region', () => {
    const wrapper = mount(PageHeader, {
      props: { titleFallback: 'Server status' },
      slots: {
        actions: '<button type="button">Refresh</button>',
        extra: '<span>Updated just now</span>',
      },
    });

    expect(wrapper.get('.page-header__side').text()).toContain('Refresh');
    expect(wrapper.get('.page-header__actions button').text()).toBe('Refresh');
    expect(wrapper.get('.page-header__extra').text()).toBe('Updated just now');
  });

  it('provides a compact-only action slot for shared responsive headers', () => {
    const wrapper = mount(PageHeader, {
      props: { titleFallback: 'Images' },
      slots: {
        actions: '<button type="button">Clean</button>',
        compactActions: '<button type="button">More</button>',
      },
    });

    expect(wrapper.get('.page-header__compact-actions button').text()).toBe('More');
    expect(PAGE_HEADER_SOURCE).toContain('$slots.compactActions');
  });

  it('uses its own container width to stack title and actions', () => {
    expect(PAGE_HEADER_SOURCE).toContain('container-type: inline-size');
    expect(PAGE_HEADER_SOURCE).toContain('@container (width < @screen-sm)');
    expect(PAGE_HEADER_SOURCE).not.toContain('@media (width <= 768px)');
  });

  it('supports compact icon actions that remain alongside the title', () => {
    const wrapper = mount(PageHeader, {
      props: { actionLayout: 'inline', titleFallback: 'Runtime Targets' },
      slots: { actions: '<button type="button">Discover</button>' },
    });

    expect(wrapper.classes()).toContain('page-header--inline-actions');
  });
});
