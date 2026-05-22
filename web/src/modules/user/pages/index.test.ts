import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { RBAC_PERMISSION_CODE } from '@/modules/rbac/contract/permissions';
import { USER_PERMISSION_CODE } from '@/modules/user/contract/permissions';

import UserPage from './index.vue';

const userApiMocks = vi.hoisted(() => ({
  getUsers: vi.fn(),
}));

const roleApiMocks = vi.hoisted(() => ({
  assignUserRoles: vi.fn(),
  getRoles: vi.fn(),
  getUserRoleBindings: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
  warning: vi.fn(),
}));

const permissionState = vi.hoisted(() => ({
  grantedCodes: [] as string[],
}));

vi.mock('@/modules/user/api/users', () => ({
  getUsers: userApiMocks.getUsers,
}));

vi.mock('@/modules/user/api/user-roles', () => ({
  assignUserRoles: roleApiMocks.assignUserRoles,
  getRoles: roleApiMocks.getRoles,
  getUserRoleBindings: roleApiMocks.getUserRoleBindings,
}));

vi.mock('@/store', () => ({
  usePermissionStore: () => ({
    hasAnyPermission: (codes: string[]) => codes.some((code) => permissionState.grantedCodes.includes(code)),
    hasPermission: (code: string) => permissionState.grantedCodes.includes(code),
  }),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
    locale: {
      value: 'en-US',
    },
  }),
}));

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: {},
  }),
  useRouter: () => ({
    replace: vi.fn(),
  }),
}));

