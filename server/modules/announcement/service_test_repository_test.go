// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package announcement

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	announcementstore "graft/server/modules/announcement/store"
)

type memoryAnnouncementRepository struct {
	mu         sync.Mutex
	nextID     uint64
	nextReadID uint64
	items      map[uint64]announcementstore.Announcement
	reads      map[uint64]map[uint64]announcementstore.AnnouncementRead
}

func newMemoryAnnouncementRepository() *memoryAnnouncementRepository {
	return &memoryAnnouncementRepository{
		nextID:     1,
		nextReadID: 1,
		items:      make(map[uint64]announcementstore.Announcement),
		reads:      make(map[uint64]map[uint64]announcementstore.AnnouncementRead),
	}
}

func (r *memoryAnnouncementRepository) Ping(context.Context) error {
	return nil
}

func (r *memoryAnnouncementRepository) ListAdmin(_ context.Context, query announcementstore.ListQuery) (announcementstore.ListResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]announcementstore.Announcement, 0, len(r.items))
	for _, item := range r.items {
		if !memoryMatchesAdminQuery(item, query) {
			continue
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i int, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return items[i].ID > items[j].ID
	})
	paged, total := memoryPaginate(items, query.Offset, query.Limit)
	return announcementstore.ListResult{Items: paged, Total: total}, nil
}

func (r *memoryAnnouncementRepository) ListCurrentUser(_ context.Context, query announcementstore.UserListQuery) (announcementstore.UserListResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]announcementstore.UserAnnouncement, 0, len(r.items))
	for _, item := range r.items {
		if !memoryVisibleAnnouncement(item, query.Now) {
			continue
		}
		userItem := announcementstore.UserAnnouncement{
			Announcement: item,
			ReadAt:       r.readAtLocked(item.ID, query.UserID),
		}
		if query.UnreadOnly && userItem.ReadAt != nil {
			continue
		}
		items = append(items, userItem)
	}
	sort.SliceStable(items, func(i int, j int) bool {
		if items[i].Announcement.Pinned != items[j].Announcement.Pinned {
			return items[i].Announcement.Pinned
		}
		left := publishSortTime(items[i].Announcement)
		right := publishSortTime(items[j].Announcement)
		if !left.Equal(right) {
			return left.After(right)
		}
		return items[i].Announcement.ID > items[j].Announcement.ID
	})
	total := len(items)
	start := query.Offset
	if start > total {
		start = total
	}
	end := start + query.Limit
	if end > total {
		end = total
	}
	return announcementstore.UserListResult{Items: append([]announcementstore.UserAnnouncement(nil), items[start:end]...), Total: total}, nil
}

func memoryMatchesAdminQuery(item announcementstore.Announcement, query announcementstore.ListQuery) bool {
	if item.DeletedAt != 0 {
		return false
	}
	if query.Status != "" && item.Status != query.Status {
		return false
	}
	if query.Level != "" && item.Level != query.Level {
		return false
	}
	if query.Pinned != nil && item.Pinned != *query.Pinned {
		return false
	}
	if query.Keyword != "" && !strings.Contains(strings.ToLower(item.Title+" "+item.Content), strings.ToLower(query.Keyword)) {
		return false
	}
	return true
}

func memoryPaginate[T any](items []T, offset int, limit int) ([]T, int) {
	total := len(items)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	return append([]T(nil), items[start:end]...), total
}

