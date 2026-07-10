const canonicalProjectNamePattern = /^[a-z0-9][a-z0-9_-]*$/;

/**
 * 校验项目名称是否符合规范格式。
 *
 * @param value - 待校验的项目名称
 * @returns `true` 如果去除首尾空白后的名称以小写字母或数字开头，且仅包含小写字母、数字、下划线或短横线；否则返回 `false`
 */
export function isValidProjectCanonicalName(value: string) {
  return canonicalProjectNamePattern.test(value.trim());
}
