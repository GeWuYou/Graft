<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <t-form
    ref="form"
    class="item-container"
    :class="[`login-${type}`]"
    :data="formData"
    :rules="FORM_RULES"
    label-width="0"
    @submit="onSubmit"
  >
    <template v-if="type === 'password'">
      <t-form-item name="account">
        <t-input v-model="formData.account" size="large" :placeholder="`${t('app.auth.login.input.account')}：admin`">
          <template #prefix-icon>
            <t-icon name="user" />
          </template>
        </t-input>
      </t-form-item>

      <t-form-item name="password">
        <t-input
          v-model="formData.password"
          size="large"
          :type="showPsw ? 'text' : 'password'"
          clearable
          :placeholder="`${t('app.auth.login.input.password')}：admin`"
        >
          <template #prefix-icon>
            <t-icon name="lock-on" />
          </template>
          <template #suffix-icon>
            <t-icon :name="showPsw ? 'browse' : 'browse-off'" @click="showPsw = !showPsw" />
          </template>
        </t-input>
      </t-form-item>

      <div class="check-container remember-pwd">
        <t-checkbox>{{ t('app.auth.login.remember') }}</t-checkbox>
        <span class="tip">{{ t('app.auth.login.forget') }}</span>
      </div>
    </template>

    <template v-else-if="type === 'qrcode'">
      <div class="tip-container">
        <span class="tip">{{ t('app.auth.login.wechatLogin') }}</span>
        <span class="refresh">{{ t('app.auth.login.refresh') }} <t-icon name="refresh" /> </span>
      </div>
      <t-qrcode value="tdesign" :size="160" level="H" />
    </template>

    <template v-else>
      <t-form-item name="phone">
        <t-input v-model="formData.phone" size="large" :placeholder="t('app.auth.login.input.phone')">
          <template #prefix-icon>
            <t-icon name="mobile" />
          </template>
        </t-input>
      </t-form-item>

      <t-form-item class="verification-code" name="verifyCode">
        <t-input v-model="formData.verifyCode" size="large" :placeholder="t('app.auth.login.input.verification')" />
        <t-button size="large" variant="outline" :disabled="countDown > 0" @click="sendCode">
          {{
            countDown === 0 ? t('app.auth.login.sendVerification') : t('app.auth.login.countdown', { count: countDown })
          }}
        </t-button>
      </t-form-item>
    </template>

    <t-form-item v-if="type !== 'qrcode'" class="btn-container">
      <t-button block size="large" type="submit"> {{ t('app.auth.login.signIn') }} </t-button>
    </t-form-item>

    <div class="switch-container">
      <span v-if="type !== 'password'" class="tip" @click="switchType('password')">{{
        t('app.auth.login.accountLogin')
      }}</span>
      <span v-if="type !== 'qrcode'" class="tip" @click="switchType('qrcode')">{{
        t('app.auth.login.wechatLogin')
      }}</span>
      <span v-if="type !== 'phone'" class="tip" @click="switchType('phone')">{{ t('app.auth.login.phoneLogin') }}</span>
    </div>
  </t-form>
</template>
<script setup lang="ts">
import type { FormInstanceFunctions, FormRule, SubmitContext } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { API_CODE } from '@/contracts/api/codes';
import { t } from '@/locales';
import { AUTH_ROUTE_PATH } from '@/modules/auth/contract/routes';
import { useAuthSessionStore } from '@/modules/auth/store';
import { resolveLocalizedErrorMessage } from '@/modules/shared/localized-api-error';
import { useCounter } from '@/shared/composables';
import { isApiRequestError } from '@/utils/request';

const userStore = useAuthSessionStore();

const INITIAL_DATA = {
  phone: '',
  account: '',
  password: '',
  verifyCode: '',
  checked: false,
};

const FORM_RULES = computed<Record<string, FormRule[]>>(() => ({
  phone: [{ required: true, message: t('app.auth.login.required.phone'), type: 'error' }],
  account: [{ required: true, message: t('app.auth.login.required.account'), type: 'error' }],
  password: [{ required: true, message: t('app.auth.login.required.password'), type: 'error' }],
  verifyCode: [{ required: true, message: t('app.auth.login.required.verification'), type: 'error' }],
}));

const type = ref('password');

const form = ref<FormInstanceFunctions>();
const formData = ref({ ...INITIAL_DATA });
const showPsw = ref(false);

const [countDown, handleCounter] = useCounter();

const switchType = (value: string) => {
  type.value = value;
};

const router = useRouter();
const route = useRoute();

const sendCode = () => {
  form.value?.validate({ fields: ['phone'] }).then((result) => {
    if (result === true) {
      handleCounter();
    }
  });
};

const onSubmit = async (ctx: SubmitContext) => {
  if (ctx.validateResult === true) {
    try {
      await userStore.login(formData.value);

      MessagePlugin.success(t('app.auth.login.loginSuccess'));
      const redirectQuery = route.query.redirect;
      const redirect = typeof redirectQuery === 'string' ? redirectQuery : '';
      const redirectUrl = (() => {
        if (!redirect) {
          return '/';
        }

        try {
          const decoded = decodeURIComponent(redirect);
          return decoded.startsWith('/') ? decoded : '/';
        } catch {
          return '/';
        }
      })();
      const nextPath = userStore.mustChangePassword ? AUTH_ROUTE_PATH.RESTRICTED_SESSION : redirectUrl;
      router.push(nextPath);
    } catch (error) {
      if (userStore.token && userStore.mustChangePassword) {
        router.push(AUTH_ROUTE_PATH.RESTRICTED_SESSION);
        return;
      }

      if (isApiRequestError(error) && error.status === 400 && error.code === API_CODE.AUTH_INVALID_CREDENTIALS) {
        MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('app.auth.login.loginFailed')));
        return;
      }

      MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('app.auth.login.loginFailed')));
    }
  }
};
</script>
<style lang="less" scoped>
@import '../index.less';
</style>