vi.mock('tdesign-vue-next', () => ({
  MessagePlugin: {
    error: messageMocks.error,
    success: messageMocks.success,
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
    title: {
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
  props: {
    disabled: {
      type: Boolean,
      default: false,
    },
    loading: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['click'],
  setup(props, { emit, slots, attrs }) {
    return () =>
      h(
        'button',
        {
          ...attrs,
          disabled: props.disabled,
          'data-loading': String(props.loading),
          onClick: (event: MouseEvent) => {
            if (!props.disabled) {
              emit('click', event);
            }
          },
        },
        slots.default?.(),
      );
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
    columns: {
      type: Array,
      default: () => [],
    },
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
          h('div', { 'data-testid': `user-row-${index}` }, [
            slots.user?.({ row }),
            slots.status?.({ row }),
            slots.roles?.({ row }),
            slots.operation?.({ row }),
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

const paginationStub = defineComponent({
  name: 'TPaginationStub',
  props: {
    current: {
      type: Number,
      default: 1,
    },
    pageSize: {
      type: Number,
      default: 10,
    },
  },
  setup(_, { attrs }) {
    return () => h('div', { ...attrs, 'data-testid': 'pagination' });
  },
});

const checkboxGroupStub = defineComponent({
  name: 'TCheckboxGroupStub',
  props: {
    disabled: {
      type: Boolean,
      default: false,
    },
    modelValue: {
      type: Array<number>,
      default: () => [],
    },
  },
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        {
          'data-testid': 'role-checkbox-group',
          'data-disabled': String(props.disabled),
          'data-selected-role-ids': JSON.stringify(props.modelValue),
        },
        slots.default?.(),
      );
  },
});

const checkboxStub = defineComponent({
  name: 'TCheckboxStub',
  props: {
    value: {
      type: [Number, Boolean],
      default: undefined,
    },
  },
  setup(props, { slots }) {
    return () => h('label', { 'data-role-id': String(props.value ?? '') }, slots.default?.());
  },
});

function createUserListResponse() {
  return {
    items: [
      {
        id: 7,
        username: 'alice',
        display: 'Alice',
        status: 'enabled',
        created_at: '2026-05-17T00:00:00Z',
        updated_at: '2026-05-17T00:00:00Z',
      },
    ],
  };
}

function createRoleListResponse() {
  return {
    items: [
      {
        id: 2,
        name: 'editor',
        display: 'Editor',
        description: 'Editor role',
        builtin: false,
        updated_at: '2026-05-18T00:00:00Z',
        permission_count: 3,
        user_count: 1,
      },
    ],
  };
}

function mountUserPage() {
  return mount(UserPage, {
    global: {
      directives: {
        permission: {
          mounted() {},
        },
      },
      stubs: {
        't-button': buttonStub,
        't-checkbox': checkboxStub,
        't-checkbox-group': checkboxGroupStub,
        't-dropdown': passthroughStub,
        't-drawer': drawerStub,
        't-empty': passthroughStub,
        't-input': inputStub,
        't-pagination': paginationStub,
        't-select': selectStub,
        't-table': tableStub,
        't-tag': passthroughStub,
      },
    },
  });
}

describe('UserPage', () => {
  beforeEach(() => {
    permissionState.grantedCodes = [];
    userApiMocks.getUsers.mockReset();
    roleApiMocks.assignUserRoles.mockReset();
    roleApiMocks.getRoles.mockReset();
    roleApiMocks.getUserRoleBindings.mockReset();
    messageMocks.error.mockReset();
    messageMocks.success.mockReset();
    messageMocks.warning.mockReset();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('loads users on mount without rendering role management controls for read-only sessions', async () => {
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());

    const wrapper = mountUserPage();
    await flushPromises();

    expect(userApiMocks.getUsers).toHaveBeenCalledTimes(1);
    expect(wrapper.attributes('data-page-type')).toBe('list-form-detail');
    expect(wrapper.text()).toContain('Alice');
    expect(wrapper.text()).not.toContain('user.userList.assignRoles');
    expect(wrapper.text()).not.toContain('user.userList.stats.totalUsers');
    expect(wrapper.text()).not.toContain('user.userList.stats.recentCreated');
  });

  it('renders the create action when the current session has user.create permission', async () => {
    permissionState.grantedCodes = [USER_PERMISSION_CODE.CREATE];
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());

    const wrapper = mountUserPage();
    await flushPromises();

    expect(wrapper.find('[data-testid="user-create"]').exists()).toBe(true);
    await wrapper.get('[data-testid="user-create"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="drawer"]').attributes('data-header')).toBe('user.userList.form.createTitle');
  });

  it('keeps role assignment blocked when the current snapshot fails to load', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.USER_ROLE_READ, RBAC_PERMISSION_CODE.USER_ROLE_ASSIGN];
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());
    roleApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    roleApiMocks.getUserRoleBindings
      .mockResolvedValueOnce({ role_ids: [2] })
      .mockRejectedValueOnce(new Error('selection load failed'));

    const wrapper = mountUserPage();
    await flushPromises();

    await wrapper.get('[data-testid="user-manage-roles"]').trigger('click');
    await flushPromises();

    expect(roleApiMocks.getUserRoleBindings).toHaveBeenCalledWith(7);
    expect(wrapper.text()).toContain('selection load failed');
    expect(wrapper.get('[data-testid="role-checkbox-group"]').attributes('data-disabled')).toBe('true');
    expect(wrapper.get('[data-testid="user-role-save"]').attributes('disabled')).toBeDefined();
  });

  it('submits the selected role snapshot and closes the drawer on success', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.USER_ROLE_READ, RBAC_PERMISSION_CODE.USER_ROLE_ASSIGN];
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());
    roleApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    roleApiMocks.getUserRoleBindings.mockResolvedValue({ role_ids: [2] });
    roleApiMocks.assignUserRoles.mockResolvedValue(null);

    const wrapper = mountUserPage();
    await flushPromises();

    await wrapper.get('[data-testid="user-manage-roles"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="user-role-save"]').trigger('click');
    await flushPromises();

    expect(roleApiMocks.assignUserRoles).toHaveBeenCalledWith(7, { role_ids: [2] });
    expect(messageMocks.success).toHaveBeenCalledWith('user.userList.assignSuccess');
    expect(wrapper.find('[data-testid="user-role-drawer"]').exists()).toBe(false);
  });

  it('renders the table empty state and clears filters from the empty action area', async () => {
    userApiMocks.getUsers.mockResolvedValue({ items: [] });
    roleApiMocks.getRoles.mockResolvedValue({ items: [] });

    const wrapper = mountUserPage();
    await flushPromises();

    expect(wrapper.text()).toContain('user.userList.emptyTitle');
    expect(wrapper.text()).toContain('user.userList.emptyDescription');
    expect(wrapper.find('[data-testid="user-empty-clear-filters"]').exists()).toBe(false);

    await wrapper.get('input[placeholder="user.userList.toolbar.searchPlaceholder"]').setValue('alice');
    await flushPromises();

    expect(wrapper.find('[data-testid="user-empty-clear-filters"]').exists()).toBe(true);
    await wrapper.get('[data-testid="user-empty-clear-filters"]').trigger('click');
    await flushPromises();

    expect(
      (wrapper.get('input[placeholder="user.userList.toolbar.searchPlaceholder"]').element as HTMLInputElement).value,
    ).toBe('');
  });
});
