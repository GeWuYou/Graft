import type { paths } from '@/contracts/openapi/generated/schema';
import { USER_API_PATH } from '@/modules/user/contract/paths';
import type { SessionSummary } from '@/modules/user/types/user';
import { request } from '@/utils/request';

type UserSessionsTemplatePath = (typeof USER_API_PATH)['USER_SESSIONS_TEMPLATE'];
type UserSessionsRevokeAllTemplatePath = (typeof USER_API_PATH)['USER_SESSIONS_REVOKE_ALL_TEMPLATE'];
type UserSessionRevokeTemplatePath = (typeof USER_API_PATH)['USER_SESSION_REVOKE_TEMPLATE'];
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
    url: buildUserSessionsPath(userId),
    params: options.limit === undefined ? undefined : { limit: options.limit },
  });
}

export async function revokeAllUserSessions(userId: GetUserSessionsPathParams['id']): Promise<void> {
  await request.post<PostUserSessionsRevokeAllResponseData>({
    url: buildUserSessionsRevokeAllPath(userId),
  });
}

export async function revokeUserSession(
  userId: PostUserSessionRevokePathParams['id'],
  sessionID: PostUserSessionRevokePathParams['sessionID'],
): Promise<void> {
  await request.post<PostUserSessionRevokeResponseData>({
    url: buildUserSessionRevokePath(userId, sessionID),
  });
}

function buildUserSessionsPath(userId: GetUserSessionsPathParams['id']) {
  return USER_API_PATH.USER_SESSIONS_TEMPLATE.replace('{id}', encodeURIComponent(String(userId)));
}

function buildUserSessionsRevokeAllPath(userId: GetUserSessionsPathParams['id']) {
  return USER_API_PATH.USER_SESSIONS_REVOKE_ALL_TEMPLATE.replace('{id}', encodeURIComponent(String(userId)));
}

function buildUserSessionRevokePath(
  userId: PostUserSessionRevokePathParams['id'],
  sessionID: PostUserSessionRevokePathParams['sessionID'],
) {
  return USER_API_PATH.USER_SESSION_REVOKE_TEMPLATE.replace('{id}', encodeURIComponent(String(userId))).replace(
    '{sessionID}',
    encodeURIComponent(sessionID),
  );
}
