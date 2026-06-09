// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { paths } from '@/contracts/openapi/generated/schema';
import { AUTH_API_PATH } from '@/modules/auth/contract/paths';
import type {
  BootstrapResponse,
  ChangePasswordPayload,
  CompleteRequiredPasswordChangePayload,
  LoginPayload,
  LoginResponse,
  SessionSummary,
} from '@/modules/auth/contract/types';
import { request } from '@/utils/request';

type LoginPath = (typeof AUTH_API_PATH)['LOGIN'];
type BootstrapPath = (typeof AUTH_API_PATH)['BOOTSTRAP'];
type RefreshPath = (typeof AUTH_API_PATH)['REFRESH'];
type LogoutPath = (typeof AUTH_API_PATH)['LOGOUT'];
type ChangePasswordPath = (typeof AUTH_API_PATH)['CHANGE_PASSWORD'];
type CompleteRequiredPasswordChangePath = (typeof AUTH_API_PATH)['COMPLETE_REQUIRED_PASSWORD_CHANGE'];
type SessionsPath = (typeof AUTH_API_PATH)['SESSIONS'];
type SessionRevokeTemplatePath = (typeof AUTH_API_PATH)['SESSION_REVOKE_TEMPLATE'];
type SessionsRevokeAllPath = (typeof AUTH_API_PATH)['SESSIONS_REVOKE_ALL'];
type SessionsRevokeOthersPath = (typeof AUTH_API_PATH)['SESSIONS_REVOKE_OTHERS'];
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

// Keep generated request/response typing at the module API boundary; callers still own form-local state.
export function login(payload: LoginPayload) {
  return request.post<PostAuthLoginResponseData>({
    url: AUTH_API_PATH.LOGIN,
    data: payload,
  });
}

export function refresh() {
  return request.post<LoginResponse & PostAuthRefreshResponseData>({
    url: AUTH_API_PATH.REFRESH,
  });
}

export async function logout(): Promise<void> {
  await request.post<PostAuthLogoutResponseData>({
    url: AUTH_API_PATH.LOGOUT,
  });
}

export function listSessions(options: ListSessionsOptions = {}) {
  return request.get<SessionSummary[] & GetAuthSessionsResponseData>({
    url: AUTH_API_PATH.SESSIONS,
    params: options.limit === undefined ? undefined : { limit: options.limit },
  });
}

export async function revokeAllSessions(): Promise<void> {
  await request.post<PostAuthSessionsRevokeAllResponseData>({
    url: AUTH_API_PATH.SESSIONS_REVOKE_ALL,
  });
}

export async function revokeOtherSessions(): Promise<void> {
  await request.post<PostAuthSessionsRevokeOthersResponseData>({
    url: AUTH_API_PATH.SESSIONS_REVOKE_OTHERS,
  });
}

export async function revokeSession(sessionID: PostAuthSessionRevokePathParams['sessionID']): Promise<void> {
  await request.post<PostAuthSessionRevokeResponseData>({
    url: buildSessionRevokePath(sessionID),
  });
}

export async function changePassword(payload: ChangePasswordPayload): Promise<void> {
  await request.post<PostAuthChangePasswordResponseData>({
    url: AUTH_API_PATH.CHANGE_PASSWORD,
    data: payload,
  });
}

export function completeRequiredPasswordChange(payload: CompleteRequiredPasswordChangePayload) {
  return request.post<PostAuthCompleteRequiredPasswordChangeResponseData>({
    url: AUTH_API_PATH.COMPLETE_REQUIRED_PASSWORD_CHANGE,
    data: payload,
  });
}

export function getBootstrap() {
  return request.get<BootstrapResponse & GetAuthBootstrapResponseData>({
    url: AUTH_API_PATH.BOOTSTRAP,
  });
}

function buildSessionRevokePath(sessionID: PostAuthSessionRevokePathParams['sessionID']) {
  return AUTH_API_PATH.SESSION_REVOKE_TEMPLATE.replace('{sessionID}', encodeURIComponent(sessionID));
}
