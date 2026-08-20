import { BUILD_ROUTE_PATH } from './paths';

export const BUILD_BOOTSTRAP_ROUTE = {
  JOBS: {
    menuPath: BUILD_ROUTE_PATH.JOBS,
    routeName: 'BuildJobList',
  },
  ARTIFACTS: {
    menuPath: BUILD_ROUTE_PATH.ARTIFACTS,
    routeName: 'BuildArtifactList',
  },
  WORKSPACES: {
    menuPath: BUILD_ROUTE_PATH.WORKSPACES,
    routeName: 'BuildWorkspaceList',
  },
  CREATE: {
    path: BUILD_ROUTE_PATH.CREATE,
    pageRouteName: 'BuildJobCreateIndex',
    routeName: 'BuildJobCreate',
  },
  CREATE_WORKSPACE: {
    path: BUILD_ROUTE_PATH.CREATE_WORKSPACE,
    pageRouteName: 'BuildWorkspaceCreateIndex',
    routeName: 'BuildWorkspaceCreate',
  },
} as const;
