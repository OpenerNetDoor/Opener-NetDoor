"use client";

import { FormEvent, useState } from "react";
import type { Scope, ThemeMode } from "@opener-netdoor/shared-types";
import { setDefaultScopeId, setSession, setTheme } from "../../lib/auth/session";
import { ensureBaseURL } from "../../lib/api/client";
import { Card, PageTitle, StatusBadge } from "../../components/ui";

const defaultScopes = "admin:read,admin:write";
const defaultScope = process.env.NEXT_PUBLIC_DEFAULT_SCOPE_ID ?? "default";

export default function LoginPage() {
  const [baseUrl, setBaseUrl] = useState(process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080");
  const [subject, setSubject] = useState("owner");
  const [token, setToken] = useState("");
  const [scopeId, setScopeId] = useState(defaultScope);
  const [scopes, setScopes] = useState(defaultScopes);
  const [theme, setThemeMode] = useState<ThemeMode>("dark");

  const onSubmit = (event: FormEvent) => {
    event.preventDefault();
    const parsedScopes = scopes
      .split(",")
      .map((scope) => scope.trim())
      .filter(Boolean) as Scope[];

    setTheme(theme);
    setDefaultScopeId(scopeId);
    setSession({
      subject: subject.trim(),
      token: token.trim(),
      tenantId: scopeId.trim() || undefined,
      scopes: parsedScopes,
      baseUrl: ensureBaseURL(baseUrl),
      theme,
    });

    window.location.href = "/dashboard";
  };

  return (
    <main style={{ minHeight: "100vh", display: "grid", placeItems: "center", padding: 16 }}>
      <div style={{ width: "min(920px, 100%)", display: "grid", gap: 16 }}>
        <PageTitle title="Opener NetDoor" subtitle="Single-owner admin panel login." />

        <div className="grid-two">
          <Card title="Panel access" subtitle="This form stores local browser session only.">
            <form onSubmit={onSubmit} className="row">
              <label>
                API Base URL
                <input value={baseUrl} onChange={(event) => setBaseUrl(event.target.value)} required />
              </label>
              <label>
                Owner name
                <input value={subject} onChange={(event) => setSubject(event.target.value)} required />
              </label>
              <label>
                Theme
                <select value={theme} onChange={(event) => setThemeMode(event.target.value as ThemeMode)}>
                  <option value="dark">dark</option>
                  <option value="light">light</option>
                  <option value="system">system</option>
                </select>
              </label>
              <details style={{ width: "100%" }}>
                <summary>Advanced session options</summary>
                <div style={{ display: "grid", gap: 8, marginTop: 8 }}>
                  <label>
                    Hidden workspace scope
                    <input value={scopeId} onChange={(event) => setScopeId(event.target.value)} />
                  </label>
                  <label>
                    Scopes (comma separated)
                    <input value={scopes} onChange={(event) => setScopes(event.target.value)} required />
                  </label>
                </div>
              </details>
              <label style={{ flexBasis: "100%" }}>
                JWT token
                <textarea value={token} onChange={(event) => setToken(event.target.value)} rows={6} required />
              </label>
              <div style={{ width: "100%", display: "flex", justifyContent: "flex-end" }}>
                <button className="nd-btn is-primary" type="submit">
                  Open panel
                </button>
              </div>
            </form>
          </Card>

          <Card title="Access scopes" subtitle="Navigation and write actions depend on JWT claims.">
            <div className="row">
              {scopes
                .split(",")
                .map((scope) => scope.trim())
                .filter(Boolean)
                .map((scope) => (
                  <StatusBadge key={scope} value={scope} />
                ))}
            </div>
            <p>Internal scope isolation remains enforced in gateway/core-platform even when hidden in UI.</p>
          </Card>
        </div>
      </div>
    </main>
  );
}

