const resourceTypeLabelKeys = {
  scheduled_task_run: 'notification.resourceType.scheduledTaskRun',
} as const;

type NotificationResourceType = keyof typeof resourceTypeLabelKeys;
type Translate = (key: string) => string;

export function presentNotification(item: { message: string; resource_type: string; title: string }, t: Translate) {
  const labelKey = resourceTypeLabelKeys[item.resource_type as NotificationResourceType];

  return {
    message: item.message,
    resourceTypeLabel: labelKey ? t(labelKey) : '',
    title: item.title,
  };
}
