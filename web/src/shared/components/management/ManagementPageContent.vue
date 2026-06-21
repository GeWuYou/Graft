<template>
  <div class="management-page-content">
    <slot />
  </div>
</template>
<style scoped lang="less">
.management-page-content {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-width: 0;
  padding-bottom: max(
    calc(var(--td-comp-margin-xxl) + var(--graft-density-gap-8)),
    calc(var(--graft-floating-toolbar-safe-area) + var(--graft-density-gap-8))
  );
  width: 100%;
}

@media (width <= 768px) {
  .management-page-content {
    gap: var(--graft-density-gap-12);
    padding-bottom: var(--td-comp-margin-xl);
  }
}
</style>
