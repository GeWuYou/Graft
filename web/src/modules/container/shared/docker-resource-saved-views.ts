import type { ResourceQueryState } from '@/shared/components/query-list';
import {
  normalizeSavedQueryView,
  type PersistedSavedQueryView,
  type SavedQueryViewAdapter,
  type SavedQueryViewController,
  type SavedQueryViewInput,
  type SavedQueryViewOperation,
  serializeSavedQueryViewRequest,
  useSavedQueryViews,
} from '@/shared/components/query-list';

export type DockerResourceSavedViewState = {
  pageSize: number;
  queryState: ResourceQueryState;
  visibleColumns: string[];
};

export type DockerResourceSavedViewApi = {
  list: () => Promise<PersistedSavedQueryView<number>[]>;
  create: (payload: ReturnType<typeof serializeSavedQueryViewRequest>) => Promise<PersistedSavedQueryView<number>>;
  update: (
    id: number,
    payload: ReturnType<typeof serializeSavedQueryViewRequest>,
  ) => Promise<PersistedSavedQueryView<number>>;
  remove: (id: number) => Promise<void>;
};

function createDockerResourceSavedViewAdapter(
  api: DockerResourceSavedViewApi,
): SavedQueryViewAdapter<DockerResourceSavedViewState, number> {
  return {
    list: async () => (await api.list()).map((view) => normalizeSavedQueryView<ResourceQueryState, number>(view)),
    create: async (input: SavedQueryViewInput<DockerResourceSavedViewState>) =>
      normalizeSavedQueryView<ResourceQueryState, number>(await api.create(serializeSavedQueryViewRequest(input))),
    update: async (id: number, input: SavedQueryViewInput<DockerResourceSavedViewState>) =>
      normalizeSavedQueryView<ResourceQueryState, number>(await api.update(id, serializeSavedQueryViewRequest(input))),
    remove: api.remove,
  };
}

export function useDockerResourceSavedViews(options: {
  api: DockerResourceSavedViewApi;
  applyState: (state: DockerResourceSavedViewState) => void;
  getState: () => DockerResourceSavedViewState;
  onError?: (error: unknown, operation: SavedQueryViewOperation) => void;
}): SavedQueryViewController<DockerResourceSavedViewState, number> {
  return useSavedQueryViews<DockerResourceSavedViewState, number>({
    adapter: createDockerResourceSavedViewAdapter(options.api),
    applyView: (view) => options.applyState(view.state),
    onError: options.onError,
    serializeCurrentState: options.getState,
  });
}
