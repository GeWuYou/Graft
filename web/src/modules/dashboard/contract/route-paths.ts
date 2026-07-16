/** 统一 dashboard 路径契约，移除 query/hash、收敛首尾斜杠，避免菜单路径比较受输入写法影响。 */

export function normalizeDashboardRoutePath(path: string) {
  const trimmed = path.trim();
  if (!trimmed) {
    return '';
  }

  const pathOnly = trimmed.split(/[?#]/, 1)[0] ?? '';
  const withLeadingSlash = pathOnly.startsWith('/') ? pathOnly : `/${pathOnly}`;
  return withLeadingSlash === '/' ? withLeadingSlash : withLeadingSlash.replace(/\/+$/, '');
}

/** 按绝对子路径优先、相对路径拼接的规则构造 dashboard 路径，供菜单和快捷入口共享同一语义。 */
export function normalizeJoinedDashboardRoutePath(parentPath: string, routePath: string) {
  if (routePath.startsWith('/')) {
    return normalizeDashboardRoutePath(routePath);
  }

  if (!routePath) {
    return normalizeDashboardRoutePath(parentPath);
  }

  return normalizeDashboardRoutePath(`${parentPath}/${routePath}`);
}
