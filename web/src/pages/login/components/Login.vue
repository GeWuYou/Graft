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
        <t-input v-model="formData.account" size="large" :placeholder="`${t('pages.login.input.account')}：admin`">
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
          :placeholder="`${t('pages.login.input.password')}：admin`"
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
        <t-checkbox>{{ t('pages.login.remember') }}</t-checkbox>
        <span class="tip">{{ t('pages.login.forget') }}</span>
      </div>
    </template>

    <!-- 扫码登录 -->
    <template v-else-if="type === 'qrcode'">
      <div class="tip-container">
        <span class="tip">{{ t('pages.login.wechatLogin') }}</span>
        <span class="refresh">{{ t('pages.login.refresh') }} <t-icon name="refresh" /> </span>
      </div>
      <t-qrcode value="tdesign" :size="160" level="H" />
    </template>

    <!-- 手机号登录 -->
    <template v-else>
      <t-form-item name="phone">
        <t-input v-model="formData.phone" size="large" :placeholder="t('pages.login.input.phone')">
          <template #prefix-icon>
            <t-icon name="mobile" />
          </template>
        </t-input>
      </t-form-item>

      <t-form-item class="verification-code" name="verifyCode">
        <t-input v-model="formData.verifyCode" size="large" :placeholder="t('pages.login.input.verification')" />
        <t-button size="large" variant="outline" :disabled="countDown > 0" @click="sendCode">
          {{ countDown === 0 ? t('pages.login.sendVerification') : t('pages.login.countdown', { count: countDown }) }}
        </t-button>
      </t-form-item>
    </template>

    <t-form-item v-if="type !== 'qrcode'" class="btn-container">
      <t-button block size="large" type="submit"> {{ t('pages.login.signIn') }} </t-button>
    </t-form-item>

    <div class="switch-container">
      <span v-if="type !== 'password'" class="tip" @click="switchType('password')">{{
        t('pages.login.accountLogin')
      }}</span>
      <span v-if="type !== 'qrcode'" class="tip" @click="switchType('qrcode')">{{ t('pages.login.wechatLogin') }}</span>
      <span v-if="type !== 'phone'" class="tip" @click="switchType('phone')">{{ t('pages.login.phoneLogin') }}</span>
    </div>
  </t-form>
</template>
<script setup lang="ts">
import type { FormInstanceFunctions, FormRule, SubmitContext } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { API_CODE } from '@/api/model/authModel';
import { AUTH_ROUTE_PATH } from '@/contracts/auth/routes';
import { useCounter } from '@/hooks';
import { t } from '@/locales';
import { useUserStore } from '@/store';
import { isApiRequestError } from '@/utils/request';

const userStore = useUserStore();

const INITIAL_DATA = {
  phone: '',
  account: '',
  password: '',
  verifyCode: '',
  checked: false,
};

const FORM_RULES = computed<Record<string, FormRule[]>>(() => ({
  phone: [{ required: true, message: t('pages.login.required.phone'), type: 'error' }],
  account: [{ required: true, message: t('pages.login.required.account'), type: 'error' }],
  password: [{ required: true, message: t('pages.login.required.password'), type: 'error' }],
  verifyCode: [{ required: true, message: t('pages.login.required.verification'), type: 'error' }],
}));

const type = ref('password');

const form = ref<FormInstanceFunctions>();
const formData = ref({ ...INITIAL_DATA });
const showPsw = ref(false);

const [countDown, handleCounter] = useCounter();

const switchType = (val: string) => {
  type.value = val;
};

const router = useRouter();
const route = useRoute();

/**
 * 发送验证码
 */
const sendCode = () => {
  form.value?.validate({ fields: ['phone'] }).then((e) => {
    if (e === true) {
      handleCounter();
    }
  });
};

const onSubmit = async (ctx: SubmitContext) => {
  if (ctx.validateResult === true) {
    try {
      await userStore.login(formData.value);

      MessagePlugin.success(t('pages.login.loginSuccess'));
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
    } catch (e) {
      if (userStore.token && userStore.mustChangePassword) {
        router.push(AUTH_ROUTE_PATH.RESTRICTED_SESSION);
        return;
      }

      if (isApiRequestError(e) && e.status === 400 && e.code === API_CODE.AUTH_INVALID_CREDENTIALS) {
        MessagePlugin.error(e.message);
        return;
      }

      MessagePlugin.error(e instanceof Error ? e.message : String(e));
    }
  }
};
</script>
<style lang="less" scoped>
@import '../index.less';
</style>
