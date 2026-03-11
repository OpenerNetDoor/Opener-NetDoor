const PREFIX = "ond:secure:";

export async function setSecret(key: string, value: string): Promise<void> {
  // TODO: replace localStorage fallback with Tauri secure storage plugin or OS keychain.
  window.localStorage.setItem(PREFIX + key, value);
}

export async function getSecret(key: string): Promise<string | null> {
  return window.localStorage.getItem(PREFIX + key);
}
