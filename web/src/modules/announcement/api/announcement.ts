import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type {
  AnnouncementItem,
  AnnouncementListQuery,
  AnnouncementListResponse,
  AnnouncementReadAllResponse,
  AnnouncementSavedView,
  AnnouncementSavedViewRequest,
  AnnouncementUnreadCountResponse,
  CreateAnnouncementRequest,
  MyAnnouncementListQuery,
  PublishAnnouncementRequest,
  UpdateAnnouncementRequest,
} from '../types/announcement';

type AnnouncementListPath = typeof OPENAPI_RUNTIME_PATH.getAnnouncements;
type GetAnnouncementsOperation = paths[AnnouncementListPath]['get'];
type GetAnnouncementsEnvelope = GetAnnouncementsOperation['responses'][200]['content']['application/json'];
type GetAnnouncementsData = NonNullable<GetAnnouncementsEnvelope['data']>;
type GetAnnouncementsQuery = NonNullable<GetAnnouncementsOperation['parameters']['query']>;

type AnnouncementSavedViewsPath = typeof OPENAPI_RUNTIME_PATH.getAnnouncementSavedViews;
type GetAnnouncementSavedViewsOperation = paths[AnnouncementSavedViewsPath]['get'];
type GetAnnouncementSavedViewsResponse =
  GetAnnouncementSavedViewsOperation['responses'][200]['content']['application/json'];
type GetAnnouncementSavedViewsResponseData = NonNullable<GetAnnouncementSavedViewsResponse['data']>;
type PostAnnouncementSavedViewOperation = paths[AnnouncementSavedViewsPath]['post'];
type PostAnnouncementSavedViewResponse =
  PostAnnouncementSavedViewOperation['responses'][201]['content']['application/json'];
type PostAnnouncementSavedViewResponseData = NonNullable<PostAnnouncementSavedViewResponse['data']>;

type AnnouncementSavedViewPath = typeof OPENAPI_RUNTIME_PATH.putAnnouncementSavedView;
type PutAnnouncementSavedViewOperation = paths[AnnouncementSavedViewPath]['put'];
type PutAnnouncementSavedViewResponse =
  PutAnnouncementSavedViewOperation['responses'][200]['content']['application/json'];
type PutAnnouncementSavedViewResponseData = NonNullable<PutAnnouncementSavedViewResponse['data']>;

type PostAnnouncementsOperation = paths[AnnouncementListPath]['post'];
type PostAnnouncementsEnvelope = PostAnnouncementsOperation['responses'][201]['content']['application/json'];
type PostAnnouncementsData = NonNullable<PostAnnouncementsEnvelope['data']>;
type PostAnnouncementsBody = PostAnnouncementsOperation['requestBody']['content']['application/json'];

type AnnouncementDetailPath = typeof OPENAPI_RUNTIME_PATH.getAnnouncement;
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

type AnnouncementPublishPath = typeof OPENAPI_RUNTIME_PATH.postAnnouncementPublish;
type PostAnnouncementPublishOperation = paths[AnnouncementPublishPath]['post'];
type PostAnnouncementPublishEnvelope =
  PostAnnouncementPublishOperation['responses'][200]['content']['application/json'];
type PostAnnouncementPublishData = NonNullable<PostAnnouncementPublishEnvelope['data']>;
type PostAnnouncementPublishPathParams = PostAnnouncementPublishOperation['parameters']['path'];
type PostAnnouncementPublishBody = NonNullable<
  PostAnnouncementPublishOperation['requestBody']
>['content']['application/json'];

type AnnouncementArchivePath = typeof OPENAPI_RUNTIME_PATH.postAnnouncementArchive;
type PostAnnouncementArchiveOperation = paths[AnnouncementArchivePath]['post'];
type PostAnnouncementArchiveEnvelope =
  PostAnnouncementArchiveOperation['responses'][200]['content']['application/json'];
type PostAnnouncementArchiveData = NonNullable<PostAnnouncementArchiveEnvelope['data']>;
type PostAnnouncementArchivePathParams = PostAnnouncementArchiveOperation['parameters']['path'];

type MyAnnouncementListPath = typeof OPENAPI_RUNTIME_PATH.getMyAnnouncements;
type GetMyAnnouncementsOperation = paths[MyAnnouncementListPath]['get'];
type GetMyAnnouncementsEnvelope = GetMyAnnouncementsOperation['responses'][200]['content']['application/json'];
type GetMyAnnouncementsData = NonNullable<GetMyAnnouncementsEnvelope['data']>;
type GetMyAnnouncementsQuery = NonNullable<GetMyAnnouncementsOperation['parameters']['query']>;

type MyAnnouncementReadPath = typeof OPENAPI_RUNTIME_PATH.postMyAnnouncementRead;
type PostMyAnnouncementReadOperation = paths[MyAnnouncementReadPath]['post'];
type PostMyAnnouncementReadEnvelope = PostMyAnnouncementReadOperation['responses'][200]['content']['application/json'];
type PostMyAnnouncementReadData = NonNullable<PostMyAnnouncementReadEnvelope['data']>;
type PostMyAnnouncementReadPathParams = PostMyAnnouncementReadOperation['parameters']['path'];

