import { BUILD_ROUTE_PATH } from './paths';

export function buildCreateJobPath(applicationId?: string) {
  if (!applicationId) return BUILD_ROUTE_PATH.CREATE;
  const query = new URLSearchParams({ application_id: applicationId });
  return `${BUILD_ROUTE_PATH.CREATE}?${query.toString()}`;
}
