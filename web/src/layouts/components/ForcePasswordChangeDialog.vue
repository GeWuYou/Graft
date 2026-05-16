<template>
  <t-dialog
    :visible="visible"
    :header="t('pages.login.forcePasswordChange.title')"
    :close-btn="false"
    :close-on-esc-keydown="false"
    :close-on-overlay-click="false"
    :confirm-btn="null"
    :cancel-btn="null"
    :footer="false"
    :width="520"
    :destroy-on-close="false"
  >
    <template #body>
      <div class="force-password-change-dialog">
        <p class="force-password-change-dialog__description">
          {{ t('pages.login.forcePasswordChange.description') }}
        </p>
        <p class="force-password-change-dialog__hint">
          {{ t('pages.login.forcePasswordChange.policyHint') }}
        </p>

        <t-form ref="formRef" :data="formData" :rules="formRules" label-align="top" @submit="handleSubmit">
          <t-form-item :label="t('pages.login.forcePasswordChange.currentPassword')" name="currentPassword">
            <t-input
              v-model="formData.currentPassword"
              type="password"
              autocomplete="current-password"
              :placeholder="t('pages.login.forcePasswordChange.currentPasswordPlaceholder')"
            />
          </t-form-item>

          <t-form-item :label="t('pages.login.forcePasswordChange.newPassword')" name="newPassword">
            <t-input
              v-model="formData.newPassword"
              type="password"
              autocomplete="new-password"
              :placeholder="t('pages.login.forcePasswordChange.newPasswordPlaceholder')"
            />
          </t-form-item>

          <t-form-item :label="t('pages.login.forcePasswordChange.confirmPassword')" name="confirmPassword">
            <t-input
              v-model="formData.confirmPassword"
              type="password"
              autocomplete="new-password"
              :placeholder="t('pages.login.forcePasswordChange.confirmPasswordPlaceholder')"
            />
          </t-form-item>

          <div class="force-password-change-dialog__actions">
            <t-button block theme="primary" :loading="submitting" type="submit">
              {{ t('pages.login.forcePasswordChange.submit') }}
            </t-button>
          </div>
        </t-form>
      </div>
    </template>
  </t-dialog>
</template>
<script setup lang="ts">
import type { FormInstanceFunctions, FormRule, SubmitContext } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, ref } from 'vue';

import { changePassword } from '@/api/auth';
import { API_CODE } from '@/api/model/authModel';
import { t } from '@/locales';
import { useUserStore } from '@/store';
import { isApiRequestError } from '@/utils/request';

type ForcePasswordChangeForm = {
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
};

const PASSWORD_POLICY = /^(?=.*[A-Za-z])(?=.*\d).{12,}$/;

const INITIAL_FORM_DATA: ForcePasswordChangeForm = {
  currentPassword: '',
  newPassword: '',
  confirmPassword: '',
};

const userStore = useUserStore();
const formRef = ref<FormInstanceFunctions>();
const submitting = ref(false);
const formData = ref<ForcePasswordChangeForm>({ ...INITIAL_FORM_DATA });

// 强制改密状态只以后端 login/bootstrap 返回值为准；前端不根据用户名或默认密码推断。
const visible = computed(() => Boolean(userStore.token && userStore.bootstrapLoaded && userStore.mustChangePassword));

const formRules = computed<Record<keyof ForcePasswordChangeForm, FormRule[]>>(() => ({
  currentPassword: [
    { required: true, message: t('pages.login.forcePasswordChange.required.currentPassword'), type: 'error' },
  ],
  newPassword: [{ required: true, message: t('pages.login.forcePasswordChange.required.newPassword'), type: 'error' }],
  confirmPassword: [
    { required: true, message: t('pages.login.forcePasswordChange.required.confirmPassword'), type: 'error' },
  ],
}));

const resetForm = () => {
  formData.value = { ...INITIAL_FORM_DATA };
};

function isPasswordChangeApiCode(code: string) {
  return (
    code === API_CODE.AUTH_CURRENT_PASSWORD_INVALID ||
    code === API_CODE.AUTH_PASSWORD_POLICY_VIOLATION ||
    code === API_CODE.AUTH_PASSWORD_REUSE_FORBIDDEN ||
    code === API_CODE.AUTH_PASSWORD_CHANGE_REQUIRED ||
    code === API_CODE.COMMON_INVALID_ARGUMENT
  );
}

function validatePasswordPolicy() {
  if (formData.value.newPassword !== formData.value.confirmPassword) {
    return t('pages.login.forcePasswordChange.errors.confirmMismatch');
  }

  if (formData.value.newPassword === formData.value.currentPassword) {
    return t('pages.login.forcePasswordChange.errors.sameAsCurrent');
  }

  if (!PASSWORD_POLICY.test(formData.value.newPassword)) {
    return t('pages.login.forcePasswordChange.errors.policyViolation');
  }

  return '';
}

const handleSubmit = async (ctx: SubmitContext) => {
  if (ctx.validateResult !== true || submitting.value) {
    return;
  }

  const validationError = validatePasswordPolicy();
  if (validationError) {
    MessagePlugin.warning(validationError);
    return;
  }

  submitting.value = true;
  try {
    await changePassword({
      current_password: formData.value.currentPassword,
      new_password: formData.value.newPassword,
    });
    await userStore.bootstrap(true);
    resetForm();
    formRef.value?.clearValidate();
    MessagePlugin.success(t('pages.login.forcePasswordChange.success'));
  } catch (error) {
    if (isApiRequestError(error) && isPasswordChangeApiCode(error.code)) {
      MessagePlugin.warning(error.message);
      return;
    }

    MessagePlugin.error(error instanceof Error ? error.message : String(error));
  } finally {
    submitting.value = false;
  }
};
</script>
<style lang="less" scoped>
.force-password-change-dialog {
  &__description,
  &__hint {
    color: var(--td-text-color-secondary);
    line-height: 22px;
    margin: 0 0 16px;
  }

  &__actions {
    margin-top: 8px;
  }
}
</style>
