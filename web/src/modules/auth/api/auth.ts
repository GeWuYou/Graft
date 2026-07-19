import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import type {
  BootstrapResponse,
  ChangePasswordPayload,
  CompleteRequiredPasswordChangePayload,
  LoginPayload,
  LoginResponse,
  SessionSummary,
} from '@/modules/auth/contract/types';
import { request } from '@/utils/request';

type LoginPath = typeof OPENAPI_RUNTIME_PATH.postAuthLogin;
type BootstrapPath = typeof OPENAPI_RUNTIME_PATH.getAuthBootstrap;
type RefreshPath = typeof OPENAPI_RUNTIME_PATH.postAuthRefresh;
type LogoutPath = typeof OPENAPI_RUNTIME_PATH.postAuthLogout;
type ChangePasswordPath = typeof OPENAPI_RUNTIME_PATH.postAuthChangePassword;
type CompleteRequiredPasswordChangePath = typeof OPENAPI_RUNTIME_PATH.postAuthCompleteRequiredPasswordChange;
type SessionsPath = typeof OPENAPI_RUNTIME_PATH.getAuthSessions;
type SessionRevokeTemplatePath = typeof OPENAPI_RUNTIME_PATH.postAuthSessionRevoke;
type SessionsRevokeAllPath = typeof OPENAPI_RUNTIME_PATH.postAuthSessionsRevokeAll;
type SessionsRevokeOthersPath = typeof OPENAPI_RUNTIME_PATH.postAuthSessionsRevokeOthers;
type PostAuthLoginOperation = paths[LoginPath]['post'];
type GetAuthBootstrapOperation = paths[BootstrapPath]['get'];
type PostAuthRefreshOperation = paths[RefreshPath]['post'];
type PostAuthLogoutOperation = paths[LogoutPath]['post'];
type PostAuthChangePasswordOperation = paths[ChangePasswordPath]['post'];
type PostAuthCompleteRequiredPasswordChangeOperation = paths[CompleteRequiredPasswordChangePath]['post'];
type GetAuthSessionsOperation = paths[SessionsPath]['get'];
type PostAuthSessionsRevokeAllOperation = paths[SessionsRevokeAllPath]['post'];
type PostAuthSessionsRevokeOthersOperation = paths[SessionsRevokeOthersPath]['post'];
type PostAuthSessionRevokeOperation = paths[SessionRevokeTemplatePath]['post'];
type PostAuthSessionRevokePathParams = NonNullable<PostAuthSessionRevokeOperation['parameters']['path']>;
type PostAuthLoginResponse = PostAuthLoginOperation['responses']['200']['content']['application/json'];
type GetAuthBootstrapResponse = GetAuthBootstrapOperation['responses']['200']['content']['application/json'];
type PostAuthRefreshResponse = PostAuthRefreshOperation['responses']['200']['content']['application/json'];
type PostAuthLogoutResponse = PostAuthLogoutOperation['responses']['200']['content']['application/json'];
type PostAuthChangePasswordResponse =
  PostAuthChangePasswordOperation['responses']['200']['content']['application/json'];
type PostAuthCompleteRequiredPasswordChangeResponse =
  PostAuthCompleteRequiredPasswordChangeOperation['responses']['200']['content']['application/json'];
type GetAuthSessionsResponse = GetAuthSessionsOperation['responses']['200']['content']['application/json'];
type PostAuthSessionsRevokeAllResponse =
  PostAuthSessionsRevokeAllOperation['responses']['200']['content']['application/json'];
type PostAuthSessionsRevokeOthersResponse =
  PostAuthSessionsRevokeOthersOperation['responses']['200']['content']['application/json'];
type PostAuthSessionRevokeResponse = PostAuthSessionRevokeOperation['responses']['200']['content']['application/json'];
type PostAuthLoginResponseData = NonNullable<PostAuthLoginResponse['data']>;
type GetAuthBootstrapResponseData = NonNullable<GetAuthBootstrapResponse['data']>;
type PostAuthRefreshResponseData = NonNullable<PostAuthRefreshResponse['data']>;
type PostAuthLogoutResponseData = PostAuthLogoutResponse['data'];
type PostAuthChangePasswordResponseData = PostAuthChangePasswordResponse['data'];
type PostAuthCompleteRequiredPasswordChangeResponseData = PostAuthCompleteRequiredPasswordChangeResponse['data'];
type GetAuthSessionsResponseData = NonNullable<GetAuthSessionsResponse['data']>;
type PostAuthSessionsRevokeAllResponseData = PostAuthSessionsRevokeAllResponse['data'];
type PostAuthSessionsRevokeOthersResponseData = PostAuthSessionsRevokeOthersResponse['data'];
type PostAuthSessionRevokeResponseData = PostAuthSessionRevokeResponse['data'];

type GetAuthSessionsQuery = NonNullable<GetAuthSessionsOperation['parameters']['query']>;

export type ListSessionsOptions = {
  limit?: GetAuthSessionsQuery['limit'];
};

// 模块 API 边界直接复用 OpenAPI 生成类型；表单局部状态仍由调用方拥有，避免 API 类型侵入页面模型。
export function login(payload: LoginPayload) {
  return request.post<PostAuthLoginResponseData>({
    url: OPENAPI_RUNTIME_PATH.postAuthLogin,
    data: payload,
  });
}

export function refresh() {
  return request.post<LoginResponse & PostAuthRefreshResponseData>({
    url: OPENAPI_RUNTIME_PATH.postAuthRefresh,
  });
}

export async function logout(): Promise<void> {
  await request.post<PostAuthLogoutResponseData>({
    url: OPENAPI_RUNTIME_PATH.postAuthLogout,
  });
}

export function listSessions(options: ListSessionsOptions = {}) {
  return request.get<SessionSummary[] & GetAuthSessionsResponseData>({
    url: OPENAPI_RUNTIME_PATH.getAuthSessions,
    params: options.limit === undefined ? undefined : { limit: options.limit },
  });
}

export async function revokeAllSessions(): Promise<void> {
  await request.post<PostAuthSessionsRevokeAllResponseData>({
    url: OPENAPI_RUNTIME_PATH.postAuthSessionsRevokeAll,
  });
}

export async function revokeOtherSessions(): Promise<void> {
  await request.post<PostAuthSessionsRevokeOthersResponseData>({
    url: OPENAPI_RUNTIME_PATH.postAuthSessionsRevokeOthers,
  });
}

export async function revokeSession(sessionID: PostAuthSessionRevokePathParams['sessionID']): Promise<void> {
  await request.post<PostAuthSessionRevokeResponseData>({
    url: buildSessionRevokePath(sessionID),
  });
}

export async function changePassword(payload: ChangePasswordPayload): Promise<void> {
  await request.post<PostAuthChangePasswordResponseData>({
    url: OPENAPI_RUNTIME_PATH.postAuthChangePassword,
    data: payload,
  });
}

export function completeRequiredPasswordChange(payload: CompleteRequiredPasswordChangePayload) {
  return request.post<PostAuthCompleteRequiredPasswordChangeResponseData>({
    url: OPENAPI_RUNTIME_PATH.postAuthCompleteRequiredPasswordChange,
    data: payload,
  });
}

export function getBootstrap() {
  return request.get<BootstrapResponse & GetAuthBootstrapResponseData>({
    url: OPENAPI_RUNTIME_PATH.getAuthBootstrap,
  });
}

function buildSessionRevokePath(sessionID: PostAuthSessionRevokePathParams['sessionID']) {
  return buildOpenApiRuntimePath('postAuthSessionRevoke', { sessionID });
}
