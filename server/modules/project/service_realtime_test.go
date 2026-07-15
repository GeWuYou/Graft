package project

import (
	"context"
	"testing"

	"graft/server/internal/moduleapi"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

const realtimeTestApplicationID = "app_01ARZ3NDEKTSV4RRFFQ69G5FAV"

type realtimeApplicationRepository struct {
	projectstore.Repository
	aggregate projectstore.ProjectAggregate
}

func (r realtimeApplicationRepository) Get(_ context.Context, projectID uint64) (projectstore.ProjectAggregate, error) {
	if projectID != r.aggregate.Project.ID {
		return projectstore.ProjectAggregate{}, projectstore.ErrProjectNotFound
	}
	return r.aggregate, nil
}

func (r realtimeApplicationRepository) GetByApplicationID(
	_ context.Context,
	applicationID string,
) (projectstore.ProjectAggregate, error) {
	if applicationID != r.aggregate.Project.ApplicationID {
		return projectstore.ProjectAggregate{}, projectstore.ErrProjectNotFound
	}
	return r.aggregate, nil
}

type realtimeSubscriptionTestAuthorizer struct{}

func (realtimeSubscriptionTestAuthorizer) Authorize(context.Context, moduleapi.RequestAuthContext, string) error {
	return nil
}

func TestIssueProjectRealtimeSubscriptionsUsePublicApplicationID(t *testing.T) {
	t.Parallel()

	service, err := NewService(
		realtimeApplicationRepository{aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{ID: 42, ApplicationID: realtimeTestApplicationID},
		}},
		WithAuthorizer(realtimeSubscriptionTestAuthorizer{}),
		WithRealtime(realtimeauth.NewMemoryService(), realtime.NewHub(), realtime.NewTopicIssuerRegistry()),
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer func() {
		if closeErr := service.Close(context.Background()); closeErr != nil {
			t.Errorf("close service: %v", closeErr)
		}
	}()

	for _, topic := range []string{
		projectcontract.ProjectRuntimeTopicPrefix + realtimeTestApplicationID,
		projectcontract.ProjectLogsTopicPrefix + realtimeTestApplicationID,
	} {
		response, issueErr := service.IssueSubscription(context.Background(), realtime.SubscriptionRequest{
			Topic: topic,
			RequestAuth: moduleapi.RequestAuthContext{
				User: &moduleapi.CurrentUser{ID: 7},
			},
		})
		if issueErr != nil {
			t.Fatalf("issue subscription for %q: %v", topic, issueErr)
		}
		if response.Topic != topic {
			t.Fatalf("expected topic %q, got %q", topic, response.Topic)
		}
		if response.Ticket == "" || response.WebSocketURL == "" {
			t.Fatalf("expected ticket and websocket URL, got %#v", response)
		}
	}
}
