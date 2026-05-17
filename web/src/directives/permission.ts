import type { App, DirectiveBinding } from 'vue';

import { usePermissionStore } from '@/store';

type PermissionBinding =
  | string
  | string[]
  | {
      allOf?: string[];
      anyOf?: string[];
    };

type PermissionDirectiveElement = HTMLElement & {
  __graftPermissionPlaceholder__?: Comment;
};

function hasPermission(binding: DirectiveBinding<PermissionBinding>) {
  const permissionStore = usePermissionStore();
  const value = binding.value;

  if (typeof value === 'string') {
    return permissionStore.hasPermission(value);
  }
  if (Array.isArray(value)) {
    return permissionStore.hasAllPermissions(value);
  }
  if (value && typeof value === 'object') {
    const allOf = value.allOf ?? [];
    const anyOf = value.anyOf ?? [];
    const matchesAll = allOf.length === 0 || permissionStore.hasAllPermissions(allOf);
    const matchesAny = anyOf.length === 0 || permissionStore.hasAnyPermission(anyOf);

    return matchesAll && matchesAny;
  }

  return false;
}

function updateVisibility(el: PermissionDirectiveElement, binding: DirectiveBinding<PermissionBinding>) {
  const allowed = hasPermission(binding);
  const parent = el.parentNode;

  if (allowed) {
    const placeholder = el.__graftPermissionPlaceholder__;
    if (placeholder?.parentNode) {
      placeholder.parentNode.replaceChild(el, placeholder);
    }
    return;
  }

  if (!parent) {
    return;
  }

  if (!el.__graftPermissionPlaceholder__) {
    el.__graftPermissionPlaceholder__ = document.createComment('graft-permission');
  }

  if (parent.contains(el)) {
    parent.replaceChild(el.__graftPermissionPlaceholder__, el);
  }
}

export function registerPermissionDirective(app: App<Element>) {
  app.directive('permission', {
    mounted(el, binding) {
      updateVisibility(el as PermissionDirectiveElement, binding);
    },
    updated(el, binding) {
      updateVisibility(el as PermissionDirectiveElement, binding);
    },
  });
}
