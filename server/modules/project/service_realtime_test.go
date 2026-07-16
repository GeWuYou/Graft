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
	aggregate projectstore.ApplicationAggregate
}

func (r realtimeApplicationRepository) Get(_ context.Context, projectID uint64) (projectstore.ApplicationAggregate, error) {
	if projectID != r.aggregate.Application.ApplicationRecordID {
		return projectstore.ApplicationAggregate{}, projectstore.ErrApplicationNotFound
	}
	return r.aggregate, nil
}

func (r realtimeApplicationRepository) GetByApplicationID(
	_ context.Context,
	applicationID string,
) (projectstore.ApplicationAggregate, error) {
	if applicationID != r.aggregate.Application.ApplicationID {
		return projectstore.ApplicationAggregate{}, projectstore.ErrApplicationNotFound
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
		realtimeApplicationRepository{aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{ApplicationRecordID: 42, ApplicationID: realtimeTestApplicationID},
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
		projectcontract.ApplicationRuntimeTopicPrefix + realtimeTestApplicationID,
		projectcontract.ApplicationLogsTopicPrefix + realtimeTestApplicationID,
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
