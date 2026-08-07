export type RealtimeAvailabilityController = {
  close(): void;
  reconnect(): void;
};

const controllers = new Set<RealtimeAvailabilityController>();
let platformAvailable = true;

/** 实时通道只消费平台可达性状态；其自身不得再维护第二套重连 authority。 */
export function setRealtimePlatformAvailable(available: boolean) {
  if (platformAvailable === available) {
    return;
  }
  platformAvailable = available;
  for (const controller of controllers) {
    if (available) controller.reconnect();
    else controller.close();
  }
}

export function isRealtimePlatformAvailable() {
  return platformAvailable;
}

export function registerRealtimeAvailabilityController(controller: RealtimeAvailabilityController) {
  controllers.add(controller);
  if (!platformAvailable) controller.close();
  return () => controllers.delete(controller);
}
