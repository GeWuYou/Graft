/**
 * 构造升级操作的实时主题名；主题格式由服务端实时订阅契约统一定义。
 */
export function buildUpdateOperationTopicName(operationID: string) {
  return `platform.update.operations.${operationID}`;
}
