export function toNotification(error: Error) {
  return {
    messageKey: 'demo.request.failed',
    fallbackMessage: error.message || 'Request failed',
  };
}
