import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import { API_CODE } from '@/contracts/api/codes';
import { RBAC_PERMISSION_CODE } from '@/modules/rbac/contract/permissions';
import { USER_PERMISSION_CODE } from '@/modules/user/contract/permissions';

import UserPage from './index.vue';

const userApiMocks = vi.hoisted(() => ({
  createUser: vi.fn(),
  deleteUser: vi.fn(),
  getUsers: vi.fn(),
  resetUserPassword: vi.fn(),
  updateUser: vi.fn(),
  updateUserStatus: vi.fn(),
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
  createUser: userApiMocks.createUser,
  deleteUser: userApiMocks.deleteUser,
  getUsers: userApiMocks.getUsers,
  resetUserPassword: userApiMocks.resetUserPassword,
  updateUser: userApiMocks.updateUser,
  updateUserStatus: userApiMocks.updateUserStatus,
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

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>();
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: {
        value: 'en-US',
      },
    }),
  };
});

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

const dropdownStub = defineComponent({
  name: 'TDropdownStub',
  props: {
    options: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h('div', [
        slots.default?.(),
        ...(props.options as Array<{ value: string; content: string; disabled?: boolean }>).map((option) =>
          h(
            'button',
            {
              type: 'button',
              disabled: Boolean(option.disabled),
              'data-testid': `dropdown-option-${option.value}`,
              onClick: () => {
                if (!option.disabled) {
                  emit('click', { value: option.value });
                }
              },
            },
            option.content,
          ),
        ),
      ]);
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

const formStub = defineComponent({
  name: 'TFormStub',
  props: {
    data: {
      type: Object,
      default: () => ({}),
    },
    rules: {
      type: Object,
      default: () => ({}),
    },
  },
  emits: ['submit'],
  setup(props, { emit, expose, slots, attrs }) {
    const validateMessages = ref<Record<string, string[]>>({});

    function clearValidate(fields?: string[]) {
      if (!fields?.length) {
        validateMessages.value = {};
        return;
      }

      const nextMessages = { ...validateMessages.value };
      fields.forEach((field) => {
        delete nextMessages[field];
      });
      validateMessages.value = nextMessages;
    }

    function setValidateMessage(message: Record<string, Array<{ message: string }>>) {
      const nextMessages = { ...validateMessages.value };
      Object.entries(message).forEach(([field, items]) => {
        nextMessages[field] = items.map((item) => item.message);
      });
      validateMessages.value = nextMessages;
    }

    expose({
      clearValidate,
      setValidateMessage,
    });

    async function validate() {
      const nextMessages: Record<string, string[]> = {};
      const formData = props.data as Record<string, unknown>;
      const formRules = props.rules as Record<
        string,
        Array<{ required?: boolean; message?: string; validator?: (value: unknown) => unknown | Promise<unknown> }>
      >;

      for (const [field, rules] of Object.entries(formRules)) {
        const value = formData[field];

        for (const rule of rules ?? []) {
          if (rule.required && !value) {
            nextMessages[field] = [rule.message ?? ''];
            break;
          }

          if (typeof rule.validator === 'function') {
            const result = await rule.validator(value);
            if (result !== true) {
              const message =
                typeof result === 'object' && result && 'message' in result ? String(result.message ?? '') : '';
              nextMessages[field] = [message || rule.message || ''];
              break;
            }
          }
        }
      }

      validateMessages.value = nextMessages;

      if (Object.keys(nextMessages).length === 0) {
        return { validateResult: true, firstError: '' };
      }

      return {
        validateResult: Object.fromEntries(
          Object.entries(nextMessages).map(([field, messages]) => [
            field,
            messages.map((message) => ({ message, type: 'error' })),
          ]),
        ),
        firstError: Object.values(nextMessages)[0]?.[0] ?? '',
      };
    }

    return () =>
      h(
        'form',
        {
          ...attrs,
          onSubmit: async (event: Event) => {
            event.preventDefault();
            emit('submit', await validate());
          },
        },
        [
          slots.default?.({ data: props.data }),
          Object.entries(validateMessages.value).map(([field, messages]) =>
            h(
              'div',
              { 'data-testid': `validate-${field}` },
              messages.map((message) => h('p', message)),
            ),
          ),
        ],
      );
  },
});

const formItemStub = defineComponent({
  name: 'TFormItemStub',
  props: {
    label: {
      type: String,
      default: '',
    },
    name: {
      type: String,
      default: '',
    },
    tips: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () =>
      h('div', { 'data-testid': `form-item-${props.name}` }, [
        props.label ? h('label', props.label) : null,
        slots.default?.(),
        props.tips ? h('p', { 'data-testid': `form-item-tips-${props.name}` }, props.tips) : null,
      ]);
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

const dialogStub = defineComponent({
  name: 'TDialogStub',
  props: {
    visible: {
      type: Boolean,
      default: false,
    },
    header: {
      type: String,
      default: '',
    },
  },
  emits: ['confirm', 'close'],
  setup(props, { emit, slots }) {
    return () =>
      props.visible
        ? h('section', { 'data-testid': 'dialog', 'data-header': props.header }, [
            slots.default?.(),
            h(
              'button',
              {
                type: 'button',
                'data-testid': 'dialog-confirm',
                onClick: () => emit('confirm'),
              },
              'confirm',
            ),
          ])
        : null;
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
        't-dialog': dialogStub,
        't-dropdown': dropdownStub,
        't-drawer': drawerStub,
        't-empty': passthroughStub,
        't-form': formStub,
        't-form-item': formItemStub,
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
    userApiMocks.createUser.mockReset();
    userApiMocks.deleteUser.mockReset();
    userApiMocks.getUsers.mockReset();
    userApiMocks.resetUserPassword.mockReset();
    userApiMocks.updateUser.mockReset();
    userApiMocks.updateUserStatus.mockReset();
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

  it('shows the lightweight password help text in the create drawer', async () => {
    permissionState.grantedCodes = [USER_PERMISSION_CODE.CREATE];
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());

    const wrapper = mountUserPage();
    await flushPromises();
    await wrapper.get('[data-testid="user-create"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="form-item-tips-password"]').text()).toContain(
      'user.userList.form.passwordPolicy.hint',
    );
    expect(wrapper.text()).not.toContain('user.userList.form.passwordPolicy.strength');
  });

  it('shows a field error and hides the help text when the initial password is invalid', async () => {
    permissionState.grantedCodes = [USER_PERMISSION_CODE.CREATE];
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());

    const wrapper = mountUserPage();
    await flushPromises();
    await wrapper.get('[data-testid="user-create"]').trigger('click');
    await flushPromises();

    const passwordInput = wrapper.get('input[placeholder="user.userList.form.passwordPlaceholder"]');
    await passwordInput.setValue('short');
    await flushPromises();

    expect(wrapper.get('[data-testid="validate-password"]').text()).toContain(
      'user.userList.form.passwordPolicy.error',
    );
    expect(wrapper.find('[data-testid="form-item-tips-password"]').exists()).toBe(false);
  });

  it('restores the help text after the initial password satisfies the policy', async () => {
    permissionState.grantedCodes = [USER_PERMISSION_CODE.CREATE];
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());

    const wrapper = mountUserPage();
    await flushPromises();
    await wrapper.get('[data-testid="user-create"]').trigger('click');
    await flushPromises();

    await wrapper.get('input[placeholder="user.userList.form.passwordPlaceholder"]').setValue('short');
    await flushPromises();
    await wrapper.get('input[placeholder="user.userList.form.passwordPlaceholder"]').setValue('BetterPassword123');
    await flushPromises();

    expect(wrapper.find('[data-testid="validate-password"]').exists()).toBe(false);
    expect(wrapper.get('[data-testid="form-item-tips-password"]').text()).toContain(
      'user.userList.form.passwordPolicy.hint',
    );
  });

  it('binds backend password policy violations to the password field instead of a global error toast', async () => {
    permissionState.grantedCodes = [USER_PERMISSION_CODE.CREATE];
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());
    userApiMocks.createUser.mockRejectedValue({
      isApiRequestError: true,
      code: API_CODE.AUTH_PASSWORD_POLICY_VIOLATION,
      message: '新密码不符合安全要求',
      messageKey: '',
      responseData: {
        data: {
          field: 'password',
        },
      },
    });

    const wrapper = mountUserPage();
    await flushPromises();
    await wrapper.get('[data-testid="user-create"]').trigger('click');
    await flushPromises();

    await wrapper.get('input[placeholder="user.userList.form.usernamePlaceholder"]').setValue('carol');
    await wrapper.get('input[placeholder="user.userList.form.displayPlaceholder"]').setValue('Carol');
    await wrapper.get('input[placeholder="user.userList.form.passwordPlaceholder"]').setValue('BetterPassword123');
    await wrapper.get('form').trigger('submit');
    await flushPromises();

    expect(wrapper.get('[data-testid="validate-password"]').text()).toContain('新密码不符合安全要求');
    expect(messageMocks.error).not.toHaveBeenCalledWith('user.userList.createFailed');
    expect(messageMocks.error).not.toHaveBeenCalledWith('新密码不符合安全要求');
  });

  it('binds backend invalid argument errors to the matching create-user field', async () => {
    permissionState.grantedCodes = [USER_PERMISSION_CODE.CREATE];
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());
    userApiMocks.createUser.mockRejectedValue({
      isApiRequestError: true,
      code: API_CODE.COMMON_INVALID_ARGUMENT,
      message: '请求参数不合法',
      messageKey: '',
      responseData: {
        data: {
          field: 'username',
        },
      },
    });

    const wrapper = mountUserPage();
    await flushPromises();
    await wrapper.get('[data-testid="user-create"]').trigger('click');
    await flushPromises();

    await wrapper.get('input[placeholder="user.userList.form.usernamePlaceholder"]').setValue('carol');
    await wrapper.get('input[placeholder="user.userList.form.displayPlaceholder"]').setValue('Carol');
    await wrapper.get('input[placeholder="user.userList.form.passwordPlaceholder"]').setValue('BetterPassword123');
    await wrapper.get('form').trigger('submit');
    await flushPromises();

    expect(wrapper.get('[data-testid="validate-username"]').text()).toContain('请求参数不合法');
    expect(messageMocks.error).not.toHaveBeenCalled();
  });

  it('binds backend invalid argument errors to the matching edit-user field', async () => {
    permissionState.grantedCodes = [USER_PERMISSION_CODE.UPDATE];
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());
    userApiMocks.updateUser.mockRejectedValue({
      isApiRequestError: true,
      code: API_CODE.COMMON_INVALID_ARGUMENT,
      message: '用户名已存在',
      messageKey: '',
      responseData: {
        data: {
          field: 'username',
        },
      },
    });

    const wrapper = mountUserPage();
    await flushPromises();

    await wrapper.get('[data-testid="user-edit"]').trigger('click');
    await flushPromises();
    await wrapper.get('input[placeholder="user.userList.form.usernamePlaceholder"]').setValue('alice');
    await wrapper.get('input[placeholder="user.userList.form.displayPlaceholder"]').setValue('Alice Updated');
    await wrapper.get('form').trigger('submit');
    await flushPromises();

    expect(wrapper.get('[data-testid="validate-username"]').text()).toContain('用户名已存在');
    expect(messageMocks.error).not.toHaveBeenCalled();
  });

  it('binds reset-password API errors to the password field inside the dialog', async () => {
    permissionState.grantedCodes = [USER_PERMISSION_CODE.UPDATE];
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());
    userApiMocks.resetUserPassword.mockRejectedValue({
      isApiRequestError: true,
      code: API_CODE.AUTH_PASSWORD_POLICY_VIOLATION,
      message: '新密码不符合安全要求',
      messageKey: '',
      responseData: {
        data: {
          field: 'new_password',
        },
      },
    });

    const wrapper = mountUserPage();
    await flushPromises();

    await wrapper.get('[data-testid="dropdown-option-reset-password"]').trigger('click');
    await flushPromises();
    await wrapper.get('input[placeholder="user.userList.resetPasswordDialog.passwordPlaceholder"]').setValue('short');
    await wrapper.get('[data-testid="dialog-confirm"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="validate-password"]').text()).toContain('新密码不符合安全要求');
    expect(messageMocks.error).not.toHaveBeenCalledWith('user.userList.resetPasswordFailed');
  });

  it('uses the API error message for status update failures on covered write routes', async () => {
    permissionState.grantedCodes = [USER_PERMISSION_CODE.DISABLE];
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());
    userApiMocks.updateUserStatus.mockRejectedValue({
      isApiRequestError: true,
      status: 404,
      code: 'USER_NOT_FOUND',
      message: '用户不存在',
      messageKey: 'user.not_found',
      responseData: {},
    });
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true);

    const wrapper = mountUserPage();
    await flushPromises();

    await wrapper.get('[data-testid="dropdown-option-toggle-status"]').trigger('click');
    await flushPromises();

    expect(messageMocks.error).toHaveBeenCalledWith('用户不存在');
    expect(messageMocks.error).not.toHaveBeenCalledWith('user.userList.statusUpdateFailed');

    confirmSpy.mockRestore();
  });

  it('keeps role assignment blocked when the current snapshot fails to load', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.USER_ROLE_READ, RBAC_PERMISSION_CODE.USER_ROLE_ASSIGN];
    userApiMocks.getUsers.mockResolvedValue(createUserListResponse());
    roleApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    roleApiMocks.getUserRoleBindings.mockRejectedValueOnce(new Error('selection load failed'));

    const wrapper = mountUserPage();
    await flushPromises();

    await wrapper.get('[data-testid="user-manage-roles"]').trigger('click');
    await flushPromises();

    expect(roleApiMocks.getUserRoleBindings).toHaveBeenCalledWith(7);
    expect(wrapper.text()).toContain('user.userList.roleDialog.selectionLoadFailed');
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
