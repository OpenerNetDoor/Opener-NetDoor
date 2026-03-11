import { createLocalPlannerServerOperationsAdapter, type ServerOperationsAdapter } from "@opener-netdoor/sdk-ts";

let singleton: ServerOperationsAdapter | null = null;

export function getServerOperationsAdapter(): ServerOperationsAdapter {
  if (!singleton) {
    singleton = createLocalPlannerServerOperationsAdapter();
  }
  return singleton;
}
