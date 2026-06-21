<template>
  <section>
    <h2>{{ t('demo.report.title') }}</h2>
    <button>
      <span aria-hidden="true">
        <span>Create report</span>
      </span>
      {{ t('demo.report.description') }}
    </button>
  </section>
</template>

<script setup lang="ts">
import { t } from '@/locales';
</script>
