<template>
  <t-card class="login-page" :bordered="false">
    <div class="login-page__header">
      <h2>登录 Graft</h2>
      <p>当前阶段使用静态登录态模拟后台返回的 token、用户信息和权限集合。</p>
    </div>

    <form class="login-page__form" @submit.prevent="handleSubmit">
      <label class="login-page__field">
        <span>用户名</span>
        <t-input v-model="form.userName" clearable placeholder="请输入用户名" />
      </label>

      <label class="login-page__field">
        <span>密码</span>
        <t-input
          v-model="form.password"
          type="password"
          clearable
          placeholder="请输入密码"
        />
      </label>

      <t-button
        block
        size="large"
        theme="primary"
        type="submit"
        :loading="submitting"
      >
        登录
      </t-button>
    </form>

    <div class="login-page__footer">
      <t-tag theme="primary" variant="light">建议账号：admin</t-tag>
      <t-tag theme="default" variant="light">建议密码：任意非空值</t-tag>
    </div>
  </t-card>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue';
import { MessagePlugin } from 'tdesign-vue-next';
import { useRoute, useRouter } from 'vue-router';

import { useAuthStore } from '@/stores/auth';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

const submitting = ref(false);
const form = reactive({
  userName: 'admin',
  password: '',
});

async function handleSubmit() {
  if (!form.userName.trim() || !form.password.trim()) {
    MessagePlugin.warning('请输入用户名和密码');
    return;
  }

  submitting.value = true;

  try {
    authStore.login(form.userName.trim());
    MessagePlugin.success('登录成功');

    const redirect =
      typeof route.query.redirect === 'string' && route.query.redirect
        ? route.query.redirect
        : '/dashboard';

    await router.replace(redirect);
  } finally {
    submitting.value = false;
  }
}
</script>

<style scoped>
.login-page {
  border-radius: 24px;
  box-shadow: 0 24px 72px rgba(15, 23, 42, 0.12);
}

.login-page__header h2 {
  margin: 0 0 12px;
  color: #1a2433;
  font-size: 28px;
}

.login-page__header p {
  margin: 0;
  color: #62748a;
  line-height: 1.7;
}

.login-page__form {
  display: grid;
  gap: 16px;
  margin-top: 28px;
}

.login-page__field {
  display: grid;
  gap: 8px;
}

.login-page__field span {
  color: #1f2f45;
  font-size: 14px;
  font-weight: 600;
}

.login-page__footer {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 20px;
}
</style>
