<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

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
