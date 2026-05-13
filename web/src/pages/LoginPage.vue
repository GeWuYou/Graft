<template>
  <t-card class="login-page" :bordered="false">
    <div class="login-page__header">
      <h2>{{ t('login.title') }}</h2>
      <p>{{ t('login.description') }}</p>
    </div>

    <form class="login-page__form" @submit.prevent="handleSubmit">
      <label class="login-page__field">
        <span>{{ t('login.fields.userName') }}</span>
        <t-input
          v-model="form.userName"
          clearable
          :placeholder="t('login.fields.userNamePlaceholder')"
        />
      </label>

      <label class="login-page__field">
        <span>{{ t('login.fields.password') }}</span>
        <t-input
          v-model="form.password"
          type="password"
          clearable
          :placeholder="t('login.fields.passwordPlaceholder')"
        />
      </label>

      <t-button
        block
        size="large"
        theme="primary"
        type="submit"
        :loading="submitting"
      >
        {{ t('common.actions.login') }}
      </t-button>
    </form>

    <div class="login-page__footer">
      <t-tag theme="primary" variant="light">{{
        t('login.tips.recommendedUser')
      }}</t-tag>
      <t-tag theme="default" variant="light">{{
        t('login.tips.recommendedPassword')
      }}</t-tag>
    </div>
  </t-card>
</template>

<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next';
import { reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { useI18n } from '@/app/i18n';
import { useAuthStore } from '@/stores/auth';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const { t } = useI18n();

const submitting = ref(false);
const form = reactive({
  userName: 'admin',
  password: '',
});

async function handleSubmit() {
  if (!form.userName.trim() || !form.password.trim()) {
    MessagePlugin.warning(t('login.messages.missingCredentials'));
    return;
  }

  submitting.value = true;

  try {
    authStore.login(form.userName.trim());
    MessagePlugin.success(t('login.messages.success'));

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
  box-shadow: 0 24px 72px rgb(15 23 42 / 12%);
}

.login-page__header h2 {
  color: #1a2433;
  font-size: 28px;
  margin: 0 0 12px;
}

.login-page__header p {
  color: #62748a;
  line-height: 1.7;
  margin: 0;
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