type MyAnnouncementReadAllPath = typeof OPENAPI_RUNTIME_PATH.postMyAnnouncementsReadAll;
type PostMyAnnouncementsReadAllOperation = paths[MyAnnouncementReadAllPath]['post'];
type PostMyAnnouncementsReadAllEnvelope =
  PostMyAnnouncementsReadAllOperation['responses'][200]['content']['application/json'];
type PostMyAnnouncementsReadAllData = NonNullable<PostMyAnnouncementsReadAllEnvelope['data']>;

type MyAnnouncementUnreadCountPath = typeof OPENAPI_RUNTIME_PATH.getMyAnnouncementsUnreadCount;
type GetMyAnnouncementsUnreadCountOperation = paths[MyAnnouncementUnreadCountPath]['get'];
type GetMyAnnouncementsUnreadCountEnvelope =
  GetMyAnnouncementsUnreadCountOperation['responses'][200]['content']['application/json'];
type GetMyAnnouncementsUnreadCountData = NonNullable<GetMyAnnouncementsUnreadCountEnvelope['data']>;

export function getAnnouncements(query?: AnnouncementListQuery): Promise<AnnouncementListResponse> {
  return request.get<GetAnnouncementsData>({
    url: OPENAPI_RUNTIME_PATH.getAnnouncements,
    params: normalizeAnnouncementListQuery(query),
  });
}

export async function getAnnouncementSavedViews(): Promise<AnnouncementSavedView[]> {
  const data = await request.get<GetAnnouncementSavedViewsResponseData>({
    url: OPENAPI_RUNTIME_PATH.getAnnouncementSavedViews,
  });
  return data.items;
}

export function postAnnouncementSavedView(payload: AnnouncementSavedViewRequest) {
  return request.post<PostAnnouncementSavedViewResponseData>({
    url: OPENAPI_RUNTIME_PATH.postAnnouncementSavedView,
    data: payload,
  }) as Promise<AnnouncementSavedView>;
}

export function putAnnouncementSavedView(viewId: number, payload: AnnouncementSavedViewRequest) {
  return request.put<PutAnnouncementSavedViewResponseData>({
    url: buildOpenApiRuntimePath('putAnnouncementSavedView', { viewId }),
    data: payload,
  }) as Promise<AnnouncementSavedView>;
}

export function deleteAnnouncementSavedView(viewId: number) {
  return request.delete({ url: buildOpenApiRuntimePath('deleteAnnouncementSavedView', { viewId }) });
}

export function createAnnouncement(payload: CreateAnnouncementRequest): Promise<AnnouncementItem> {
  return request.post<PostAnnouncementsData>({
    url: OPENAPI_RUNTIME_PATH.postAnnouncements,
    data: payload as PostAnnouncementsBody,
  });
}

export function getAnnouncement(id: GetAnnouncementPathParams['id']): Promise<AnnouncementItem> {
  return request.get<GetAnnouncementData>({
    url: buildOpenApiRuntimePath('getAnnouncement', { id }),
  });
}

export function updateAnnouncement(
  id: PutAnnouncementPathParams['id'],
  payload: UpdateAnnouncementRequest,
): Promise<AnnouncementItem> {
  return request.put<PutAnnouncementData>({
    url: buildOpenApiRuntimePath('putAnnouncement', { id }),
    data: payload as PutAnnouncementBody,
  });
}

export function publishAnnouncement(
  id: PostAnnouncementPublishPathParams['id'],
  payload?: PublishAnnouncementRequest,
): Promise<AnnouncementItem> {
  return request.post<PostAnnouncementPublishData>({
    url: buildOpenApiRuntimePath('postAnnouncementPublish', { id }),
    data: payload as PostAnnouncementPublishBody | undefined,
  });
}

export function archiveAnnouncement(id: PostAnnouncementArchivePathParams['id']): Promise<AnnouncementItem> {
  return request.post<PostAnnouncementArchiveData>({
    url: buildOpenApiRuntimePath('postAnnouncementArchive', { id }),
  });
}

export function getMyAnnouncements(query?: MyAnnouncementListQuery): Promise<AnnouncementListResponse> {
  return request.get<GetMyAnnouncementsData>({
    url: OPENAPI_RUNTIME_PATH.getMyAnnouncements,
    params: normalizeMyAnnouncementListQuery(query),
  });
}

export function markAnnouncementRead(id: PostMyAnnouncementReadPathParams['id']): Promise<AnnouncementItem> {
  return request.post<PostMyAnnouncementReadData>({
    url: buildOpenApiRuntimePath('postMyAnnouncementRead', { id }),
  });
}

export function markAllAnnouncementsRead(): Promise<AnnouncementReadAllResponse> {
  return request.post<PostMyAnnouncementsReadAllData>({
    url: OPENAPI_RUNTIME_PATH.postMyAnnouncementsReadAll,
  });
}

export function getAnnouncementUnreadCount(): Promise<AnnouncementUnreadCountResponse> {
  return request.get<GetMyAnnouncementsUnreadCountData>({
    url: OPENAPI_RUNTIME_PATH.getMyAnnouncementsUnreadCount,
  });
}

export function deleteAnnouncement(id: DeleteAnnouncementPathParams['id']) {
  return request.delete<Record<string, never>>({
    url: buildOpenApiRuntimePath('deleteAnnouncement', { id }),
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
