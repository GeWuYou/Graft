export function presentNotification(item: { message: string; resource_type: string; title: string }) {
  return {
    message: item.message,
    resourceTypeLabel: item.resource_type ? 'Resource' : '',
    title: item.title,
  };
}
