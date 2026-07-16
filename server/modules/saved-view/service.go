package savedview

import (
	"context"

	"graft/server/internal/moduleapi"
	"graft/server/modules/saved-view/store"
)

// Service 校验通用保存视图的边界输入，并将消费者自定义的查询状态原样委托给仓储。
type Service struct{ repository store.Repository }

// NewService 创建一个已保存视图服务；仓储为空时服务调用返回 ErrSavedViewInvalidInput。
func NewService(repository store.Repository) *Service { return &Service{repository: repository} }

// List 返回指定用户和消费界面拥有的未删除视图；服务未配置仓储时返回 ErrSavedViewInvalidInput。
func (s *Service) List(ctx context.Context, ownerUserID uint64, surfaceKey string) ([]moduleapi.SavedView, error) {
	if s == nil || s.repository == nil {
		return nil, moduleapi.ErrSavedViewInvalidInput
	}
	return s.repository.List(ctx, ownerUserID, surfaceKey)
}

// Create 持久化一个由消费者提供并由仓储校验的视图。
func (s *Service) Create(ctx context.Context, input moduleapi.SavedViewCreateInput) (moduleapi.SavedView, error) {
	if s == nil || s.repository == nil {
		return moduleapi.SavedView{}, moduleapi.ErrSavedViewInvalidInput
	}
	return s.repository.Create(ctx, input)
}

// Update 替换一个由指定用户和消费界面共同拥有的视图；缺少 ID、用户或界面标识时提前拒绝。
func (s *Service) Update(ctx context.Context, input moduleapi.SavedViewUpdateInput) (moduleapi.SavedView, error) {
	if s == nil || s.repository == nil {
		return moduleapi.SavedView{}, moduleapi.ErrSavedViewInvalidInput
	}
	if input.ID == 0 || input.OwnerUserID == 0 || input.SurfaceKey == "" {
		return moduleapi.SavedView{}, moduleapi.ErrSavedViewInvalidInput
	}
	return s.repository.Update(ctx, input)
}

// Delete 软删除一个由指定用户和消费界面共同拥有的视图。
func (s *Service) Delete(ctx context.Context, ownerUserID uint64, surfaceKey string, id uint64) error {
	if s == nil || s.repository == nil {
		return moduleapi.ErrSavedViewInvalidInput
	}
	if ownerUserID == 0 || surfaceKey == "" || id == 0 {
		return moduleapi.ErrSavedViewInvalidInput
	}
	return s.repository.Delete(ctx, ownerUserID, surfaceKey, id)
}
