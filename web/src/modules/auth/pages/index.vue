<template>
  <div class="login-wrapper">
    <login-header />

    <main class="auth-main">
      <section class="login-container" aria-labelledby="auth-page-title">
        <div class="title-container">
          <h1 id="auth-page-title" class="title">{{ t('app.auth.login.title') }}</h1>
          <p class="brand-name">{{ t('common.appName') }}</p>
          <div class="sub-title">
            <p class="tip">
              {{ type === 'register' ? t('app.auth.login.haveAccount') : t('app.auth.login.noAccount') }}
            </p>
            <button
              type="button"
              class="tip switch-link"
              @click="switchType(type === 'register' ? 'login' : 'register')"
            >
              {{ type === 'register' ? t('app.auth.login.signIn') : t('app.auth.login.createAccount') }}
            </button>
          </div>
        </div>

        <div class="auth-surface">
          <login-panel v-if="type === 'login'" />
          <register-panel v-else @register-success="switchType('login')" />
        </div>
      </section>
      <infrastructure-canvas />
    </main>

    <footer class="copyright">{{ t(MESSAGE_KEY.COMMON_COPYRIGHT) }}</footer>
  </div>
</template>
<script setup lang="ts">
// 认证页只编排登录/注册视图与受限会话反馈，会话写入仍由 auth store 统一拥有。
import { ref } from 'vue';

import { MESSAGE_KEY } from '@/contracts/api/messages';
import { t } from '@/locales';

import LoginHeader from './components/Header.vue';
import InfrastructureCanvas from './components/InfrastructureCanvas.vue';
import LoginPanel from './components/Login.vue';
import RegisterPanel from './components/Register.vue';

defineOptions({
  name: 'AuthPage',
});

type AuthPanelType = 'login' | 'register';

const type = ref<AuthPanelType>('login');

const switchType = (value: AuthPanelType) => {
  type.value = value;
};
</script>
<style lang="less" scoped>
@import './index.less';
</style>
