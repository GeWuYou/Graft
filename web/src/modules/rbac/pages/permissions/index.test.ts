import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import PermissionPage from './index.vue';

const rbacApiMocks = vi.hoisted(() => ({
  getPermissions: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  warning: vi.fn(),
}));

vi.mock('../../api/rbac', () => ({
  getPermissions: rbacApiMocks.getPermissions,
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
    locale: {
      value: 'en-US',
    },
  }),
}));

vi.mock('tdesign-vue-next', () => ({
  MessagePlugin: {
    error: messageMocks.error,
    warning: messageMocks.warning,
  },
}));

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: {
    description: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () => h('div', [props.description, slots.default?.()]);
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_, { emit, slots, attrs }) {
    return () => h('button', { ...attrs, onClick: (event: MouseEvent) => emit('click', event) }, slots.default?.());
  },
});

const inputStub = defineComponent({
  name: 'TInputStub',
  props: {
    modelValue: {
      type: String,
      default: '',
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, attrs }) {
    return () =>
      h('input', {
        ...attrs,
        value: props.modelValue,
        onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
      });
  },
});

const selectStub = defineComponent({
  name: 'TSelectStub',
  props: {
    modelValue: {
      type: [String, Number, null],
      default: null,
    },
    options: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, attrs }) {
    return () =>
      h(
        'select',
        {
          ...attrs,
          value: props.modelValue ?? '',
          onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value || null),
        },
        (props.options as Array<{ label: string; value: string | number | null }>).map((option) =>
          h('option', { value: option.value ?? '' }, option.label),
        ),
      );
  },
});

const tableStub = defineComponent({
  name: 'TTableStub',
  props: {
    data: {
      type: Array,
      default: () => [],
    },
  },
  setup(props, { slots }) {
    return () => {
      if (props.data.length === 0) {
        return h('div', slots.empty?.());
      }

      return h(
        'div',
        (props.data as Array<Record<string, unknown>>).map((row, index) =>
          h('div', { 'data-testid': `permission-row-${index}` }, [
            slots.permission?.({ row }),
            h('span', { 'data-testid': `permission-code-${index}` }, String(row.code ?? '')),
            h('span', { 'data-testid': `permission-category-${index}` }, String(row.category ?? '')),
            h('span', { 'data-testid': `permission-created-at-${index}` }, String(row.created_at ?? '')),
            h('span', { 'data-testid': `permission-updated-at-${index}` }, String(row.updated_at ?? '')),
            h('span', { 'data-testid': `permission-role-count-${index}` }, String(row.role_binding_count ?? '')),
          ]),
        ),
      );
    };
  },
});

const drawerStub = defineComponent({
  name: 'TDrawerStub',
  props: {
    visible: {
      type: Boolean,
      default: false,
    },
  },
  setup(props, { slots }) {
    return () => (props.visible ? h('section', slots.default?.()) : null);
  },
});

function mountPermissionPage() {
  return mount(PermissionPage, {
    global: {
      stubs: {
        't-button': buttonStub,
        't-checkbox': passthroughStub,
        't-checkbox-group': passthroughStub,
        't-drawer': drawerStub,
        't-empty': passthroughStub,
        't-input': inputStub,
        't-pagination': passthroughStub,
        't-select': selectStub,
        't-table': tableStub,
        't-tag': passthroughStub,
      },
    },
  });
}

describe('PermissionPage', () => {
  beforeEach(() => {
    rbacApiMocks.getPermissions.mockReset();
    messageMocks.error.mockReset();
    messageMocks.warning.mockReset();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('loads permissions on mount', async () => {
    rbacApiMocks.getPermissions.mockResolvedValue({
      items: [
        {
          id: 1,
          code: 'permission.read',
          display: 'Permission Read',
          description: null,
          category: 'rbac',
          created_at: '2026-05-22T10:00:00Z',
          updated_at: '2026-05-23T10:00:00Z',
          role_binding_count: 2,
        },
      ],
    });

    const wrapper = mountPermissionPage();
    await flushPromises();

    expect(wrapper.attributes('data-page-type')).toBe('list-form-detail');
    expect(rbacApiMocks.getPermissions).toHaveBeenCalledTimes(1);
    expect(wrapper.text()).toContain('Permission Read');
    expect(wrapper.get('[data-testid="permission-code-0"]').text()).toBe('permission.read');
    expect(wrapper.get('[data-testid="permission-created-at-0"]').text()).toBeTruthy();
    expect(wrapper.text()).toContain('rbac.permissionList.factSourceHint');
    expect(wrapper.text()).not.toContain('rbac.permissionList.detail');
  });

  it('filters permissions by keyword', async () => {
    rbacApiMocks.getPermissions.mockResolvedValue({
      items: [
        {
          id: 1,
          code: 'permission.read',
          display: 'Permission Read',
          description: null,
          category: 'rbac',
          created_at: '2026-05-22T10:00:00Z',
          updated_at: '2026-05-23T10:00:00Z',
          role_binding_count: 2,
        },
        {
          id: 2,
          code: 'user.create',
          display: 'Create User',
          description: 'Create users',
          category: 'user',
          created_at: '2026-05-21T10:00:00Z',
          updated_at: '2026-05-23T11:00:00Z',
          role_binding_count: 1,
        },
      ],
    });

    const wrapper = mountPermissionPage();
    await flushPromises();

    await wrapper.get('.toolbar__search').setValue('user.create');
    await flushPromises();

    expect(wrapper.find('[data-testid="permission-row-0"]').text()).toContain('user.create');
    expect(wrapper.text()).not.toContain('permission.read');
  });
});
