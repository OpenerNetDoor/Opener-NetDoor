"use client";

import { useState } from "react";
import {
  Bell,
  Database,
  Globe,
  Palette,
  RefreshCw,
  Shield,
  SlidersHorizontal,
} from "lucide-react";
import { AdminShell } from "../../components/admin-shell";
import { RouteGuard } from "../../components/route-guard";
import { ActionButton, Card, PageTitle, StatusBadge, SupportBadge } from "../../components/ui";

type SettingsTab = "general" | "security" | "notifications" | "system";

interface ToggleState {
  registration: boolean;
  maintenance: boolean;
  serverAlerts: boolean;
  userAlerts: boolean;
  paymentAlerts: boolean;
  usageAlerts: boolean;
}

export default function SettingsPage() {
  const [tab, setTab] = useState<SettingsTab>("general");
  const [theme, setTheme] = useState("Dark");
  const [defaultProtocol, setDefaultProtocol] = useState("VLESS");
  const [logRetention, setLogRetention] = useState(30);
  const [toggles, setToggles] = useState<ToggleState>({
    registration: false,
    maintenance: false,
    serverAlerts: false,
    userAlerts: false,
    paymentAlerts: false,
    usageAlerts: false,
  });

  return (
    <RouteGuard requiredScopes={["admin:read"]}>
      <AdminShell>
        <PageTitle title="Settings" subtitle="Configure system preferences" actions={<SupportBadge state="frontend_seam" />} />

        <div className="nd-segmented" role="tablist" aria-label="Settings tabs">
          <button type="button" className={tab === "general" ? "is-active" : ""} onClick={() => setTab("general")}>
            <Globe size={14} /> General
          </button>
          <button type="button" className={tab === "security" ? "is-active" : ""} onClick={() => setTab("security")}>
            <Shield size={14} /> Security
          </button>
          <button type="button" className={tab === "notifications" ? "is-active" : ""} onClick={() => setTab("notifications")}>
            <Bell size={14} /> Notifications
          </button>
          <button type="button" className={tab === "system" ? "is-active" : ""} onClick={() => setTab("system")}>
            <SlidersHorizontal size={14} /> System
          </button>
        </div>

        {tab === "general" ? (
          <>
            <Card title="Site Information" subtitle="Basic site configuration">
              <div className="grid-two">
                <label>
                  Site Name
                  <input value="Opener NetDoor" readOnly />
                </label>
                <label>
                  Site Description
                  <input value="Secure VPN Service" readOnly />
                </label>
              </div>
            </Card>

            <Card title="Appearance" subtitle="Customize the look and feel">
              <div className="row" style={{ alignItems: "center" }}>
                <Palette size={16} />
                <span>Default Theme</span>
                <select value={theme} onChange={(event) => setTheme(event.target.value)} style={{ maxWidth: 180 }}>
                  <option value="Dark">Dark</option>
                  <option value="Light">Light</option>
                  <option value="System">System</option>
                </select>
              </div>
            </Card>

            <Card title="Default Protocol" subtitle="Default VPN protocol for new users">
              <select value={defaultProtocol} onChange={(event) => setDefaultProtocol(event.target.value)} style={{ maxWidth: 240 }}>
                <option value="VLESS">VLESS</option>
                <option value="VMess">VMess</option>
                <option value="Trojan">Trojan</option>
                <option value="Shadowsocks">Shadowsocks</option>
              </select>
            </Card>
          </>
        ) : null}

        {tab === "security" ? (
          <Card title="Access Control" subtitle="Manage access and authentication">
            <div className="nd-setting-row">
              <div>
                <strong>Allow Registration</strong>
                <p style={{ margin: 0, color: "var(--nd-text-muted)" }}>Let new users create accounts</p>
              </div>
              <button
                className={`nd-switch ${toggles.registration ? "is-on" : ""}`}
                type="button"
                onClick={() => setToggles((prev) => ({ ...prev, registration: !prev.registration }))}
                aria-label="Toggle registration"
              />
            </div>
            <div className="nd-setting-row">
              <div>
                <strong>Maintenance Mode</strong>
                <p style={{ margin: 0, color: "var(--nd-text-muted)" }}>Disable access for maintenance window</p>
              </div>
              <button
                className={`nd-switch ${toggles.maintenance ? "is-on" : ""}`}
                type="button"
                onClick={() => setToggles((prev) => ({ ...prev, maintenance: !prev.maintenance }))}
                aria-label="Toggle maintenance"
              />
            </div>
            <label style={{ marginTop: 12, display: "block" }}>
              Session Timeout (minutes)
              <input value="60" readOnly />
            </label>
          </Card>
        ) : null}

        {tab === "notifications" ? (
          <Card title="Notification Preferences" subtitle="Configure alert settings">
            {[
              ["Server Alerts", "Get notified when servers go offline", "serverAlerts"],
              ["New User Registrations", "Receive alerts for new signups", "userAlerts"],
              ["Payment Notifications", "Get notified of new payments", "paymentAlerts"],
              ["High Usage Alerts", "Alert when server load is high", "usageAlerts"],
            ].map(([title, copy, key]) => {
              const typedKey = key as keyof ToggleState;
              return (
                <div className="nd-setting-row" key={key}>
                  <div>
                    <strong>{title}</strong>
                    <p style={{ margin: 0, color: "var(--nd-text-muted)" }}>{copy}</p>
                  </div>
                  <button
                    className={`nd-switch ${toggles[typedKey] ? "is-on" : ""}`}
                    type="button"
                    onClick={() => setToggles((prev) => ({ ...prev, [typedKey]: !prev[typedKey] }))}
                    aria-label={`Toggle ${title}`}
                  />
                </div>
              );
            })}
          </Card>
        ) : null}

        {tab === "system" ? (
          <>
            <Card title="Data Management" subtitle="Configure data retention policies">
              <div className="row" style={{ alignItems: "center" }}>
                <Database size={16} />
                <label style={{ maxWidth: 260 }}>
                  Log Retention (days)
                  <input
                    type="number"
                    value={logRetention}
                    onChange={(event) => setLogRetention(Number(event.target.value || 30))}
                  />
                </label>
              </div>
            </Card>

            <Card title="Danger Zone" subtitle="Irreversible actions">
              <div className="nd-setting-row">
                <div>
                  <strong>Clear All Logs</strong>
                  <p style={{ margin: 0, color: "var(--nd-text-muted)" }}>Delete all connection history</p>
                </div>
                <ActionButton variant="danger">Clear Logs</ActionButton>
              </div>
              <div className="nd-setting-row">
                <div>
                  <strong>Reset to Defaults</strong>
                  <p style={{ margin: 0, color: "var(--nd-text-muted)" }}>Restore all settings to factory defaults</p>
                </div>
                <ActionButton variant="secondary">
                  <RefreshCw size={15} /> Reset
                </ActionButton>
              </div>
              <div style={{ marginTop: 10 }}>
                <StatusBadge value="frontend_seam" />
              </div>
            </Card>
          </>
        ) : null}
      </AdminShell>
    </RouteGuard>
  );
}
