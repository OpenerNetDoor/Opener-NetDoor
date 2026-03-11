# Admin-Web Reuse Baseline

This document captures reusable presentation patterns from `apps/admin-web` for the next `apps/manager-desktop` UX pass.

## Reuse Targets (Single-Owner Mode)

- Design tokens
  - Dark-first color tokens with practical status semantics (`success/warning/danger/info`).
  - Compact spacing rhythm for dense operator workflows.
  - Card, border, and hover hierarchy tuned for long-running dashboards.
- Shell concepts
  - Compact sidebar with owner-focused IA: Dashboard, Users, Servers, Keys, Subscriptions, Analytics, Settings.
  - Topbar with search/date/theme/notifications/profile controls.
  - Technical routes moved to an explicit advanced area.
- Components
  - Stat cards
  - Status chips and support badges (`supported/frontend_seam/planned/unsupported`)
  - Table shell + pagination
  - Empty/error/loading states
  - Drawer/modal for contextual diagnostics
- Operational patterns
  - Actions that are not backed by runtime endpoints remain visible but clearly labeled as seams.
  - Server diagnostics split into simple main cards and advanced detail drawer.

## Constraints

- Keep Tauri-safe implementation.
- Avoid runtime coupling to Next.js-specific code.
- Preserve existing manager-desktop route model and IPC seams.
- Keep backend contract honesty: no fake live orchestration where endpoint is absent.
