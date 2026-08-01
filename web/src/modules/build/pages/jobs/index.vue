<template>
  <div class="build-jobs-page">
    <h1>{{ t('build.jobs.title') }}</h1>
    <p>{{ t('build.jobs.placeholder') }}</p>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
</script>
