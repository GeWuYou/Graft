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
  CREATE: {
    path: BUILD_ROUTE_PATH.CREATE,
    pageRouteName: 'BuildJobCreateIndex',
    routeName: 'BuildJobCreate',
  },
} as const;
