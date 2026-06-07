import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import { RBAC_PERMISSION_CODE } from '../contract/permissions';
import RolePage from './index.vue';

const rbacApiMocks = vi.hoisted(() => ({
  addRolePermissions: vi.fn(),
  createRole: vi.fn(),
  deleteRole: vi.fn(),
  getPermissions: vi.fn(),
  getRoleDetail: vi.fn(),
  getRolePermissionBindings: vi.fn(),
  getRoles: vi.fn(),
  removeRolePermissions: vi.fn(),
  replaceRolePermissions: vi.fn(),
  updateRoleStatus: vi.fn(),
  updateRole: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
  warning: vi.fn(),
}));

const permissionState = vi.hoisted(() => ({
  grantedCodes: [] as string[],
}));

const tabSnapshotState = vi.hoisted(() => ({
  activeTabKey: '/access-control/roles',
  snapshots: {} as Record<string, Record<string, unknown>>,
}));

vi.mock('../api/rbac', () => ({
  addRolePermissions: rbacApiMocks.addRolePermissions,
  createRole: rbacApiMocks.createRole,
  deleteRole: rbacApiMocks.deleteRole,
  getPermissions: rbacApiMocks.getPermissions,
  getRoleDetail: rbacApiMocks.getRoleDetail,
  getRolePermissionBindings: rbacApiMocks.getRolePermissionBindings,
  getRoles: rbacApiMocks.getRoles,
  removeRolePermissions: rbacApiMocks.removeRolePermissions,
  replaceRolePermissions: rbacApiMocks.replaceRolePermissions,
  updateRoleStatus: rbacApiMocks.updateRoleStatus,
  updateRole: rbacApiMocks.updateRole,
}));

vi.mock('@/store', () => ({
  usePermissionStore: () => ({
    hasAnyPermission: (codes: string[]) => codes.some((code) => permissionState.grantedCodes.includes(code)),
    hasPermission: (code: string) => permissionState.grantedCodes.includes(code),
  }),
  useTabsRouterStore: () => ({
    activeTabKey: tabSnapshotState.activeTabKey,
    getPageSnapshot: (tabKey?: string) => (tabKey ? tabSnapshotState.snapshots[tabKey] : undefined),
    setPageSnapshot: (tabKey?: string, snapshot?: Record<string, unknown>) => {
      if (tabKey && snapshot) {
        tabSnapshotState.snapshots[tabKey] = JSON.parse(JSON.stringify(snapshot));
      }
    },
  }),
}));

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>();
  const translations: Record<string, string> = {
    'rbac.permissionCatalog.futureRead.display': 'Future Read Localized',
    'rbac.permissionCatalog.futureRead.description': 'Future description localized',
    'rbac.permissionCatalog.permissionRead.display': 'Read Permissions Localized',
    'rbac.permissionCatalog.permissionRead.description': 'Localized permission description',
    'rbac.permissionCatalog.roleUpdate.display': 'Update Roles Localized',
    'rbac.permissionCatalog.roleUpdate.description': 'Localized update-role description',
    'rbac.permissionCatalog.auditRead.display': 'Read Audit Localized',
    'rbac.permissionCatalog.auditRead.description': 'Localized audit description',
  };
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => translations[key] ?? key,
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

const cardStub = defineComponent({
  name: 'TCardStub',
  props: {
    title: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots, attrs }) {
    return () => h('section', attrs, [props.title, slots.default?.()]);
  },
});

const alertStub = defineComponent({
  name: 'TAlertStub',
  props: {
    title: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots, attrs }) {
    return () => h('aside', attrs, [props.title, slots.message?.(), slots.default?.()]);
  },
});

const descriptionsStub = defineComponent({
  name: 'TDescriptionsStub',
  setup(_, { slots }) {
    return () => h('dl', slots.default?.());
  },
});

const descriptionsItemStub = defineComponent({
  name: 'TDescriptionsItemStub',
  props: {
    label: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () => h('div', [h('dt', props.label), h('dd', slots.default?.())]);
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
      h(
        'div',
        {
          'data-testid': 'dropdown',
          'data-options': JSON.stringify(props.options),
          onClick: () => emit('click', { value: 'noop' }),
        },
        slots.default?.(),
      );
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

const textareaStub = defineComponent({
  name: 'TTextareaStub',
  props: {
    modelValue: {
      type: String,
      default: '',
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, attrs }) {
    return () =>
      h('textarea', {
        ...attrs,
        value: props.modelValue,
        onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLTextAreaElement).value),
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
          h('div', { 'data-testid': `role-row-${index}` }, [slots.role?.({ row }), slots.operation?.({ row })]),
        ),
      );
    };
  },
});

