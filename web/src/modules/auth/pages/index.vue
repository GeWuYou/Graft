<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="login-wrapper">
    <login-header />

    <div class="login-container">
      <div class="title-container">
        <h1 class="title margin-no">{{ t('app.auth.login.title') }}</h1>
        <h1 class="title">{{ t('common.appName') }}</h1>
        <div class="sub-title">
          <p class="tip">{{ type === 'register' ? t('app.auth.login.haveAccount') : t('app.auth.login.noAccount') }}</p>
          <button type="button" class="tip switch-link" @click="switchType(type === 'register' ? 'login' : 'register')">
            {{ type === 'register' ? t('app.auth.login.signIn') : t('app.auth.login.createAccount') }}
          </button>
        </div>
      </div>

      <login-panel v-if="type === 'login'" />
      <register-panel v-else @register-success="switchType('login')" />
    </div>

    <footer class="copyright">{{ t(MESSAGE_KEY.COMMON_COPYRIGHT) }}</footer>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue';

import { MESSAGE_KEY } from '@/contracts/api/messages';
import { t } from '@/locales';

import LoginHeader from './components/Header.vue';
import LoginPanel from './components/Login.vue';
import RegisterPanel from './components/Register.vue';

defineOptions({
  name: 'AuthPage',
});

const type = ref('login');

const switchType = (value: string) => {
  type.value = value;
};
</script>
<style lang="less" scoped>
@import './index.less';
</style>
