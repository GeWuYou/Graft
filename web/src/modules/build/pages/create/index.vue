<template>
  <div class="build-create-page">
    <h1>{{ t('build.jobs.create.title') }}</h1>
    <p>{{ t('build.jobs.create.placeholder') }}</p>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
</script>
