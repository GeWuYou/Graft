// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

declare module '*.svg?component' {
  import type { DefineComponent } from 'vue';

  const component: DefineComponent<object, object, unknown>;
  export default component;
}
