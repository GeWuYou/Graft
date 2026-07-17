/** 描述新建 Compose 工作区中默认提供的文本文件。 */
export type BlankComposeWorkspaceFile = {
  path: string;
  node_type: 'file';
  content: string;
};

const DEFAULT_ENV_CONTENT = 'APP_IMAGE=nginx:alpine\nAPP_PORT=8080\n';
const DEFAULT_COMPOSE_CONTENT = 'services:\n  app:\n    image: ${APP_IMAGE}\n    ports:\n      - "${APP_PORT}:80"\n';

/**
 * 创建模板向导和应用向导共用的最小可编辑 Compose 工作区。
 *
 * @returns 相互独立的默认 `.env` 和 `compose.yaml` 文件条目。
 */
export function createBlankComposeWorkspaceFiles(): BlankComposeWorkspaceFile[] {
  return [
    { path: '.env', node_type: 'file', content: DEFAULT_ENV_CONTENT },
    {
      path: 'compose.yaml',
      node_type: 'file',
      content: DEFAULT_COMPOSE_CONTENT,
    },
  ];
}
