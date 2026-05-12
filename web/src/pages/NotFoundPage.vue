<template>
  <div class="not-found-page">
    <t-card class="not-found-page__card" :bordered="false">
      <span class="not-found-page__code">404</span>
      <h1>页面不存在</h1>
      <p>当前地址没有匹配的静态路由，后续动态模块接入后也会复用同一套兜底页。</p>
      <t-space>
        <t-button theme="primary" @click="goDashboard">返回仪表盘</t-button>
        <t-button variant="outline" theme="default" @click="goLogin">返回登录页</t-button>
      </t-space>
    </t-card>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';

const router = useRouter();

function goDashboard() {
  void router.push('/dashboard');
}

function goLogin() {
  void router.push('/login');
}
</script>

<style scoped>
.not-found-page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 24px;
  background: linear-gradient(180deg, #f4f7fb 0%, #eef3f8 100%);
}

.not-found-page__card {
  width: min(100%, 480px);
  text-align: center;
  border-radius: 24px;
}

.not-found-page__code {
  display: inline-block;
  color: #0052d9;
  font-size: 64px;
  font-weight: 700;
  line-height: 1;
}

.not-found-page__card h1 {
  margin: 20px 0 12px;
  color: #1a2433;
}

.not-found-page__card p {
  margin: 0 0 24px;
  color: #66788f;
  line-height: 1.7;
}
</style>
