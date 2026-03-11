"use client";

const ADMIN_DATA_CHANGED_EVENT = "opener-netdoor:admin-data-changed";

export function emitAdminDataChanged(): void {
  if (typeof window === "undefined") {
    return;
  }
  window.dispatchEvent(new Event(ADMIN_DATA_CHANGED_EVENT));
}

export function subscribeAdminDataChanged(handler: () => void): () => void {
  if (typeof window === "undefined") {
    return () => undefined;
  }
  const listener = () => handler();
  window.addEventListener(ADMIN_DATA_CHANGED_EVENT, listener);
  return () => window.removeEventListener(ADMIN_DATA_CHANGED_EVENT, listener);
}
