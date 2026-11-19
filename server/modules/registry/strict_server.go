package registry

import (
	"context"
	"github.com/gin-gonic/gin"
	openapigen "graft/server/internal/contract/openapi/generated"
	registryopenapi "graft/server/internal/contract/openapi/registry"
	"graft/server/internal/contract/openapi/strictsupport"
	"graft/server/internal/module"
)

// registryStrictServer 将现有 HTTP 响应写入接入生成的请求契约。
type registryStrictServer struct {
	module  *module.Context
	service *Service
}

var _ registryopenapi.StrictServerInterface = registryStrictServer{}

func (s registryStrictServer) GetRegistryAvailableDestinations(ctx context.Context, _ registryopenapi.GetRegistryAvailableDestinationsRequestObject) (registryopenapi.GetRegistryAvailableDestinationsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleAvailableDestinations)
}

func (s registryStrictServer) GetRegistries(ctx context.Context, _ registryopenapi.GetRegistriesRequestObject) (registryopenapi.GetRegistriesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleListConnections)
}

func (s registryStrictServer) PostRegistry(ctx context.Context, _ registryopenapi.PostRegistryRequestObject) (registryopenapi.PostRegistryResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleCreateConnection)
}

func (s registryStrictServer) GetRegistry(ctx context.Context, _ registryopenapi.GetRegistryRequestObject) (registryopenapi.GetRegistryResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleGetConnection)
}

func (s registryStrictServer) PutRegistry(ctx context.Context, req registryopenapi.PutRegistryRequestObject) (registryopenapi.PutRegistryResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.handleUpdateConnection(c, (*openapigen.PutRegistryJSONRequestBody)(req.Body)) })
}

func (s registryStrictServer) DeleteRegistry(ctx context.Context, _ registryopenapi.DeleteRegistryRequestObject) (registryopenapi.DeleteRegistryResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleDeleteConnection)
}

func (s registryStrictServer) PostRegistryVerify(ctx context.Context, req registryopenapi.PostRegistryVerifyRequestObject) (registryopenapi.PostRegistryVerifyResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) {
		s.handleVerifyConnection(c, (*openapigen.PostRegistryVerifyJSONRequestBody)(req.Body))
	})
}

func (s registryStrictServer) GetRegistryArtifactRepositories(ctx context.Context, _ registryopenapi.GetRegistryArtifactRepositoriesRequestObject) (registryopenapi.GetRegistryArtifactRepositoriesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleListRepositories)
}

func (s registryStrictServer) PostRegistryArtifactRepository(ctx context.Context, req registryopenapi.PostRegistryArtifactRepositoryRequestObject) (registryopenapi.PostRegistryArtifactRepositoryResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) {
		s.handleCreateRepository(c, (*openapigen.PostRegistryArtifactRepositoryJSONRequestBody)(req.Body))
	})
}

func (s registryStrictServer) PutRegistryArtifactRepository(ctx context.Context, req registryopenapi.PutRegistryArtifactRepositoryRequestObject) (registryopenapi.PutRegistryArtifactRepositoryResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) {
		s.handleUpdateRepository(c, (*openapigen.PutRegistryArtifactRepositoryJSONRequestBody)(req.Body))
	})
}

func (s registryStrictServer) DeleteRegistryArtifactRepository(ctx context.Context, _ registryopenapi.DeleteRegistryArtifactRepositoryRequestObject) (registryopenapi.DeleteRegistryArtifactRepositoryResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleDeleteRepository)
}

func (s registryStrictServer) GetRegistryRepositoryAssignmentCandidates(ctx context.Context, _ registryopenapi.GetRegistryRepositoryAssignmentCandidatesRequestObject) (registryopenapi.GetRegistryRepositoryAssignmentCandidatesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleListAssignmentCandidates)
}

func (s registryStrictServer) GetRegistryArtifactRepositoryAssignments(ctx context.Context, _ registryopenapi.GetRegistryArtifactRepositoryAssignmentsRequestObject) (registryopenapi.GetRegistryArtifactRepositoryAssignmentsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleListAssignments)
}

func (s registryStrictServer) PostRegistryArtifactRepositoryAssignment(ctx context.Context, req registryopenapi.PostRegistryArtifactRepositoryAssignmentRequestObject) (registryopenapi.PostRegistryArtifactRepositoryAssignmentResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) {
		s.handleGrantAssignment(c, (*openapigen.PostRegistryArtifactRepositoryAssignmentJSONRequestBody)(req.Body))
	})
}

func (s registryStrictServer) PutRegistryArtifactRepositoryAssignments(ctx context.Context, req registryopenapi.PutRegistryArtifactRepositoryAssignmentsRequestObject) (registryopenapi.PutRegistryArtifactRepositoryAssignmentsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) {
		s.handleReplaceAssignments(c, (*openapigen.PutRegistryArtifactRepositoryAssignmentsJSONRequestBody)(req.Body))
	})
}

func (s registryStrictServer) PostRegistryArtifactRepositoryAssignmentsBatchAdd(ctx context.Context, req registryopenapi.PostRegistryArtifactRepositoryAssignmentsBatchAddRequestObject) (registryopenapi.PostRegistryArtifactRepositoryAssignmentsBatchAddResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) {
		s.handleAddAssignments(c, (*openapigen.PostRegistryArtifactRepositoryAssignmentsBatchAddJSONRequestBody)(req.Body))
	})
}

func (s registryStrictServer) PostRegistryArtifactRepositoryAssignmentsBatchRevoke(ctx context.Context, req registryopenapi.PostRegistryArtifactRepositoryAssignmentsBatchRevokeRequestObject) (registryopenapi.PostRegistryArtifactRepositoryAssignmentsBatchRevokeResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) {
		s.handleRevokeAssignments(c, (*openapigen.PostRegistryArtifactRepositoryAssignmentsBatchRevokeJSONRequestBody)(req.Body))
	})
}

func (s registryStrictServer) DeleteRegistryArtifactRepositoryAssignment(ctx context.Context, _ registryopenapi.DeleteRegistryArtifactRepositoryAssignmentRequestObject) (registryopenapi.DeleteRegistryArtifactRepositoryAssignmentResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handleRevokeAssignment)
}
