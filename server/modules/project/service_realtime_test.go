package project

import (
	"context"
	"errors"
	"testing"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
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

func (r realtimeApplicationRepository) List(context.Context, projectstore.ListQuery) (projectstore.ListResult, error) {
	return projectstore.ListResult{Items: []projectstore.ApplicationAggregate{r.aggregate}, Total: 1}, nil
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

func TestIssueProjectListSummaryRealtimeSubscriptionRequiresAllScope(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name       string
		scope      moduleapi.PermissionScope
		wantErr    error
		wantTicket bool
	}{
		{name: "all scope", scope: moduleapi.PermissionScopeAll, wantTicket: true},
		{name: "owned scope", scope: moduleapi.PermissionScopeOwned, wantErr: realtime.ErrTopicForbidden},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			service, err := NewService(
				realtimeApplicationRepository{aggregate: projectstore.ApplicationAggregate{Application: projectstore.Application{ApplicationRecordID: 42, ApplicationID: realtimeTestApplicationID}}},
				WithAuthorizer(realtimeSubscriptionTestAuthorizer{}),
				WithPermissionScopeResolver(fixedPermissionScopeResolver(testCase.scope)),
				WithRealtime(realtimeauth.NewMemoryService(), realtime.NewHub(), realtime.NewTopicIssuerRegistry()),
			)
			if err != nil {
				t.Fatalf("new service: %v", err)
			}
			t.Cleanup(func() {
				if err := service.Close(context.Background()); err != nil {
					t.Errorf("close service: %v", err)
				}
			})

			auth := moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}}
			ctx := moduleapi.WithRequestAuthContext(context.Background(), auth)
			response, issueErr := service.IssueSubscription(ctx, realtime.SubscriptionRequest{Topic: projectcontract.ApplicationListSummaryTopic, RequestAuth: auth})
			if !errors.Is(issueErr, testCase.wantErr) {
				t.Fatalf("issue subscription error = %v, want %v", issueErr, testCase.wantErr)
			}
			if testCase.wantTicket && (response.Topic != projectcontract.ApplicationListSummaryTopic || response.Ticket == "") {
				t.Fatalf("all-scope response = %#v, want topic and ticket", response)
			}
		})
	}
}

func TestProjectListTopicStreamerPublishesWithoutRequestAuthContext(t *testing.T) {
	t.Parallel()

	hub := realtime.NewHub()
	service, err := NewService(
		realtimeApplicationRepository{aggregate: projectstore.ApplicationAggregate{Application: projectstore.Application{ApplicationRecordID: 42, ApplicationID: realtimeTestApplicationID}}},
		WithPermissionScopeResolver(fixedPermissionScopeResolver(moduleapi.PermissionScopeOwned)),
		WithRealtime(realtimeauth.NewMemoryService(), hub, realtime.NewTopicIssuerRegistry()),
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	events, unsubscribe := hub.Subscribe(projectcontract.ApplicationListSummaryTopic)
	defer unsubscribe()
	streamer, err := newProjectListTopicStreamer(hub, nil, service)
	if err != nil {
		t.Fatalf("new list topic streamer: %v", err)
	}
	streamer.publish(context.Background(), projectcontract.ApplicationListSummaryTopic)

	select {
	case event := <-events:
		payload, ok := event.Data.(projectListSummaryRealtimePayload)
		if !ok {
			t.Fatalf("payload type = %T", event.Data)
		}
		if len(payload.Items) != 1 || payload.Items[0].ApplicationID != realtimeTestApplicationID || payload.Items[0].RuntimeStatus != generated.ApplicationRuntimeStatusUnknown {
			t.Fatalf("payload = %#v", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("expected published list summary snapshot")
	}
}
