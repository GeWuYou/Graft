type BlankComposeWorkspaceFile = {
  path: string;
  node_type: 'file';
  content: string;
};

const DEFAULT_ENV_CONTENT = 'APP_IMAGE=nginx:alpine\nAPP_PORT=8080\n';
const DEFAULT_COMPOSE_CONTENT = 'services:\n  app:\n    image: ${APP_IMAGE}\n    ports:\n      - "${APP_PORT}:80"\n';

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