func (r *memoryAnnouncementRepository) Create(_ context.Context, input announcementstore.CreateInput) (announcementstore.Announcement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	item := announcementstore.Announcement{
		ID:           r.nextID,
		Title:        strings.TrimSpace(input.Title),
		Content:      strings.TrimSpace(input.Content),
		Level:        strings.TrimSpace(input.Level),
		Status:       strings.TrimSpace(input.Status),
		DeliveryMode: strings.TrimSpace(input.DeliveryMode),
		Pinned:       input.Pinned,
		PublishAt:    cloneTime(input.PublishAt),
		PublishedBy:  nil,
		ArchivedAt:   nil,
		ExpireAt:     cloneTime(input.ExpireAt),
		CreatedBy:    cloneUint64(input.ActorID),
		UpdatedBy:    cloneUint64(input.ActorID),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	r.nextID++
	r.items[item.ID] = item
	return item, nil
}

func (r *memoryAnnouncementRepository) GetAdmin(_ context.Context, id uint64) (announcementstore.Announcement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok || item.DeletedAt != 0 {
		return announcementstore.Announcement{}, announcementstore.ErrAnnouncementNotFound
	}
	return item, nil
}

func (r *memoryAnnouncementRepository) Update(_ context.Context, id uint64, input announcementstore.UpdateInput) (announcementstore.Announcement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok || item.DeletedAt != 0 {
		return announcementstore.Announcement{}, announcementstore.ErrAnnouncementNotFound
	}
	item.Title = strings.TrimSpace(input.Title)
	item.Content = strings.TrimSpace(input.Content)
	item.Level = strings.TrimSpace(input.Level)
	item.DeliveryMode = strings.TrimSpace(input.DeliveryMode)
	item.Pinned = input.Pinned
	item.PublishAt = cloneTime(input.PublishAt)
	item.ExpireAt = cloneTime(input.ExpireAt)
	item.UpdatedBy = cloneUint64(input.ActorID)
	item.UpdatedAt = time.Now().UTC()
	r.items[id] = item
	return item, nil
}

func (r *memoryAnnouncementRepository) Publish(_ context.Context, id uint64, publishAt time.Time, actorID *uint64) (announcementstore.Announcement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok || item.DeletedAt != 0 {
		return announcementstore.Announcement{}, announcementstore.ErrAnnouncementNotFound
	}
	item.Status = "published"
	publishAt = publishAt.UTC()
	item.PublishAt = &publishAt
	item.PublishedBy = cloneUint64(actorID)
	item.ArchivedAt = nil
	item.UpdatedBy = cloneUint64(actorID)
	item.UpdatedAt = time.Now().UTC()
	r.items[id] = item
	return item, nil
}

func (r *memoryAnnouncementRepository) Archive(_ context.Context, id uint64, actorID *uint64) (announcementstore.Announcement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok || item.DeletedAt != 0 {
		return announcementstore.Announcement{}, announcementstore.ErrAnnouncementNotFound
	}
	item.Status = "archived"
	archivedAt := time.Now().UTC()
	item.ArchivedAt = &archivedAt
	item.UpdatedBy = cloneUint64(actorID)
	item.UpdatedAt = time.Now().UTC()
	r.items[id] = item
	return item, nil
}

func (r *memoryAnnouncementRepository) Delete(_ context.Context, id uint64, actorID uint64, deletedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok || item.DeletedAt != 0 {
		return announcementstore.ErrAnnouncementNotFound
	}
	item.DeletedBy = cloneUint64(&actorID)
	item.DeletedAt = deletedAt.UTC().Unix()
	item.UpdatedBy = cloneUint64(&actorID)
	item.UpdatedAt = deletedAt.UTC()
	r.items[id] = item
	return nil
}

func (r *memoryAnnouncementRepository) MarkRead(_ context.Context, userID uint64, announcementID uint64, readAt time.Time) (announcementstore.UserAnnouncement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[announcementID]
	if !ok || !memoryVisibleAnnouncement(item, time.Now().UTC()) {
		return announcementstore.UserAnnouncement{}, announcementstore.ErrAnnouncementNotFound
	}
	if _, ok := r.reads[userID]; !ok {
		r.reads[userID] = make(map[uint64]announcementstore.AnnouncementRead)
	}
	if _, exists := r.reads[userID][announcementID]; !exists {
		r.reads[userID][announcementID] = announcementstore.AnnouncementRead{
			ID:             r.nextReadID,
			AnnouncementID: announcementID,
			UserID:         userID,
			ReadAt:         readAt.UTC(),
			CreatedAt:      readAt.UTC(),
		}
		r.nextReadID++
	}
	return announcementstore.UserAnnouncement{
		Announcement: item,
		ReadAt:       r.readAtLocked(announcementID, userID),
	}, nil
}

func (r *memoryAnnouncementRepository) MarkAllRead(_ context.Context, userID uint64, readAt time.Time, now time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.reads[userID]; !ok {
		r.reads[userID] = make(map[uint64]announcementstore.AnnouncementRead)
	}
	count := 0
	for _, item := range r.items {
		if !memoryVisibleAnnouncement(item, now) {
			continue
		}
		if _, exists := r.reads[userID][item.ID]; exists {
			continue
		}
		r.reads[userID][item.ID] = announcementstore.AnnouncementRead{
			ID:             r.nextReadID,
			AnnouncementID: item.ID,
			UserID:         userID,
			ReadAt:         readAt.UTC(),
			CreatedAt:      readAt.UTC(),
		}
		r.nextReadID++
		count++
	}
	return count, nil
}

func (r *memoryAnnouncementRepository) UnreadCount(_ context.Context, userID uint64, now time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, item := range r.items {
		if !memoryVisibleAnnouncement(item, now) || r.readAtLocked(item.ID, userID) != nil {
			continue
		}
		count++
	}
	return count, nil
}

func (r *memoryAnnouncementRepository) readAtLocked(announcementID uint64, userID uint64) *time.Time {
	userReads, ok := r.reads[userID]
	if !ok {
		return nil
	}
	read, ok := userReads[announcementID]
	if !ok {
		return nil
	}
	readAt := read.ReadAt.UTC()
	return &readAt
}

func memoryVisibleAnnouncement(item announcementstore.Announcement, now time.Time) bool {
	if item.DeletedAt != 0 || item.Status != "published" || item.PublishAt == nil || item.PublishAt.After(now.UTC()) {
		return false
	}
	return item.ExpireAt == nil || item.ExpireAt.After(now.UTC())
}

func publishSortTime(item announcementstore.Announcement) time.Time {
	if item.PublishAt == nil {
		return time.Time{}
	}
	return item.PublishAt.UTC()
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func cloneUint64(value *uint64) *uint64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

var _ announcementstore.Repository = (*memoryAnnouncementRepository)(nil)
