package notification

import notificationstore "graft/server/modules/notification/store"

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

func runNotificationRepository(service *Service, query func(notificationstore.Repository) error) error {
	_, err := withNotificationRepository(service, func(repository notificationstore.Repository) (struct{}, error) {
		return struct{}{}, query(repository)
	})
	return err
}
