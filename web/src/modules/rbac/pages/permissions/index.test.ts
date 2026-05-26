import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import PermissionPage from './index.vue';

const i18nMessages: Record<string, string> = {
  'rbac.permissionCatalog.permissionRead.display': 'Read Permissions Localized',
  'rbac.permissionCatalog.permissionRead.description': 'Localized permission description',
  'rbac.permissionCatalog.userCreate.display': 'Create Users Localized',
  'rbac.permissionCatalog.userCreate.description': 'Localized create-user description',
  'rbac.permissionList.detail': 'Permission Details',
  'rbac.permissionList.detailTitle': 'Permission Details',
  'rbac.permissionList.detailLoadFailed': 'Failed to load permission details',
  'rbac.permissionList.detailLoadFailedTitle': 'Permission details are temporarily unavailable',
  'rbac.permissionList.detailFields.description': 'Description',
  'rbac.permissionList.emptyDescription': 'No description',
  'rbac.permissionList.emptyFilteredDescription': 'No permissions match the current filters',
};

const rbacApiMocks = vi.hoisted(() => ({
  getPermissionDetail: vi.fn(),
  getPermissions: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  warning: vi.fn(),
}));

vi.mock('../../api/rbac', () => ({
  getPermissionDetail: rbacApiMocks.getPermissionDetail,
  getPermissions: rbacApiMocks.getPermissions,
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => i18nMessages[key] ?? key,
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
    title: {
      type: String,
      default: '',
    },
    description: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () => h('div', [props.title, props.description, slots.default?.(), slots.action?.()]);
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
            slots.description?.({ row }),
            slots.operation?.({ row }),
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
    header: {
      type: String,
      default: '',
    },
    visible: {
      type: Boolean,
      default: false,
    },
  },
  setup(props, { slots }) {
    return () =>
      props.visible ? h('section', { 'data-testid': 'drawer', 'data-header': props.header }, slots.default?.()) : null;
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
    rbacApiMocks.getPermissionDetail.mockReset();
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
    expect(rbacApiMocks.getPermissions).toHaveBeenLastCalledWith({});
    expect(wrapper.text()).toContain('Read Permissions Localized');
    expect(wrapper.text()).toContain('Localized permission description');
    expect(wrapper.get('[data-testid="permission-code-0"]').text()).toBe('permission.read');
    expect(wrapper.get('[data-testid="permission-created-at-0"]').text()).toBeTruthy();
    expect(wrapper.text()).toContain('rbac.permissionList.factSourceHint');
    expect(wrapper.text()).toContain('Permission Details');
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

    expect(rbacApiMocks.getPermissions).toHaveBeenLastCalledWith({ keyword: 'user.create' });
  });

  it('falls back to API copy for unknown permission codes', async () => {
    rbacApiMocks.getPermissions.mockResolvedValue({
      items: [
        {
          id: 1,
          code: 'custom.permission',
          display: 'Custom Permission',
          description: 'Custom description',
          category: 'custom',
          created_at: '2026-05-22T10:00:00Z',
          updated_at: '2026-05-23T10:00:00Z',
          role_binding_count: 0,
        },
      ],
    });

    const wrapper = mountPermissionPage();
    await flushPromises();

    expect(wrapper.text()).toContain('Custom Permission');
    expect(wrapper.text()).toContain('Custom description');
  });

  it('renders the default empty state without filter actions', async () => {
    rbacApiMocks.getPermissions.mockResolvedValue({ items: [] });

    const wrapper = mountPermissionPage();
    await flushPromises();

    expect(wrapper.text()).toContain('rbac.permissionList.emptyTitle');
    expect(wrapper.text()).toContain('rbac.permissionList.empty');
    expect(wrapper.find('[data-testid="permission-empty-clear-filters"]').exists()).toBe(false);
  });

  it('sends filtered requests and clears filters from the toolbar action', async () => {
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

    await wrapper.get('.toolbar__search').setValue('no-match');
    await flushPromises();

    expect(rbacApiMocks.getPermissions).toHaveBeenLastCalledWith({ keyword: 'no-match' });

    const clearButton = wrapper
      .findAll('button')
      .find((button) => button.text() === 'rbac.permissionList.toolbar.clearFilters');
    expect(clearButton).toBeTruthy();
    await clearButton?.trigger('click');
    await flushPromises();

    expect(rbacApiMocks.getPermissions).toHaveBeenLastCalledWith({});
    expect((wrapper.get('.toolbar__search').element as HTMLInputElement).value).toBe('');
    expect(wrapper.text()).toContain('Read Permissions Localized');
  });

  it('falls back to the empty description when the localized and API descriptions are absent', async () => {
    rbacApiMocks.getPermissions.mockResolvedValue({
      items: [
        {
          id: 1,
          code: 'custom.permission',
          display: 'Custom Permission',
          description: null,
          category: 'custom',
          created_at: '2026-05-22T10:00:00Z',
          updated_at: '2026-05-23T10:00:00Z',
          role_binding_count: 0,
        },
      ],
    });

    const wrapper = mountPermissionPage();
    await flushPromises();

    expect(wrapper.text()).toContain('No description');
  });

  it('opens the detail drawer and consumes permission detail when available', async () => {
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
    rbacApiMocks.getPermissionDetail.mockResolvedValue({
      id: 1,
      code: 'permission.read',
      display: 'Permission Read',
      description: 'Detail description',
      category: 'rbac',
      created_at: '2026-05-22T10:00:00Z',
      updated_at: '2026-05-24T10:00:00Z',
      role_binding_count: 3,
    });

    const wrapper = mountPermissionPage();
    await flushPromises();

    await wrapper.get('[data-testid="permission-detail"]').trigger('click');
    await flushPromises();

    expect(rbacApiMocks.getPermissionDetail).toHaveBeenCalledWith(1);
    expect(wrapper.findAll('[data-testid="drawer"]')).toHaveLength(1);
    expect(wrapper.text()).toContain('Localized permission description');
    expect(wrapper.text()).toContain('3');
    expect(wrapper.text()).toContain('May 24, 2026, 6:00 PM');
  });

  it('keeps the drawer open with fallback snapshot when detail loading fails', async () => {
    rbacApiMocks.getPermissions.mockResolvedValue({
      items: [
        {
          id: 2,
          code: 'custom.permission',
          display: 'Custom Permission',
          description: 'Snapshot description',
          category: 'custom',
          created_at: '2026-05-22T10:00:00Z',
          updated_at: '2026-05-23T10:00:00Z',
          role_binding_count: 1,
        },
      ],
    });
    rbacApiMocks.getPermissionDetail.mockRejectedValue(new Error('detail failed'));

    const wrapper = mountPermissionPage();
    await flushPromises();

    await wrapper.get('[data-testid="permission-detail"]').trigger('click');
    await flushPromises();

    expect(messageMocks.error).toHaveBeenCalledWith('Failed to load permission details');
    expect(wrapper.text()).toContain('Permission details are temporarily unavailable');
    expect(wrapper.text()).toContain('Snapshot description');
  });
});
