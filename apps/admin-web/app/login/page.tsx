"use client";

import { FormEvent, useEffect, useState } from "react";
import type { ThemeMode } from "@opener-netdoor/shared-types";
import { ensureBaseURL } from "../../lib/api/client";
import { getTheme, hydrateSession, setTheme } from "../../lib/auth/session";
import { Card, PageTitle } from "../../components/ui";

export default function LoginPage() {
  const [baseUrl, setBaseUrl] = useState(process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080");
  const [magicUrl, setMagicUrl] = useState("");
  const [theme, setThemeMode] = useState<ThemeMode>("dark");
  const [checking, setChecking] = useState(false);
  const [message, setMessage] = useState("Open your installer-provided admin access URL to create a session.");

  useEffect(() => {
    setThemeMode(getTheme());
    setChecking(true);
    void hydrateSession(ensureBaseURL(baseUrl))
      .then((session) => {
        if (session) {
          window.location.href = "/dashboard";
          return;
        }
        setMessage("No active session. Open admin access URL from installer output.");
      })
      .catch(() => setMessage("No active session. Open admin access URL from installer output."))
      .finally(() => setChecking(false));
  }, []);

  const onCheckSession = async (event: FormEvent) => {
    event.preventDefault();
    setTheme(theme);
    setChecking(true);
    const session = await hydrateSession(ensureBaseURL(baseUrl)).catch(() => null);
    setChecking(false);
    if (session) {
      window.location.href = "/dashboard";
      return;
    }
    setMessage("Session is not active yet. Open the admin magic URL first.");
  };

  const onOpenMagic = (event: FormEvent) => {
    event.preventDefault();
    if (!magicUrl.trim()) {
      setMessage("Paste your admin access URL from installer summary.");
      return;
    }
    window.location.href = magicUrl.trim();
  };

  return (
    <main style={{ minHeight: "100vh", display: "grid", placeItems: "center", padding: 16 }}>
      <div style={{ width: "min(860px, 100%)", display: "grid", gap: 16 }}>
        <PageTitle title="Opener NetDoor" subtitle="Single-owner admin access" />

        <div className="grid-two">
          <Card title="Open admin panel" subtitle="Use magic URL generated during install.">
            <form onSubmit={onOpenMagic} className="row">
              <label>
                Admin access URL
                <input
                  value={magicUrl}
                  onChange={(event) => setMagicUrl(event.target.value)}
                  placeholder="https://HOST/ADMIN_SECRET/OWNER_UUID/"
                />
              </label>
              <div style={{ width: "100%", display: "flex", justifyContent: "flex-end" }}>
                <button className="nd-btn is-primary" type="submit">
                  Open access URL
                </button>
              </div>
            </form>
          </Card>

          <Card title="Session check" subtitle="Cookie-based auth; JWT paste is disabled.">
            <form onSubmit={onCheckSession} className="row">
              <label>
                API base URL
                <input value={baseUrl} onChange={(event) => setBaseUrl(event.target.value)} required />
              </label>
              <label>
                Theme
                <select value={theme} onChange={(event) => setThemeMode(event.target.value as ThemeMode)}>
                  <option value="dark">dark</option>
                  <option value="light">light</option>
                  <option value="system">system</option>
                </select>
              </label>
              <div style={{ width: "100%", display: "flex", justifyContent: "flex-end" }}>
                <button className="nd-btn is-secondary" type="submit" disabled={checking}>
                  {checking ? "Checking..." : "Check session"}
                </button>
              </div>
            </form>
            <p>{message}</p>
          </Card>
        </div>
      </div>
    </main>
  );
}

