import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import type { SessionSummary } from '@/modules/user/types/user';
import { request } from '@/utils/request';

type UserSessionsTemplatePath = typeof OPENAPI_RUNTIME_PATH.getUserSessions;
type UserSessionsRevokeAllTemplatePath = typeof OPENAPI_RUNTIME_PATH.postUserSessionsRevokeAll;
type UserSessionRevokeTemplatePath = typeof OPENAPI_RUNTIME_PATH.postUserSessionRevoke;
type GetUserSessionsOperation = paths[UserSessionsTemplatePath]['get'];
type PostUserSessionsRevokeAllOperation = paths[UserSessionsRevokeAllTemplatePath]['post'];
type PostUserSessionRevokeOperation = paths[UserSessionRevokeTemplatePath]['post'];
type GetUserSessionsPathParams = NonNullable<GetUserSessionsOperation['parameters']['path']>;
type GetUserSessionsQuery = NonNullable<GetUserSessionsOperation['parameters']['query']>;
type PostUserSessionRevokePathParams = NonNullable<PostUserSessionRevokeOperation['parameters']['path']>;
type GetUserSessionsResponse = GetUserSessionsOperation['responses']['200']['content']['application/json'];
type PostUserSessionsRevokeAllResponse =
  PostUserSessionsRevokeAllOperation['responses']['200']['content']['application/json'];
type PostUserSessionRevokeResponse = PostUserSessionRevokeOperation['responses']['200']['content']['application/json'];
type GetUserSessionsResponseData = NonNullable<GetUserSessionsResponse['data']>;
type PostUserSessionsRevokeAllResponseData = PostUserSessionsRevokeAllResponse['data'];
type PostUserSessionRevokeResponseData = PostUserSessionRevokeResponse['data'];

export type ListUserSessionsOptions = {
  limit?: GetUserSessionsQuery['limit'];
};

export function listUserSessions(userId: GetUserSessionsPathParams['id'], options: ListUserSessionsOptions = {}) {
  return request.get<SessionSummary[] & GetUserSessionsResponseData>({
    url: buildOpenApiRuntimePath('getUserSessions', { id: userId }),
    params: options.limit === undefined ? undefined : { limit: options.limit },
  });
}

export async function revokeAllUserSessions(userId: GetUserSessionsPathParams['id']): Promise<void> {
  await request.post<PostUserSessionsRevokeAllResponseData>({
    url: buildOpenApiRuntimePath('postUserSessionsRevokeAll', { id: userId }),
  });
}

export async function revokeUserSession(
  userId: PostUserSessionRevokePathParams['id'],
  sessionID: PostUserSessionRevokePathParams['sessionID'],
): Promise<void> {
  await request.post<PostUserSessionRevokeResponseData>({
    url: buildOpenApiRuntimePath('postUserSessionRevoke', { id: userId, sessionID }),
  });
}
