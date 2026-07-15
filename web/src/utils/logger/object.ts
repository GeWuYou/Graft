/**
 * 判断值是否为普通对象。
 *
 * @param value - 要检查的值
 * @returns 原型为 `Object.prototype` 或 `null` 时为 `true`，否则为 `false`
 */
export function isPlainObject(value: unknown): value is Record<string, unknown> {
  if (value === null || typeof value !== 'object') {
    return false;
  }

  const prototype = Object.getPrototypeOf(value);
  return prototype === Object.prototype || prototype === null;
}
