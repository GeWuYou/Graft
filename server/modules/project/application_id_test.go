package project

import (
	"context"
	"regexp"
	"testing"

	projectstore "graft/server/modules/project/store"
)

func TestNewApplicationIDUsesPublicULIDFormat(t *testing.T) {
	t.Parallel()
	value := newApplicationID()
	if !regexp.MustCompile(`^app_[0-9A-HJKMNP-TV-Z]{26}$`).MatchString(value) {
		t.Fatalf("application id %q does not use app_<ULID> format", value)
	}
}

type applicationLookupRepository struct {
	projectstore.Repository
	aggregate projectstore.ProjectAggregate
}

func (r applicationLookupRepository) GetByApplicationID(_ context.Context, applicationID string) (projectstore.ProjectAggregate, error) {
	if applicationID != r.aggregate.Project.ApplicationID {
		return projectstore.ProjectAggregate{}, projectstore.ErrProjectNotFound
	}
	return r.aggregate, nil
}

func TestResolveApplicationIDUsesPublicLookup(t *testing.T) {
	t.Parallel()
	applicationID := newApplicationID()
	service, err := NewService(applicationLookupRepository{aggregate: projectstore.ProjectAggregate{Project: projectstore.Project{ID: 42, ApplicationID: applicationID}}})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	projectID, err := service.ResolveApplicationID(context.Background(), applicationID)
	if err != nil || projectID != 42 {
		t.Fatalf("resolve application id = %d, %v", projectID, err)
	}
	if _, err := service.ResolveApplicationID(context.Background(), "42"); err == nil {
		t.Fatal("numeric project id must not be accepted as a public application id")
	}
}
