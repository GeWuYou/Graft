package notification

import notificationstore "graft/server/modules/notification/store"

// withNotificationRepository 获取通知仓库并执行查询，统一处理仓库获取和存储错误。
// 查询成功时返回其结果；查询失败时返回 T 的零值和映射后的错误。
func withNotificationRepository[T any](
	service *Service,
	query func(notificationstore.Repository) (T, error),
) (T, error) {
	var zero T
	repository, err := service.repositoryOrErr()
	if err != nil {
		return zero, err
	}

	result, err := query(repository)
	if err != nil {
		return zero, mapStoreError(err)
	}

	return result, nil
}

// runNotificationRepository 执行一个仅返回错误的通知仓库操作，并返回统一处理后的错误。
func runNotificationRepository(service *Service, query func(notificationstore.Repository) error) error {
	_, err := withNotificationRepository(service, func(repository notificationstore.Repository) (struct{}, error) {
		return struct{}{}, query(repository)
	})
	return err
}
