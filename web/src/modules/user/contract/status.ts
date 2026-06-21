/**
 * USER_STATUS 定义用户账号状态的稳定外部字面量。
 *
 * 页面、表单和接口适配层都应复用这里的值，不要在其他位置重复声明账号状态字符串。
 */
export const USER_STATUS = {
  DISABLED: 'disabled',
  ENABLED: 'enabled',
} as const;

/**
 * UserStatus 表示允许写入和读取的用户账号状态联合类型。
 *
 * 它跟随 `USER_STATUS` 导出，作为用户状态契约的类型化边界。
 */
export type UserStatus = (typeof USER_STATUS)[keyof typeof USER_STATUS];
