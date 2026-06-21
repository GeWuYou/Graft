package announcement

import (
	"context"
	"errors"
	"testing"
	"time"

	announcementcontract "graft/server/modules/announcement/contract"
	announcementstore "graft/server/modules/announcement/store"
)

func TestAnnouncementPublishWithoutEffectiveTimeRejectsExpiredAnnouncement(t *testing.T) {
	repository := newMemoryAnnouncementRepository()
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	actorID := uint64(7)
	expireAt := time.Now().UTC().Add(-time.Minute)
	created, err := service.Create(ctx, announcementstore.CreateInput{
		Title:    "Expired immediate",
		Content:  "Expired immediate",
		Level:    announcementcontract.AnnouncementLevelInfo.String(),
		ExpireAt: &expireAt,
		ActorID:  &actorID,
	})
	if err != nil {
		t.Fatalf("create announcement: %v", err)
	}
	if _, err := service.Publish(ctx, created.ID, nil, &actorID); !errors.Is(err, errAnnouncementInvalidInput) {
		t.Fatalf("expected expired immediate publish guard, got %v", err)
	}
}
