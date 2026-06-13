// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import {
  ANNOUNCEMENT_API_PATH,
  buildAnnouncementArchiveApiPath,
  buildAnnouncementDetailApiPath,
  buildAnnouncementPublishApiPath,
  buildMyAnnouncementReadApiPath,
} from '../contract/paths';
import type {
  AnnouncementItem,
  AnnouncementListQuery,
  AnnouncementListResponse,
  AnnouncementReadAllResponse,
  AnnouncementUnreadCountResponse,
  CreateAnnouncementRequest,
  MyAnnouncementListQuery,
  PublishAnnouncementRequest,
  UpdateAnnouncementRequest,
} from '../types/announcement';

type AnnouncementListPath = (typeof ANNOUNCEMENT_API_PATH)['LIST'];
type GetAnnouncementsOperation = paths[AnnouncementListPath]['get'];
type GetAnnouncementsEnvelope = GetAnnouncementsOperation['responses'][200]['content']['application/json'];
type GetAnnouncementsData = NonNullable<GetAnnouncementsEnvelope['data']>;
type GetAnnouncementsQuery = NonNullable<GetAnnouncementsOperation['parameters']['query']>;

type PostAnnouncementsOperation = paths[AnnouncementListPath]['post'];
type PostAnnouncementsEnvelope = PostAnnouncementsOperation['responses'][201]['content']['application/json'];
type PostAnnouncementsData = NonNullable<PostAnnouncementsEnvelope['data']>;
type PostAnnouncementsBody = PostAnnouncementsOperation['requestBody']['content']['application/json'];

type AnnouncementDetailPath = (typeof ANNOUNCEMENT_API_PATH)['DETAIL'];
type GetAnnouncementOperation = paths[AnnouncementDetailPath]['get'];
type GetAnnouncementEnvelope = GetAnnouncementOperation['responses'][200]['content']['application/json'];
type GetAnnouncementData = NonNullable<GetAnnouncementEnvelope['data']>;
type GetAnnouncementPathParams = GetAnnouncementOperation['parameters']['path'];

type PutAnnouncementOperation = paths[AnnouncementDetailPath]['put'];
type PutAnnouncementEnvelope = PutAnnouncementOperation['responses'][200]['content']['application/json'];
type PutAnnouncementData = NonNullable<PutAnnouncementEnvelope['data']>;
type PutAnnouncementPathParams = PutAnnouncementOperation['parameters']['path'];
type PutAnnouncementBody = PutAnnouncementOperation['requestBody']['content']['application/json'];

type DeleteAnnouncementOperation = paths[AnnouncementDetailPath]['delete'];
type DeleteAnnouncementPathParams = DeleteAnnouncementOperation['parameters']['path'];

type AnnouncementPublishPath = (typeof ANNOUNCEMENT_API_PATH)['PUBLISH'];
type PostAnnouncementPublishOperation = paths[AnnouncementPublishPath]['post'];
type PostAnnouncementPublishEnvelope =
  PostAnnouncementPublishOperation['responses'][200]['content']['application/json'];
type PostAnnouncementPublishData = NonNullable<PostAnnouncementPublishEnvelope['data']>;
type PostAnnouncementPublishPathParams = PostAnnouncementPublishOperation['parameters']['path'];
type PostAnnouncementPublishBody = NonNullable<
  PostAnnouncementPublishOperation['requestBody']
>['content']['application/json'];

type AnnouncementArchivePath = (typeof ANNOUNCEMENT_API_PATH)['ARCHIVE'];
type PostAnnouncementArchiveOperation = paths[AnnouncementArchivePath]['post'];
type PostAnnouncementArchiveEnvelope =
  PostAnnouncementArchiveOperation['responses'][200]['content']['application/json'];
type PostAnnouncementArchiveData = NonNullable<PostAnnouncementArchiveEnvelope['data']>;
type PostAnnouncementArchivePathParams = PostAnnouncementArchiveOperation['parameters']['path'];

type MyAnnouncementListPath = (typeof ANNOUNCEMENT_API_PATH)['MY_LIST'];
type GetMyAnnouncementsOperation = paths[MyAnnouncementListPath]['get'];
type GetMyAnnouncementsEnvelope = GetMyAnnouncementsOperation['responses'][200]['content']['application/json'];
type GetMyAnnouncementsData = NonNullable<GetMyAnnouncementsEnvelope['data']>;
type GetMyAnnouncementsQuery = NonNullable<GetMyAnnouncementsOperation['parameters']['query']>;

type MyAnnouncementReadPath = (typeof ANNOUNCEMENT_API_PATH)['MY_READ'];
type PostMyAnnouncementReadOperation = paths[MyAnnouncementReadPath]['post'];
type PostMyAnnouncementReadEnvelope = PostMyAnnouncementReadOperation['responses'][200]['content']['application/json'];
type PostMyAnnouncementReadData = NonNullable<PostMyAnnouncementReadEnvelope['data']>;
type PostMyAnnouncementReadPathParams = PostMyAnnouncementReadOperation['parameters']['path'];

