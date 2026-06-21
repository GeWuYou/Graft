<template>
  <t-dialog
    :visible="visible"
    :header="t('app.auth.login.forcePasswordChange.title')"
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
          {{ t('app.auth.login.forcePasswordChange.description') }}
        </p>
        <p class="force-password-change-dialog__hint">
          {{ t('app.auth.login.forcePasswordChange.policyHint') }}
        </p>

        <t-form ref="formRef" :data="formData" :rules="formRules" label-align="top" @submit="handleSubmit">
          <t-form-item :label="t('app.auth.login.forcePasswordChange.newPassword')" name="newPassword">
            <t-input
              v-model="formData.newPassword"
              type="password"
              autocomplete="new-password"
              :placeholder="t('app.auth.login.forcePasswordChange.newPasswordPlaceholder')"
            />
          </t-form-item>

          <t-form-item :label="t('app.auth.login.forcePasswordChange.confirmPassword')" name="confirmPassword">
            <t-input
              v-model="formData.confirmPassword"
              type="password"
              autocomplete="new-password"
              :placeholder="t('app.auth.login.forcePasswordChange.confirmPasswordPlaceholder')"
            />
          </t-form-item>

          <div class="force-password-change-dialog__actions">
            <t-button block theme="primary" :loading="submitting" type="submit">
              {{ t('app.auth.login.forcePasswordChange.submit') }}
            </t-button>
          </div>
        </t-form>
      </div>
    </template>
  </t-dialog>
</template>
<script setup lang="ts">
import type { FormInstanceFunctions, FormRule, SubmitContext } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';

import { API_CODE } from '@/contracts/api/codes';
import { t } from '@/locales';
import { completeRestrictedPasswordChange } from '@/modules/auth/runtime/restricted-session';
import { useAuthSessionStore } from '@/modules/auth/store';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { usePermissionStore } from '@/store';
import { isApiRequestError } from '@/utils/request';

type ForcePasswordChangeForm = {
  newPassword: string;
  confirmPassword: string;
};

const PASSWORD_POLICY = /^(?=.*[A-Za-z])(?=.*\d).{12,}$/;

const INITIAL_FORM_DATA: ForcePasswordChangeForm = {
  newPassword: '',
  confirmPassword: '',
};

const router = useRouter();
const permissionStore = usePermissionStore();
const userStore = useAuthSessionStore();
const formRef = ref<FormInstanceFunctions>();
const submitting = ref(false);
const formData = ref<ForcePasswordChangeForm>({ ...INITIAL_FORM_DATA });

// 强制改密状态只以后端 login/bootstrap 返回值为准；前端不根据用户名或默认密码推断。
const visible = computed(() => Boolean(userStore.token && userStore.bootstrapLoaded && userStore.mustChangePassword));

const formRules = computed<Record<keyof ForcePasswordChangeForm, FormRule[]>>(() => ({
  newPassword: [
    { required: true, message: t('app.auth.login.forcePasswordChange.required.newPassword'), type: 'error' },
  ],
  confirmPassword: [
    { required: true, message: t('app.auth.login.forcePasswordChange.required.confirmPassword'), type: 'error' },
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
    code === API_CODE.COMMON_INVALID_ARGUMENT
  );
}

function validatePasswordPolicy() {
  if (formData.value.newPassword !== formData.value.confirmPassword) {
    return t('app.auth.login.forcePasswordChange.errors.confirmMismatch');
  }

  if (!PASSWORD_POLICY.test(formData.value.newPassword)) {
    return t('app.auth.login.forcePasswordChange.errors.policyViolation');
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
    await completeRestrictedPasswordChange({
      newPassword: formData.value.newPassword,
      bootstrap: (force) => userStore.bootstrap(force),
      buildAsyncRoutes: () => permissionStore.buildAsyncRoutes(),
      consumePendingRestrictedRedirect: (fallbackPath) => userStore.consumePendingRestrictedRedirect(fallbackPath),
      replace: (path) => router.replace(path),
    });
    resetForm();
    formRef.value?.clearValidate();
    MessagePlugin.success(t('app.auth.login.forcePasswordChange.success'));
  } catch (error) {
    if (isApiRequestError(error) && isPasswordChangeApiCode(error.code)) {
      MessagePlugin.warning(
        resolveLocalizedErrorMessage(t, error, t('app.auth.login.forcePasswordChange.errors.policyViolation')),
      );
      return;
    }

    MessagePlugin.error(
      resolveLocalizedErrorMessage(t, error, t('app.auth.login.forcePasswordChange.errors.submitFailed')),
    );
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
    margin: 0 0 var(--graft-density-gap-16);
  }

  &__actions {
    margin-top: var(--graft-density-gap-8);
  }
}
</style>
