import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

type ListConnectionsOperation = paths[typeof OPENAPI_RUNTIME_PATH.getRegistries]['get'];
type CreateConnectionOperation = paths[typeof OPENAPI_RUNTIME_PATH.postRegistry]['post'];
type UpdateConnectionOperation = paths[typeof OPENAPI_RUNTIME_PATH.putRegistry]['put'];
type GetConnectionOperation = paths[typeof OPENAPI_RUNTIME_PATH.getRegistry]['get'];
type VerifyConnectionOperation = paths[typeof OPENAPI_RUNTIME_PATH.postRegistryVerify]['post'];
type RepositoryListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getRegistryArtifactRepositories]['get'];
type CreateRepositoryOperation = paths[typeof OPENAPI_RUNTIME_PATH.postRegistryArtifactRepository]['post'];
type UpdateRepositoryOperation = paths[typeof OPENAPI_RUNTIME_PATH.putRegistryArtifactRepository]['put'];
type DeleteRepositoryOperation = paths[typeof OPENAPI_RUNTIME_PATH.deleteRegistryArtifactRepository]['delete'];
type AssignmentListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getRegistryArtifactRepositoryAssignments]['get'];
type GrantAssignmentOperation = paths[typeof OPENAPI_RUNTIME_PATH.postRegistryArtifactRepositoryAssignment]['post'];
type RevokeAssignmentOperation =
  paths[typeof OPENAPI_RUNTIME_PATH.deleteRegistryArtifactRepositoryAssignment]['delete'];
type AssignmentCandidatesOperation =
  paths[typeof OPENAPI_RUNTIME_PATH.getRegistryRepositoryAssignmentCandidates]['get'];

export function getRegistries(query?: ListConnectionsOperation['parameters']['query']) {
  return request.get<NonNullable<ListConnectionsOperation['responses'][200]['content']['application/json']['data']>>({
    url: OPENAPI_RUNTIME_PATH.getRegistries,
    params: query,
  });
}

export function createRegistry(payload: CreateConnectionOperation['requestBody']['content']['application/json']) {
  return request.post<NonNullable<CreateConnectionOperation['responses'][201]['content']['application/json']['data']>>({
    url: OPENAPI_RUNTIME_PATH.postRegistry,
    data: payload,
  });
}

export function getRegistry(connectionRef: GetConnectionOperation['parameters']['path']['connectionRef']) {
  return request.get<NonNullable<GetConnectionOperation['responses'][200]['content']['application/json']['data']>>({
    url: buildOpenApiRuntimePath('getRegistry', { connectionRef }),
  });
}

export function updateRegistry(
  connectionRef: UpdateConnectionOperation['parameters']['path']['connectionRef'],
  payload: UpdateConnectionOperation['requestBody']['content']['application/json'],
) {
  return request.put<NonNullable<UpdateConnectionOperation['responses'][200]['content']['application/json']['data']>>({
    url: buildOpenApiRuntimePath('putRegistry', { connectionRef }),
    data: payload,
  });
}

export function deleteRegistry(connectionRef: string) {
  return request.delete({ url: buildOpenApiRuntimePath('deleteRegistry', { connectionRef }) });
}

export function verifyRegistry(connectionRef: string) {
  return request.post<NonNullable<VerifyConnectionOperation['responses'][200]['content']['application/json']['data']>>({
    url: buildOpenApiRuntimePath('postRegistryVerify', { connectionRef }),
  });
}

export function getRegistryRepositories(connectionRef: string, query?: RepositoryListOperation['parameters']['query']) {
  return request.get<NonNullable<RepositoryListOperation['responses'][200]['content']['application/json']['data']>>({
    url: buildOpenApiRuntimePath('getRegistryArtifactRepositories', { connectionRef }),
    params: query,
  });
}

export function createRegistryRepository(
  connectionRef: string,
  payload: CreateRepositoryOperation['requestBody']['content']['application/json'],
) {
  return request.post<NonNullable<CreateRepositoryOperation['responses'][201]['content']['application/json']['data']>>({
    url: buildOpenApiRuntimePath('postRegistryArtifactRepository', { connectionRef }),
    data: payload,
  });
}

export function updateRegistryRepository(
  connectionRef: string,
  repositoryRef: string,
  payload: UpdateRepositoryOperation['requestBody']['content']['application/json'],
) {
  return request.put<NonNullable<UpdateRepositoryOperation['responses'][200]['content']['application/json']['data']>>({
    url: buildOpenApiRuntimePath('putRegistryArtifactRepository', { connectionRef }),
    params: { repository_ref: repositoryRef } satisfies UpdateRepositoryOperation['parameters']['query'],
    data: payload,
  });
}

export function deleteRegistryRepository(connectionRef: string, repositoryRef: string) {
  return request.delete({
    url: buildOpenApiRuntimePath('deleteRegistryArtifactRepository', { connectionRef }),
    params: { repository_ref: repositoryRef } satisfies DeleteRepositoryOperation['parameters']['query'],
  });
}

export function getRegistryRepositoryAssignments(connectionRef: string, repositoryRef: string) {
  return request.get<NonNullable<AssignmentListOperation['responses'][200]['content']['application/json']['data']>>({
    url: buildOpenApiRuntimePath('getRegistryArtifactRepositoryAssignments', { connectionRef }),
    params: { repository_ref: repositoryRef } satisfies AssignmentListOperation['parameters']['query'],
  });
}

export function getRegistryRepositoryAssignmentCandidates(
  connectionRef: string,
  query: AssignmentCandidatesOperation['parameters']['query'],
) {
  return request.get<
    NonNullable<AssignmentCandidatesOperation['responses'][200]['content']['application/json']['data']>
  >({
    url: buildOpenApiRuntimePath('getRegistryRepositoryAssignmentCandidates', { connectionRef }),
    params: query,
  });
}

export function grantRegistryRepositoryAssignment(
  connectionRef: string,
  repositoryRef: string,
  payload: GrantAssignmentOperation['requestBody']['content']['application/json'],
) {
  return request.post<NonNullable<GrantAssignmentOperation['responses'][201]['content']['application/json']['data']>>({
    url: buildOpenApiRuntimePath('postRegistryArtifactRepositoryAssignment', { connectionRef }),
    params: { repository_ref: repositoryRef } satisfies GrantAssignmentOperation['parameters']['query'],
    data: payload,
  });
}

export function revokeRegistryRepositoryAssignment(connectionRef: string, repositoryRef: string, userId: number) {
  return request.delete({
    url: buildOpenApiRuntimePath('deleteRegistryArtifactRepositoryAssignment', { connectionRef, userId }),
    params: { repository_ref: repositoryRef } satisfies RevokeAssignmentOperation['parameters']['query'],
  });
}
