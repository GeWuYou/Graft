package notificationopenapi

//go:generate go tool oapi-codegen --include-operation-ids getNotifications,getNotificationsUnreadCount,postNotificationRead,postNotificationsReadAll,deleteNotification --generate types --package notificationopenapi -o zz_generated.notification.go ../../../../../openapi/openapi.yaml
