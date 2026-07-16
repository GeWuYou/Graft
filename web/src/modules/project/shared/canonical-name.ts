const composeProjectNamePattern = /^[a-z0-9][a-z0-9_-]*$/;

/** 项目名称必须符合后端 Compose 标识约束，避免创建后再因规范化不一致产生资源漂移。 */
export function isValidApplicationCanonicalName(value: string) {
  return composeProjectNamePattern.test(value.trim());
}
