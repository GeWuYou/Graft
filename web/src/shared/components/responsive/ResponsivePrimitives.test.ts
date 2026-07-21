import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';

import { resolveResponsiveDialogPolicy } from './dialog-policy';
import ResponsiveContent from './ResponsiveContent.vue';
import ResponsiveDialog from './ResponsiveDialog.vue';
import ResponsiveEmpty from './ResponsiveEmpty.vue';
import ResponsiveHeader from './ResponsiveHeader.vue';
import ResponsivePage from './ResponsivePage.vue';
import ResponsiveToolbar from './ResponsiveToolbar.vue';

describe('responsive primitives', () => {
  it('keeps page, content and header composition semantic while preserving slots', () => {
    const page = mount(ResponsivePage, { props: { layout: 'grid' }, slots: { default: '<p>content</p>' } });
    const content = mount(ResponsiveContent, { props: { layout: 'split' }, slots: { default: '<p>detail</p>' } });
    const header = mount(ResponsiveHeader, {
      slots: {
        actions: '<button>create</button>',
        description: '<p>summary</p>',
        title: '<h1>Projects</h1>',
      },
    });

    expect(page.classes()).toContain('responsive-page--grid');
    expect(page.text()).toContain('content');
    expect(content.classes()).toContain('responsive-content--split');
    expect(header.find('.responsive-header__actions').text()).toContain('create');
    expect(header.find('.responsive-header__description').text()).toContain('summary');
  });

  it('provides named toolbar and empty-state slots without business props', () => {
    const toolbar = mount(ResponsiveToolbar, {
      slots: {
        filters: '<input aria-label="filter">',
        overflow: '<button>more</button>',
        primary: '<button>create</button>',
      },
    });
    const empty = mount(ResponsiveEmpty, {
      props: { tone: 'error' },
      slots: { actions: '<button>retry</button>', description: '<p>Try again</p>', title: '<h2>Unavailable</h2>' },
    });

    expect(toolbar.find('.responsive-toolbar__filters input').exists()).toBe(true);
    expect(toolbar.find('.responsive-toolbar__primary').text()).toContain('create');
    expect(toolbar.find('.responsive-toolbar__overflow').text()).toContain('more');
    expect(empty.classes()).toContain('responsive-empty--error');
    expect(empty.find('.responsive-empty__actions').text()).toContain('retry');
  });

  it('resolves dialog surfaces from semantic purpose and size without pixel props', () => {
    expect(resolveResponsiveDialogPolicy(375, 'confirm', 'compact')).toMatchObject({
      interaction: 'interactive',
      surface: 'sheet',
    });
    expect(resolveResponsiveDialogPolicy(375, 'form', 'large')).toMatchObject({ surface: 'fullscreen' });
    expect(resolveResponsiveDialogPolicy(768, 'workspace', 'large')).toMatchObject({
      interaction: 'readonly',
      surface: 'drawer',
    });
    expect(resolveResponsiveDialogPolicy(992, 'detail', 'medium')).toMatchObject({ surface: 'dialog' });

    const wrapper = mount(ResponsiveDialog, {
      props: { purpose: 'form', size: 'large' },
      slots: { default: '<p>form fields</p>', footer: '<button>save</button>' },
    });

    expect(wrapper.attributes('data-responsive-surface')).toBe('fullscreen');
    expect(wrapper.text()).toContain('form fields');
    expect(wrapper.find('.responsive-dialog__footer').text()).toContain('save');
    expect(Object.keys(wrapper.props())).toEqual(expect.arrayContaining(['purpose', 'size']));
    expect(Object.keys(wrapper.props())).not.toContain('width');
  });
});