type MyAnnouncementReadAllPath = (typeof ANNOUNCEMENT_API_PATH)['MY_READ_ALL'];
type PostMyAnnouncementsReadAllOperation = paths[MyAnnouncementReadAllPath]['post'];
type PostMyAnnouncementsReadAllEnvelope =
  PostMyAnnouncementsReadAllOperation['responses'][200]['content']['application/json'];
type PostMyAnnouncementsReadAllData = NonNullable<PostMyAnnouncementsReadAllEnvelope['data']>;

type MyAnnouncementUnreadCountPath = (typeof ANNOUNCEMENT_API_PATH)['MY_UNREAD_COUNT'];
type GetMyAnnouncementsUnreadCountOperation = paths[MyAnnouncementUnreadCountPath]['get'];
type GetMyAnnouncementsUnreadCountEnvelope =
  GetMyAnnouncementsUnreadCountOperation['responses'][200]['content']['application/json'];
type GetMyAnnouncementsUnreadCountData = NonNullable<GetMyAnnouncementsUnreadCountEnvelope['data']>;

export function getAnnouncements(query?: AnnouncementListQuery): Promise<AnnouncementListResponse> {
  return request.get<GetAnnouncementsData>({
    url: ANNOUNCEMENT_API_PATH.LIST,
    params: normalizeAnnouncementListQuery(query),
  });
}

export function createAnnouncement(payload: CreateAnnouncementRequest): Promise<AnnouncementItem> {
  return request.post<PostAnnouncementsData>({
    url: ANNOUNCEMENT_API_PATH.LIST,
    data: payload as PostAnnouncementsBody,
  });
}

export function getAnnouncement(id: GetAnnouncementPathParams['id']): Promise<AnnouncementItem> {
  return request.get<GetAnnouncementData>({
    url: buildAnnouncementDetailApiPath(id),
  });
}

export function updateAnnouncement(
  id: PutAnnouncementPathParams['id'],
  payload: UpdateAnnouncementRequest,
): Promise<AnnouncementItem> {
  return request.put<PutAnnouncementData>({
    url: buildAnnouncementDetailApiPath(id),
    data: payload as PutAnnouncementBody,
  });
}

export function publishAnnouncement(
  id: PostAnnouncementPublishPathParams['id'],
  payload?: PublishAnnouncementRequest,
): Promise<AnnouncementItem> {
  return request.post<PostAnnouncementPublishData>({
    url: buildAnnouncementPublishApiPath(id),
    data: payload as PostAnnouncementPublishBody | undefined,
  });
}

export function archiveAnnouncement(id: PostAnnouncementArchivePathParams['id']): Promise<AnnouncementItem> {
  return request.post<PostAnnouncementArchiveData>({
    url: buildAnnouncementArchiveApiPath(id),
  });
}

export function getMyAnnouncements(query?: MyAnnouncementListQuery): Promise<AnnouncementListResponse> {
  return request.get<GetMyAnnouncementsData>({
    url: ANNOUNCEMENT_API_PATH.MY_LIST,
    params: normalizeMyAnnouncementListQuery(query),
  });
}

export function markAnnouncementRead(id: PostMyAnnouncementReadPathParams['id']): Promise<AnnouncementItem> {
  return request.post<PostMyAnnouncementReadData>({
    url: buildMyAnnouncementReadApiPath(id),
  });
}

export function markAllAnnouncementsRead(): Promise<AnnouncementReadAllResponse> {
  return request.post<PostMyAnnouncementsReadAllData>({
    url: ANNOUNCEMENT_API_PATH.MY_READ_ALL,
  });
}

export function getAnnouncementUnreadCount(): Promise<AnnouncementUnreadCountResponse> {
  return request.get<GetMyAnnouncementsUnreadCountData>({
    url: ANNOUNCEMENT_API_PATH.MY_UNREAD_COUNT,
  });
}

export function deleteAnnouncement(id: DeleteAnnouncementPathParams['id']) {
  return request.delete<Record<string, never>>({
    url: buildAnnouncementDetailApiPath(id),
  });
}

export function normalizeAnnouncementListQuery(query?: AnnouncementListQuery): GetAnnouncementsQuery | undefined {
  if (!query) {
    return undefined;
  }

  return {
    ...(query.keyword ? { keyword: query.keyword } : {}),
    ...(query.level ? { level: query.level } : {}),
    ...(query.page ? { page: query.page } : {}),
    ...(query.page_size ? { page_size: query.page_size } : {}),
    ...(typeof query.pinned === 'boolean' ? { pinned: query.pinned } : {}),
    ...(query.sort ? { sort: query.sort } : {}),
    ...(query.status ? { status: query.status } : {}),
  } satisfies GetAnnouncementsQuery;
}

export function normalizeMyAnnouncementListQuery(query?: MyAnnouncementListQuery): GetMyAnnouncementsQuery | undefined {
  if (!query) {
    return undefined;
  }

  return {
    ...(query.page ? { page: query.page } : {}),
    ...(query.page_size ? { page_size: query.page_size } : {}),
    ...(typeof query.unread_only === 'boolean' ? { unread_only: query.unread_only } : {}),
  } satisfies GetMyAnnouncementsQuery;
}
