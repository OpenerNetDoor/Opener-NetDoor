interface TauriInvoke {
  <T>(command: string, args?: Record<string, unknown>): Promise<T>;
}

function getInvoke(): TauriInvoke | null {
  const maybe = (window as unknown as { __TAURI__?: { core?: { invoke?: TauriInvoke } } }).__TAURI__;
  return maybe?.core?.invoke ?? null;
}

export async function diagnostics(): Promise<Record<string, unknown>> {
  const invoke = getInvoke();
  if (!invoke) {
    return {
      runtime: "browser",
      note: "tauri invoke not available; returning frontend diagnostics",
      timestamp: new Date().toISOString(),
    };
  }
  return invoke<Record<string, unknown>>("collect_diagnostics");
}
