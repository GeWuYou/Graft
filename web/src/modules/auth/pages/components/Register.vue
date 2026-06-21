<template>
  <t-form
    ref="form"
    class="item-container"
    :class="[`register-${type}`]"
    :data="formData"
    :rules="FORM_RULES"
    label-width="0"
    @submit="onSubmit"
  >
    <template v-if="type === 'phone'">
      <t-form-item name="phone">
        <t-input
          v-model="formData.phone"
          :maxlength="11"
          size="large"
          :placeholder="t('app.auth.register.phonePlaceholder')"
        >
          <template #prefix-icon>
            <t-icon name="user" />
          </template>
        </t-input>
      </t-form-item>
    </template>

    <template v-if="type === 'email'">
      <t-form-item name="email">
        <t-input
          v-model="formData.email"
          type="text"
          size="large"
          :placeholder="t('app.auth.register.emailPlaceholder')"
        >
          <template #prefix-icon>
            <t-icon name="mail" />
          </template>
        </t-input>
      </t-form-item>
    </template>

    <t-form-item name="password">
      <t-input
        v-model="formData.password"
        size="large"
        :type="showPsw ? 'text' : 'password'"
        clearable
        :placeholder="t('app.auth.register.passwordPlaceholder')"
      >
        <template #prefix-icon>
          <t-icon name="lock-on" />
        </template>
        <template #suffix-icon>
          <t-icon :name="showPsw ? 'browse' : 'browse-off'" @click="showPsw = !showPsw" />
        </template>
      </t-input>
    </t-form-item>

    <template v-if="type === 'phone'">
      <t-form-item class="verification-code" name="verifyCode">
        <t-input
          v-model="formData.verifyCode"
          size="large"
          :placeholder="t('app.auth.register.verifyCodePlaceholder')"
        />
        <t-button variant="outline" :disabled="countDown > 0" @click="handleCounter">
          {{
            countDown === 0
              ? t('app.auth.register.sendVerifyCode')
              : t('app.auth.register.resendCountdown', { count: countDown })
          }}
        </t-button>
      </t-form-item>
    </template>

    <t-form-item class="check-container" name="checked">
      <t-checkbox v-model="formData.checked">{{ t('app.auth.register.agreeTerms') }} </t-checkbox>
      <span>{{ t('app.auth.register.serviceTerms') }}</span>
      {{ t(MESSAGE_KEY.COMMON_CONJUNCTION) }}
      <span>{{ t('app.auth.register.privacyStatement') }}</span>
    </t-form-item>

    <t-form-item>
      <t-button block size="large" type="submit"> {{ t('app.auth.register.registerBtn') }} </t-button>
    </t-form-item>

    <div class="switch-container">
      <span class="tip" @click="switchType(type === 'phone' ? 'email' : 'phone')">
        {{ type === 'phone' ? t('app.auth.register.useEmailRegister') : t('app.auth.register.usePhoneRegister') }}
      </span>
    </div>
  </t-form>
</template>
<script setup lang="ts">
import type { FormRule, SubmitContext } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, ref } from 'vue';

import { MESSAGE_KEY } from '@/contracts/api/messages';
import { t } from '@/locales';
import { useCounter } from '@/shared/composables';

const emit = defineEmits(['register-success']);

const INITIAL_DATA = {
  phone: '',
  email: '',
  password: '',
  verifyCode: '',
  checked: false,
};

const FORM_RULES = computed<Record<string, FormRule[]>>(() => ({
  phone: [{ required: true, message: t('app.auth.register.validation.phone'), type: 'error' }],
  email: [
    { required: true, message: t('app.auth.register.validation.email'), type: 'error' },
    { email: true, message: t('app.auth.register.validation.emailFormat'), type: 'warning' },
  ],
  password: [{ required: true, message: t('app.auth.register.validation.password'), type: 'error' }],
  verifyCode: [{ required: true, message: t('app.auth.register.validation.verifyCode'), type: 'error' }],
}));

const type = ref('phone');

const form = ref();
const formData = ref({ ...INITIAL_DATA });

const showPsw = ref(false);

const [countDown, handleCounter] = useCounter();

const onSubmit = (ctx: SubmitContext) => {
  if (ctx.validateResult === true) {
    if (!formData.value.checked) {
      MessagePlugin.error(t('app.auth.register.validation.agreeTerms'));
      return;
    }
    MessagePlugin.success(t('app.auth.register.messages.registerSuccess'));
    emit('register-success');
  }
};

const switchType = (value: string) => {
  form.value.reset();
  type.value = value;
};
</script>
<style lang="less" scoped>
@import '../index.less';
</style>