const drawerStub = defineComponent({
  name: 'TDrawerStub',
  props: {
    closeOnEscKeydown: {
      type: Boolean,
      default: true,
    },
    closeOnOverlayClick: {
      type: Boolean,
      default: true,
    },
    footer: {
      type: [Boolean, Object, String, null],
      default: undefined,
    },
    header: {
      type: String,
      default: '',
    },
    visible: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['close', 'close-btn-click', 'esc-keydown', 'overlay-click', 'update:visible'],
  setup(props, { emit, slots }) {
    return () =>
      props.visible
        ? h(
            'section',
            {
              'data-testid': 'drawer',
              'data-close-on-esc': String(props.closeOnEscKeydown),
              'data-close-on-overlay': String(props.closeOnOverlayClick),
              'data-footer': String(props.footer),
              'data-header': props.header,
            },
            [
              h('button', {
                'data-testid': 'drawer-close-btn',
                onClick: () => emit('close-btn-click', { e: new MouseEvent('click') }),
              }),
              h('button', {
                'data-testid': 'drawer-overlay',
                onClick: () => emit('overlay-click', { e: new MouseEvent('click') }),
              }),
              h('button', {
                'data-testid': 'drawer-esc',
                onClick: () => emit('esc-keydown', { e: new KeyboardEvent('keydown') }),
              }),
              slots.default?.(),
            ],
          )
        : null;
  },
});

const dialogStub = defineComponent({
  name: 'TDialogStub',
  props: {
    body: {
      type: String,
      default: '',
    },
    cancelBtn: {
      type: [String, Object, Boolean],
      default: '',
    },
    confirmBtn: {
      type: [String, Object, Boolean],
      default: '',
    },
    header: {
      type: String,
      default: '',
    },
    visible: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['cancel', 'close', 'confirm', 'update:visible'],
  setup(props, { emit, slots }) {
    const buttonContent = (value: unknown) => {
      if (typeof value === 'string') {
        return value;
      }
      if (value && typeof value === 'object' && 'content' in value) {
        return String((value as { content?: unknown }).content ?? '');
      }
      return '';
    };

    return () =>
      props.visible
        ? h(
            'section',
            {
              'data-testid': 'discard-permission-confirm',
              'data-header': props.header,
              'data-body': props.body,
            },
            [
              h('h2', props.header),
              h('p', props.body),
              slots.default?.(),
              h(
                'button',
                {
                  'data-testid': 'discard-confirm-cancel',
                  onClick: () => emit('cancel', { e: new MouseEvent('click') }),
                },
                buttonContent(props.cancelBtn),
              ),
              h(
                'button',
                {
                  'data-testid': 'discard-confirm-confirm',
                  onClick: () => emit('confirm', { e: new MouseEvent('click') }),
                },
                buttonContent(props.confirmBtn),
              ),
              h(
                'button',
                {
                  'data-testid': 'discard-confirm-close',
                  onClick: () => emit('close', { e: new MouseEvent('click') }),
                },
                'close',
              ),
            ],
          )
        : null;
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
  setup(props, { emit, expose, slots }) {
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

    return () =>
      h(
        'form',
        {
          'data-testid': 'role-form',
          onSubmit: (event: Event) => {
            event.preventDefault();
            emit('submit', { validateResult: true });
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
  },
  setup(props, { slots }) {
    return () =>
      h('div', { 'data-testid': `form-item-${props.name}` }, [
        props.label ? h('label', props.label) : null,
        slots.default?.(),
      ]);
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
  emits: ['update:modelValue'],
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        {
          'data-testid': 'permission-checkbox-group',
          'data-disabled': String(props.disabled),
          'data-selected-permission-ids': JSON.stringify(props.modelValue),
        },
        slots.default?.(),
      );
  },
});

const checkboxStub = defineComponent({
  name: 'TCheckboxStub',
  props: {
    disabled: {
      type: Boolean,
      default: false,
    },
    value: {
      type: [Number, Boolean],
      default: undefined,
    },
  },
  setup(props, { slots }) {
    return () =>
      h(
        'label',
        {
          'data-disabled': String(props.disabled),
          'data-permission-id': String(props.value ?? ''),
        },
        slots.default?.(),
      );
  },
});

function createRoleListResponse() {
  return {
    items: [
      {
        id: 2,
        name: 'editor',
        display: 'Editor',
        description: 'Editor role',
        builtin: false,
        status: 'enabled',
        updated_at: '2026-05-18T00:00:00Z',
        permission_count: 2,
        user_count: 1,
      },
    ],
  };
}

function createBuiltinAdminRoleListResponse() {
  return {
    items: [
      {
        id: 1,
        name: 'admin',
        display: 'Administrator',
        description: 'Builtin administrator',
        builtin: true,
        status: 'enabled',
        updated_at: '2026-05-18T00:00:00Z',
        permission_count: 3,
        user_count: 1,
      },
    ],
  };
}

function createPermissionListResponse() {
  return {
    items: [
      {
        id: 1,
        code: 'permission.read',
        display: 'Permission Read',
        description: 'Read permissions',
        category: 'permission',
        role_binding_count: 1,
      },
      {
        id: 2,
        code: 'role.update',
        display: 'Role Update',
        description: 'Update roles',
        category: 'role',
        role_binding_count: 1,
      },
      {
        id: 3,
        code: 'audit.read',
        display: 'Audit Read',
        description: 'Read audit logs',
        category: 'audit',
        role_binding_count: 1,
      },
    ],
  };
}

function mountRolePage() {
  return mount(RolePage, {
    global: {
      directives: {
        permission: {
          mounted(el, binding) {
            const value = binding.value;
            let allowed = false;

            if (typeof value === 'string') {
              allowed = permissionState.grantedCodes.includes(value);
            } else if (Array.isArray(value)) {
              allowed = value.every((code: string) => permissionState.grantedCodes.includes(code));
            } else if (value && typeof value === 'object') {
              const allOf = Array.isArray(value.allOf) ? value.allOf : [];
              const anyOf = Array.isArray(value.anyOf) ? value.anyOf : [];
              const matchesAll =
                allOf.length === 0 || allOf.every((code: string) => permissionState.grantedCodes.includes(code));
              const matchesAny =
                anyOf.length === 0 || anyOf.some((code: string) => permissionState.grantedCodes.includes(code));
              allowed = matchesAll && matchesAny;
            }

            if (!allowed) {
              el.remove();
            }
          },
        },
      },
      stubs: {
        't-alert': alertStub,
        't-button': buttonStub,
        't-card': cardStub,
        't-checkbox': checkboxStub,
        't-checkbox-group': checkboxGroupStub,
        't-descriptions': descriptionsStub,
        't-descriptions-item': descriptionsItemStub,
        't-dialog': dialogStub,
        't-dropdown': dropdownStub,
        't-drawer': drawerStub,
        't-empty': passthroughStub,
        't-form': formStub,
        't-form-item': formItemStub,
        't-input': inputStub,
        't-select': selectStub,
        't-table': tableStub,
        't-tag': passthroughStub,
        't-textarea': textareaStub,
        't-pagination': passthroughStub,
        't-tooltip': passthroughStub,
      },
    },
  });
}

function updatePermissionSelection(wrapper: ReturnType<typeof mountRolePage>, ids: number[]) {
  const checkboxGroup = wrapper.getComponent(checkboxGroupStub);
  checkboxGroup.vm.$emit('update:modelValue', ids);
}

function selectedPermissionIds(wrapper: ReturnType<typeof mountRolePage>) {
  return JSON.parse(
    wrapper.get('[data-testid="permission-checkbox-group"]').attributes('data-selected-permission-ids') ?? '[]',
  );
}

function setPermissionMutationMode(wrapper: ReturnType<typeof mountRolePage>, mode: 'replace' | 'add' | 'remove') {
  const selects = wrapper.findAll('select');
  const mutationSelect = selects.at(-1);
  if (!mutationSelect) {
    throw new Error('permission mutation select not found');
  }

  return mutationSelect.setValue(mode);
}

describe('RolePage', () => {
  beforeEach(() => {
    permissionState.grantedCodes = [];
    tabSnapshotState.snapshots = {};
    rbacApiMocks.addRolePermissions.mockReset();
    rbacApiMocks.createRole.mockReset();
    rbacApiMocks.deleteRole.mockReset();
    rbacApiMocks.getPermissions.mockReset();
    rbacApiMocks.getRoleDetail.mockReset();
    rbacApiMocks.getRolePermissionBindings.mockReset();
    rbacApiMocks.getRoles.mockReset();
    rbacApiMocks.removeRolePermissions.mockReset();
    rbacApiMocks.replaceRolePermissions.mockReset();
    rbacApiMocks.updateRoleStatus.mockReset();
    rbacApiMocks.updateRole.mockReset();
    messageMocks.error.mockReset();
    messageMocks.success.mockReset();
    messageMocks.warning.mockReset();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('loads roles and permission definitions on mount', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());

    const wrapper = mountRolePage();
    await flushPromises();

    expect(rbacApiMocks.getRoles).toHaveBeenCalledTimes(1);
    expect(rbacApiMocks.getPermissions).toHaveBeenCalledTimes(1);
    expect(wrapper.attributes('data-page-type')).toBe('list-form-detail');
    expect(wrapper.text()).toContain('Editor');
    expect(wrapper.text()).not.toContain('rbac.roleList.stats.totalRoles');
  });

  it('hides the edit button when role.update is missing', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());

    const wrapper = mountRolePage();
    await flushPromises();

    expect(wrapper.find('[data-testid="role-edit"]').exists()).toBe(false);
  });

  it('hides the create action when role.create is missing', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());

    const wrapper = mountRolePage();
    await flushPromises();

    expect(wrapper.find('[data-testid="role-create"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="role-empty-create"]').exists()).toBe(false);
  });

  it('hides the assign-permissions action when role.permission.assign is missing', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());

    const wrapper = mountRolePage();
    await flushPromises();

    expect(wrapper.find('[data-testid="role-assign-permissions"]').exists()).toBe(false);
  });

  it('submits the trimmed create payload', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.ROLE_CREATE];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue({ items: [] });
    rbacApiMocks.createRole.mockResolvedValue({
      id: 4,
      name: 'reviewer',
      display: 'Reviewer',
      description: null,
      builtin: false,
      updated_at: '2026-05-19T00:00:00Z',
      permission_count: 0,
      user_count: 0,
    });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-create"]').trigger('click');
    await flushPromises();
    await wrapper.get('input[placeholder="rbac.roleList.form.namePlaceholder"]').setValue(' reviewer ');
    await wrapper.get('input[placeholder="rbac.roleList.form.displayPlaceholder"]').setValue(' Reviewer ');
    await wrapper.get('textarea[placeholder="rbac.roleList.form.descriptionPlaceholder"]').setValue(' role summary ');
    await wrapper.get('[data-testid="role-form"]').trigger('submit');
    await flushPromises();

    expect(rbacApiMocks.createRole).toHaveBeenCalledWith({
      name: 'reviewer',
      display: 'Reviewer',
      description: 'role summary',
    });
    expect(messageMocks.success).toHaveBeenCalledWith('rbac.roleList.createSuccess');
  });

  it('binds role form invalid-argument errors to the name field', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.ROLE_CREATE];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue({ items: [] });
    rbacApiMocks.createRole.mockRejectedValue({
      isApiRequestError: true,
      status: 400,
      code: 'COMMON_INVALID_ARGUMENT',
      message: '角色编码已存在',
      messageKey: '',
      responseData: {
        data: {
          field: 'name',
        },
      },
    });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-create"]').trigger('click');
    await flushPromises();
    await wrapper.get('input[placeholder="rbac.roleList.form.namePlaceholder"]').setValue('reviewer');
    await wrapper.get('input[placeholder="rbac.roleList.form.displayPlaceholder"]').setValue('Reviewer');
    await wrapper.get('[data-testid="role-form"]').trigger('submit');
    await flushPromises();

    expect(wrapper.get('[data-testid="validate-name"]').text()).toContain('角色编码已存在');
    expect(messageMocks.error).not.toHaveBeenCalled();
  });

  it('keeps replace mode submit disabled until the selection changes', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [2, 1, 2] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="permission-drawer-save"]').attributes('disabled')).toBeDefined();
    expect(rbacApiMocks.replaceRolePermissions).not.toHaveBeenCalled();
  });

  it('allows clearing all permissions in replace mode', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1, 2] });
    rbacApiMocks.replaceRolePermissions.mockResolvedValue(null);

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    updatePermissionSelection(wrapper, []);
    await flushPromises();
    expect(wrapper.get('[data-testid="permission-drawer-save"]').attributes('disabled')).toBeUndefined();

    await wrapper.get('[data-testid="permission-drawer-save"]').trigger('click');
    await flushPromises();

    expect(rbacApiMocks.replaceRolePermissions).toHaveBeenCalledWith(2, {
      permission_ids: [],
    });
  });

  it('keeps replace mode submit disabled when the selection is unchanged', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="permission-drawer-save"]').attributes('disabled')).toBeDefined();
  });

  it('disables the drawer default footer for permission assignment', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="drawer"]').attributes('data-footer')).toBe('false');
  });

  it('prompts before closing the permission drawer with unsaved changes', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    updatePermissionSelection(wrapper, [1, 2]);
    await flushPromises();

    await wrapper.get('[data-testid="permission-drawer-cancel"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="discard-permission-confirm"]').attributes('data-header')).toBe(
      'rbac.roleList.permissionDialog.unsavedChangesTitle',
    );
    expect(wrapper.get('[data-testid="discard-permission-confirm"]').attributes('data-body')).toBe(
      'rbac.roleList.permissionDialog.unsavedChangesConfirm',
    );
    expect(wrapper.find('[data-testid="permission-drawer"]').exists()).toBe(true);
  });

  it('keeps the permission drawer state when continuing after the discard prompt', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    updatePermissionSelection(wrapper, [1, 2]);
    await flushPromises();
    await wrapper.get('input[placeholder="rbac.roleList.permissionDialog.searchPlaceholder"]').setValue('Localized');
    await setPermissionMutationMode(wrapper, 'replace');
    await wrapper.get('[data-testid="permission-drawer-cancel"]').trigger('click');
    await flushPromises();

    await wrapper.get('[data-testid="discard-confirm-cancel"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="discard-permission-confirm"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="permission-drawer"]').exists()).toBe(true);
    expect(selectedPermissionIds(wrapper)).toEqual([1, 2]);
    expect(
      (wrapper.get('input[placeholder="rbac.roleList.permissionDialog.searchPlaceholder"]').element as HTMLInputElement)
        .value,
    ).toBe('Localized');
    expect(rbacApiMocks.getRolePermissionBindings).toHaveBeenCalledTimes(1);
  });

  it('keeps the permission drawer open when overlay close is canceled from the discard prompt', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    updatePermissionSelection(wrapper, [1, 2]);
    await flushPromises();

    expect(wrapper.get('[data-testid="drawer"]').attributes('data-close-on-overlay')).toBe('false');
    expect(wrapper.get('[data-testid="drawer"]').attributes('data-close-on-esc')).toBe('false');
    await wrapper.get('[data-testid="drawer-overlay"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="permission-drawer"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="discard-permission-confirm"]').exists()).toBe(true);

    await wrapper.get('[data-testid="discard-confirm-cancel"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="permission-drawer"]').exists()).toBe(true);
    expect(selectedPermissionIds(wrapper)).toEqual([1, 2]);
    expect(rbacApiMocks.getRolePermissionBindings).toHaveBeenCalledTimes(1);
  });

  it('discards unsaved permission changes only after confirming discard', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings
      .mockResolvedValueOnce({ permission_ids: [1] })
      .mockResolvedValueOnce({ permission_ids: [1] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    updatePermissionSelection(wrapper, [1, 2]);
    await flushPromises();
    await wrapper.get('[data-testid="permission-drawer-cancel"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="discard-confirm-confirm"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="permission-drawer"]').exists()).toBe(false);

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();

    expect(rbacApiMocks.getRolePermissionBindings).toHaveBeenCalledTimes(2);
    expect(selectedPermissionIds(wrapper)).toEqual([1]);
  });

  it('submits only newly selected permissions in add mode', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1] });
    rbacApiMocks.addRolePermissions.mockResolvedValue(null);

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    await setPermissionMutationMode(wrapper, 'add');
    await flushPromises();
    updatePermissionSelection(wrapper, [2]);
    await flushPromises();
    await wrapper.get('[data-testid="permission-drawer-save"]').trigger('click');
    await flushPromises();

    expect(rbacApiMocks.addRolePermissions).toHaveBeenCalledWith(2, {
      permission_ids: [2],
    });
    expect(rbacApiMocks.replaceRolePermissions).not.toHaveBeenCalled();
    expect(rbacApiMocks.removeRolePermissions).not.toHaveBeenCalled();
  });

  it('submits only removed permissions in remove mode', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1, 2] });
    rbacApiMocks.removeRolePermissions.mockResolvedValue(null);

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    await setPermissionMutationMode(wrapper, 'remove');
    await flushPromises();
    updatePermissionSelection(wrapper, [1]);
    await flushPromises();
    await wrapper.get('[data-testid="permission-drawer-save"]').trigger('click');
    await flushPromises();

    expect(rbacApiMocks.removeRolePermissions).toHaveBeenCalledWith(2, {
      permission_ids: [1],
    });
    expect(rbacApiMocks.replaceRolePermissions).not.toHaveBeenCalled();
    expect(rbacApiMocks.addRolePermissions).not.toHaveBeenCalled();
  });

  it('resets to explicit removal selection when switching to remove mode', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1, 2] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    await setPermissionMutationMode(wrapper, 'remove');
    await flushPromises();

    expect(wrapper.get('[data-testid="permission-drawer-save"]').attributes('disabled')).toBeDefined();

    updatePermissionSelection(wrapper, [2]);
    await flushPromises();

    expect(wrapper.get('[data-testid="permission-drawer-save"]').attributes('disabled')).toBeUndefined();
  });

  it('renders localized permission copy inside the assignment drawer', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('Read Permissions Localized');
    expect(wrapper.text()).toContain('Localized permission description');
    expect(wrapper.text()).toContain('Read Audit Localized');
    expect(wrapper.text()).toContain('Localized audit description');
    expect(wrapper.text()).not.toContain('Permission Read');
    expect(wrapper.text()).not.toContain('Read permissions');
    expect(wrapper.text()).not.toContain('Audit Read');
    expect(wrapper.text()).not.toContain('Read audit logs');
  });

  it('prefers backend permission locale keys when they are present', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue({
      items: [
        {
          id: 9,
          code: 'future.read',
          display: 'Raw Permission Name',
          display_key: 'rbac.permissionCatalog.futureRead.display',
          description: 'Raw permission description',
          description_key: 'rbac.permissionCatalog.futureRead.description',
          category: 'future',
          role_binding_count: 0,
        },
      ],
    });
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('Future Read Localized');
    expect(wrapper.text()).toContain('Future description localized');
    expect(wrapper.text()).not.toContain('Raw Permission Name');
    expect(wrapper.text()).not.toContain('Raw permission description');
  });

  it('opens built-in admin permissions in readonly mode', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createBuiltinAdminRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1, 2] });

    const wrapper = mountRolePage();
    await flushPromises();

    expect(wrapper.find('[data-testid="role-assign-permissions"]').exists()).toBe(false);
    expect(wrapper.get('[data-testid="role-view-permissions"]').text()).toContain('rbac.roleList.viewPermissions');

    await wrapper.get('[data-testid="role-view-permissions"]').trigger('click');
    await flushPromises();

    expect(rbacApiMocks.getRolePermissionBindings).toHaveBeenCalledWith(1);
    expect(wrapper.find('[data-testid="permission-drawer"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="permission-readonly-protection"]').exists()).toBe(true);
    expect(wrapper.get('[data-testid="permission-checkbox-group"]').attributes('data-disabled')).toBe('true');
    expect(wrapper.find('[data-testid="permission-drawer-save"]').exists()).toBe(false);
    updatePermissionSelection(wrapper, [1, 2, 3]);
    await flushPromises();
    expect(rbacApiMocks.replaceRolePermissions).not.toHaveBeenCalled();

    await wrapper.get('[data-testid="drawer-overlay"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="permission-drawer"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="discard-permission-confirm"]').exists()).toBe(false);
  });

  it('matches permission search against localized copy', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [1] });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    await wrapper.get('input[placeholder="rbac.roleList.permissionDialog.searchPlaceholder"]').setValue('Localized');
    await flushPromises();

    expect(wrapper.text()).toContain('Read Permissions Localized');
    expect(wrapper.text()).toContain('Update Roles Localized');
  });

  it('shows overview and lifecycle guidance in the detail drawer for role delete semantics', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRoleDetail.mockResolvedValue({
      id: 2,
      name: 'editor',
      display: 'Editor',
      description: 'Editor role',
      builtin: false,
      status: 'enabled',
      updated_at: '2026-05-18T00:00:00Z',
      created_at: '2026-05-17T00:00:00Z',
      permission_count: 2,
      user_count: 1,
    });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-detail"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="role-overview"]').text()).toContain('Editor');
    expect(wrapper.get('[data-testid="role-overview"]').text()).toContain('editor');
    expect(wrapper.get('[data-testid="role-overview"]').text()).toContain('rbac.roleList.form.type.custom');
    expect(wrapper.get('[data-testid="role-overview"]').text()).toContain('rbac.roleList.lifecycle.statusEnabled');
    expect(wrapper.find('[data-testid="role-form"]').exists()).toBe(false);
    expect(wrapper.get('[data-testid="role-lifecycle-summary"]').text()).toContain(
      'rbac.roleList.lifecycle.deleteNeedsDisable',
    );
  });

  it('renders built-in admin details as overview, one rule alert, and readonly content', async () => {
    permissionState.grantedCodes = [
      RBAC_PERMISSION_CODE.PERMISSION_READ,
      RBAC_PERMISSION_CODE.ROLE_DELETE,
      RBAC_PERMISSION_CODE.ROLE_STATUS_UPDATE,
    ];
    rbacApiMocks.getRoles.mockResolvedValue(createBuiltinAdminRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRoleDetail.mockResolvedValue({
      id: 1,
      name: 'admin',
      display: 'Administrator',
      description: 'Builtin administrator',
      builtin: true,
      status: 'enabled',
      updated_at: '2026-05-18T00:00:00Z',
      created_at: '2026-05-17T00:00:00Z',
      permission_count: 3,
      user_count: 1,
    });

    const wrapper = mountRolePage();
    await flushPromises();

    wrapper.getComponent(dropdownStub).vm.$emit('click', { value: 'detail' });
    await flushPromises();

    expect(wrapper.get('[data-testid="role-overview"]').text()).toContain('Administrator');
    expect(wrapper.get('[data-testid="role-overview"]').text()).toContain('admin');
    expect(wrapper.get('[data-testid="role-overview"]').text()).toContain('rbac.roleList.form.type.system');
    expect(wrapper.get('[data-testid="role-overview"]').text()).toContain('rbac.roleList.lifecycle.statusEnabled');
    expect(wrapper.findAll('[data-testid="role-system-rules"]')).toHaveLength(1);
    expect(wrapper.get('[data-testid="role-system-rules"]').text()).toContain(
      'rbac.roleList.form.systemProtectionBody',
    );
    expect(wrapper.get('[data-testid="role-system-rules"]').text()).toContain(
      'rbac.roleList.form.systemProtectionNormal',
    );
    expect(wrapper.get('[data-testid="role-system-rules"]').text()).toContain(
      'rbac.roleList.form.systemProtectionCopyHint',
    );
    expect(wrapper.find('[data-testid="role-form"]').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('rbac.roleList.form.builtinNotice');
    expect(wrapper.text()).not.toContain('rbac.roleList.moreActions.delete');
    expect(wrapper.text()).not.toContain('rbac.roleList.edit');
    expect(wrapper.find('[data-testid="role-drawer-delete"]').exists()).toBe(false);
  });

  it('copies a built-in role as a custom role with the source permission set', async () => {
    permissionState.grantedCodes = [
      RBAC_PERMISSION_CODE.PERMISSION_READ,
      RBAC_PERMISSION_CODE.ROLE_CREATE,
      RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN,
    ];
    rbacApiMocks.getRoles.mockResolvedValue(createBuiltinAdminRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [3, 1] });
    rbacApiMocks.createRole.mockResolvedValue({
      id: 9,
      name: 'custom-admin',
      display: 'Custom Admin',
      description: 'Builtin administrator',
      builtin: false,
      status: 'enabled',
      updated_at: '2026-05-19T00:00:00Z',
      permission_count: 0,
      user_count: 0,
    });
    rbacApiMocks.replaceRolePermissions.mockResolvedValue(null);

    const wrapper = mountRolePage();
    await flushPromises();

    wrapper.getComponent(dropdownStub).vm.$emit('click', { value: 'copy-role' });
    await flushPromises();

    expect(rbacApiMocks.getRolePermissionBindings).toHaveBeenCalledWith(1);
    expect(
      (wrapper.get('input[placeholder="rbac.roleList.form.namePlaceholder"]').element as HTMLInputElement).value,
    ).toBe('');
    expect(
      (wrapper.get('input[placeholder="rbac.roleList.form.displayPlaceholder"]').element as HTMLInputElement).value,
    ).toBe('rbac.roleList.copyDisplayTemplate');

    await wrapper.get('input[placeholder="rbac.roleList.form.namePlaceholder"]').setValue(' custom-admin ');
    await wrapper.get('[data-testid="role-form"]').trigger('submit');
    await flushPromises();

    expect(rbacApiMocks.createRole).toHaveBeenCalledWith({
      name: 'custom-admin',
      display: 'rbac.roleList.copyDisplayTemplate',
      description: 'Builtin administrator',
    });
    expect(rbacApiMocks.replaceRolePermissions).toHaveBeenCalledWith(9, {
      permission_ids: [1, 3],
    });
    expect(messageMocks.success).toHaveBeenCalledWith('rbac.roleList.copySuccess');
  });

  it('blocks delete when the role is still enabled', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.ROLE_DELETE];
    rbacApiMocks.getRoles.mockResolvedValue({
      items: [
        {
          ...createRoleListResponse().items[0],
          permission_count: 0,
          user_count: 0,
          status: 'enabled',
        },
      ],
    });
    rbacApiMocks.getPermissions.mockResolvedValue({ items: [] });

    const wrapper = mountRolePage();
    await flushPromises();

    wrapper.getComponent(dropdownStub).vm.$emit('click', { value: 'delete' });
    await flushPromises();

    expect(rbacApiMocks.deleteRole).not.toHaveBeenCalled();
    expect(messageMocks.warning).toHaveBeenCalledWith('rbac.roleList.lifecycle.deleteNeedsDisable');
  });

  it('hides more actions that are missing permission instead of rendering disabled permission-only entries', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.ROLE_UPDATE];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue({ items: [] });

    const wrapper = mountRolePage();
    await flushPromises();

    expect(wrapper.find('[data-testid="dropdown"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="role-edit"]').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('rbac.roleList.moreActions.delete');
  });

  it('blocks delete when the disabled role still has bindings', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.ROLE_DELETE];
    rbacApiMocks.getRoles.mockResolvedValue({
      items: [
        {
          ...createRoleListResponse().items[0],
          permission_count: 1,
          user_count: 0,
          status: 'disabled',
        },
      ],
    });
    rbacApiMocks.getPermissions.mockResolvedValue({ items: [] });

    const wrapper = mountRolePage();
    await flushPromises();

    wrapper.getComponent(dropdownStub).vm.$emit('click', { value: 'delete' });
    await flushPromises();

    expect(rbacApiMocks.deleteRole).not.toHaveBeenCalled();
    expect(messageMocks.warning).toHaveBeenCalledWith('rbac.roleList.lifecycle.deleteNeedsBindingsCleared');
  });

  it('keeps permission assignment errors local when the backend rejects permission_ids', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [2, 1, 2] });
    rbacApiMocks.replaceRolePermissions.mockRejectedValue({
      isApiRequestError: true,
      status: 400,
      code: 'COMMON_INVALID_ARGUMENT',
      message: '权限列表包含无效条目',
      messageKey: '',
      responseData: {
        data: {
          field: 'permission_ids',
        },
      },
    });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    updatePermissionSelection(wrapper, [2]);
    await flushPromises();
    await wrapper.get('[data-testid="permission-drawer-save"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('权限列表包含无效条目');
    expect(messageMocks.error).not.toHaveBeenCalledWith('rbac.roleList.assignFailed');
  });

  it('keeps role-not-found assignment failures in the permission drawer feedback surface', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.PERMISSION_READ, RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN];
    rbacApiMocks.getRoles.mockResolvedValue(createRoleListResponse());
    rbacApiMocks.getPermissions.mockResolvedValue(createPermissionListResponse());
    rbacApiMocks.getRolePermissionBindings.mockResolvedValue({ permission_ids: [2, 1, 2] });
    rbacApiMocks.replaceRolePermissions.mockRejectedValue({
      isApiRequestError: true,
      status: 404,
      code: 'ROLE_NOT_FOUND',
      message: '角色不存在',
      messageKey: 'role.not_found',
      responseData: {},
    });

    const wrapper = mountRolePage();
    await flushPromises();

    await wrapper.get('[data-testid="role-assign-permissions"]').trigger('click');
    await flushPromises();
    updatePermissionSelection(wrapper, [2]);
    await flushPromises();
    await wrapper.get('[data-testid="permission-drawer-save"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('角色不存在');
    expect(messageMocks.error).not.toHaveBeenCalledWith('rbac.roleList.assignFailed');
  });

  it('renders the table empty state, clears filters, and opens the create drawer from the empty action', async () => {
    permissionState.grantedCodes = [RBAC_PERMISSION_CODE.ROLE_CREATE];
    rbacApiMocks.getRoles.mockResolvedValue({ items: [] });
    rbacApiMocks.getPermissions.mockResolvedValue({ items: [] });

    const wrapper = mountRolePage();
    await flushPromises();

    expect(wrapper.text()).toContain('rbac.roleList.emptyTitle');
    expect(wrapper.text()).toContain('rbac.roleList.emptyDescription');
    expect(wrapper.find('[data-testid="role-empty-clear-filters"]').exists()).toBe(false);

    await wrapper.get('input[placeholder="rbac.roleList.toolbar.searchPlaceholder"]').setValue('editor');
    await flushPromises();

    expect(wrapper.find('[data-testid="role-empty-clear-filters"]').exists()).toBe(true);
    await wrapper.get('[data-testid="role-empty-clear-filters"]').trigger('click');
    await flushPromises();

    expect(
      (wrapper.get('input[placeholder="rbac.roleList.toolbar.searchPlaceholder"]').element as HTMLInputElement).value,
    ).toBe('');

    await wrapper.get('[data-testid="role-empty-create"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="drawer"]').exists()).toBe(true);
  });
});
