"use client";

import type { ReactNode } from "react";
import { Search, X } from "lucide-react";
import type { NotificationView } from "../../lib/adapters/notifications";
import { cn } from "../../lib/format";

export function SectionHeader({
  title,
  subtitle,
  actions,
}: {
  title: string;
  subtitle?: string;
  actions?: ReactNode;
}) {
  return (
    <header className="nd-section-header">
      <div>
        <h1>{title}</h1>
        {subtitle ? <p>{subtitle}</p> : null}
      </div>
      {actions ? <div className="nd-section-actions">{actions}</div> : null}
    </header>
  );
}

export function PageTitle(props: { title: string; subtitle?: string; actions?: ReactNode }) {
  return <SectionHeader {...props} />;
}

export function Card({
  title,
  subtitle,
  actions,
  children,
  className,
}: {
  title?: string;
  subtitle?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={cn("nd-card", className)}>
      {title || actions ? (
        <div className="nd-card-head">
          <div>
            {title ? <h2>{title}</h2> : null}
            {subtitle ? <p>{subtitle}</p> : null}
          </div>
          {actions ? <div className="nd-card-actions">{actions}</div> : null}
        </div>
      ) : null}
      {children}
    </section>
  );
}

export function StatCard({
  label,
  value,
  delta,
  helper,
  icon,
  tone = "neutral",
}: {
  label: string;
  value: string;
  delta?: string;
  helper?: string;
  icon?: ReactNode;
  tone?: "neutral" | "success" | "warning" | "danger";
}) {
  return (
    <article className="nd-stat-card">
      <div className="nd-stat-top">
        <div className="nd-stat-icon" aria-hidden>
          {icon ?? "•"}
        </div>
      </div>
      <div className="nd-stat-copy">
        <p>{label}</p>
        <strong>{value}</strong>
        {delta ? <div className={cn("nd-delta", `is-${tone === "neutral" ? "success" : tone}`)}>{delta}</div> : null}
        {helper ? <span>{helper}</span> : null}
      </div>
    </article>
  );
}

export function ActionButton({
  children,
  onClick,
  variant = "primary",
  disabled,
  type = "button",
}: {
  children: ReactNode;
  onClick?: () => void;
  variant?: "primary" | "secondary" | "danger" | "ghost";
  disabled?: boolean;
  type?: "button" | "submit";
}) {
  return (
    <button className={cn("nd-btn", `is-${variant}`)} type={type} onClick={onClick} disabled={disabled}>
      {children}
    </button>
  );
}

export function IconButton({
  icon,
  onClick,
  label,
  variant = "ghost",
  disabled,
}: {
  icon: ReactNode;
  onClick?: () => void;
  label: string;
  variant?: "ghost" | "secondary" | "danger";
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      className={cn("nd-icon-btn", variant === "secondary" && "is-secondary", variant === "danger" && "is-danger")}
      onClick={onClick}
      aria-label={label}
      disabled={disabled}
    >
      {icon}
    </button>
  );
}

export function StatusBadge({ value }: { value: string }) {
  const normalized = value.toLowerCase();
  const tone =
    normalized.includes("active") ||
    normalized.includes("online") ||
    normalized.includes("connected") ||
    normalized.includes("healthy")
      ? "success"
      : normalized.includes("pending") || normalized.includes("warning") || normalized.includes("planned")
        ? "warning"
        : normalized.includes("blocked") ||
            normalized.includes("revoked") ||
            normalized.includes("error") ||
            normalized.includes("failed") ||
            normalized.includes("deny")
          ? "danger"
          : "neutral";
  return <span className={cn("nd-badge", `is-${tone}`)}>{value}</span>;
}

export function SupportBadge({ state }: { state: "supported" | "frontend_seam" | "planned" | "unsupported" }) {
  return <StatusBadge value={state.replaceAll("_", " ")} />;
}

export function HealthChip({ label, status }: { label: string; status?: string }) {
  return (
    <div className="nd-health-chip">
      <span>{label}</span>
      <StatusBadge value={status ?? "unknown"} />
    </div>
  );
}

export function ProtocolChip({ label, unsupported }: { label: string; unsupported?: boolean }) {
  return <span className={cn("nd-protocol-chip", unsupported && "unsupported")}>{label}</span>;
}

export function ProgressBar({
  value,
  label,
  hint,
}: {
  value: number;
  label?: string;
  hint?: string;
}) {
  const safe = Math.max(0, Math.min(100, value));
  return (
    <div className="nd-progress-wrap">
      {label ? (
        <div className="nd-progress-copy">
          <span>{label}</span>
          <span>{hint ?? `${safe.toFixed(0)}%`}</span>
        </div>
      ) : null}
      <div className="nd-progress-track">
        <div className="nd-progress-fill" style={{ width: `${safe}%` }} />
      </div>
    </div>
  );
}

export function SearchInput({
  value,
  onChange,
  placeholder = "Search...",
}: {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}) {
  return (
    <label className="nd-search-wrap" aria-label="Search">
      <Search size={16} />
      <input value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} />
    </label>
  );
}

export function LoadingState({ label = "Loading..." }: { label?: string }) {
  return (
    <div className="nd-state is-loading">
      <span className="nd-spinner" aria-hidden />
      <p>{label}</p>
    </div>
  );
}

export function ErrorState({ message }: { message: string }) {
  return (
    <div className="nd-state is-error">
      <strong>Request failed</strong>
      <p>{message}</p>
    </div>
  );
}

export function EmptyState({ title, description }: { title: string; description: string }) {
  return (
    <div className="nd-state is-empty">
      <strong>{title}</strong>
      <p>{description}</p>
    </div>
  );
}

export function PermissionDeniedState() {
  return <ErrorState message="Your token does not have required scopes for this route." />;
}

export function ScopeMismatchState({ expected, actual }: { expected?: string; actual?: string }) {
  return (
    <div className="nd-state is-warning">
      <strong>Scope mismatch</strong>
      <p>
        actor scope: <code>{actual || "none"}</code>; required scope: <code>{expected || "n/a"}</code>
      </p>
    </div>
  );
}

export interface DataColumn<T> {
  id: string;
  header: string;
  render: (row: T) => ReactNode;
  className?: string;
}

export function DataTable<T>({
  rows,
  columns,
  rowKey,
  emptyTitle = "No data",
  emptyDescription = "Adjust filters or create a new entity.",
}: {
  rows: T[];
  columns: DataColumn<T>[];
  rowKey: (row: T, index: number) => string;
  emptyTitle?: string;
  emptyDescription?: string;
}) {
  if (rows.length === 0) {
    return <EmptyState title={emptyTitle} description={emptyDescription} />;
  }

  return (
    <div className="nd-table-wrap">
      <table className="nd-table">
        <thead>
          <tr>
            {columns.map((column) => (
              <th key={column.id}>{column.header}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, index) => (
            <tr key={rowKey(row, index)}>
              {columns.map((column) => (
                <td key={column.id} className={column.className}>
                  {column.render(row)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function PaginationControls({
  limit,
  offset,
  onChange,
}: {
  limit: number;
  offset: number;
  onChange: (next: { limit: number; offset: number }) => void;
}) {
  return (
    <div className="nd-pagination">
      <ActionButton variant="secondary" disabled={offset === 0} onClick={() => onChange({ limit, offset: Math.max(0, offset - limit) })}>
        Previous
      </ActionButton>
      <span>
        offset <code>{offset}</code> / limit <code>{limit}</code>
      </span>
      <ActionButton variant="secondary" onClick={() => onChange({ limit, offset: offset + limit })}>
        Next
      </ActionButton>
    </div>
  );
}

export function LogViewer({ logs }: { logs: Array<unknown> }) {
  if (logs.length === 0) {
    return <EmptyState title="No logs" description="No entries for selected filters." />;
  }
  return (
    <div className="nd-log-viewer">
      {logs.map((entry, index) => {
        const item = asRecord(entry);
        return (
          <details key={`${item.action ?? item.ts ?? index}`}>
            <summary>
              {String(item.action ?? item.level ?? "event")} at {String(item.created_at ?? item.ts ?? "n/a")}
            </summary>
            <pre>{JSON.stringify(item, null, 2)}</pre>
          </details>
        );
      })}
    </div>
  );
}

export function CommandPreview({
  title,
  commands,
  warnings,
}: {
  title: string;
  commands: string[];
  warnings?: string[];
}) {
  return (
    <section className="nd-command-preview">
      <h3>{title}</h3>
      {warnings?.length ? (
        <ul>
          {warnings.map((warning) => (
            <li key={warning}>{warning}</li>
          ))}
        </ul>
      ) : null}
      <pre>{commands.join("\n")}</pre>
    </section>
  );
}

export function StepperWizard({
  steps,
  current,
}: {
  steps: Array<{ id: string; label: string }>;
  current: string;
}) {
  return (
    <ol className="nd-stepper">
      {steps.map((step) => (
        <li key={step.id} className={cn("nd-step", step.id === current && "is-current")}>
          {step.label}
        </li>
      ))}
    </ol>
  );
}

export function ConfirmDangerButton({
  label,
  prompt,
  onConfirm,
  disabled,
}: {
  label: string;
  prompt: string;
  onConfirm: () => void | Promise<void>;
  disabled?: boolean;
}) {
  return (
    <ActionButton
      variant="danger"
      disabled={disabled}
      onClick={() => {
        if (window.confirm(prompt)) {
          void onConfirm();
        }
      }}
    >
      {label}
    </ActionButton>
  );
}

export function ModalShell({
  title,
  open,
  onClose,
  children,
}: {
  title: string;
  open: boolean;
  onClose: () => void;
  children: ReactNode;
}) {
  if (!open) {
    return null;
  }
  return (
    <div className="nd-modal-overlay" role="dialog" aria-modal="true">
      <div className="nd-modal-card">
        <header>
          <h3>{title}</h3>
          <button type="button" className="nd-icon-btn" onClick={onClose} aria-label="Close">
            <X size={16} />
          </button>
        </header>
        <div>{children}</div>
      </div>
    </div>
  );
}

export function DrawerShell({
  title,
  open,
  onClose,
  children,
}: {
  title: string;
  open: boolean;
  onClose: () => void;
  children: ReactNode;
}) {
  if (!open) {
    return null;
  }
  return (
    <div className="nd-modal-overlay" role="dialog" aria-modal="true">
      <aside className="nd-drawer-card">
        <header>
          <h3>{title}</h3>
          <button type="button" className="nd-icon-btn" onClick={onClose} aria-label="Close">
            <X size={16} />
          </button>
        </header>
        <div>{children}</div>
      </aside>
    </div>
  );
}

export function NotificationItem({ item }: { item: NotificationView }) {
  return (
    <article className="nd-notification-item">
      <div className={cn("dot", `tone-${item.tone}`)} aria-hidden />
      <div>
        <p>{item.title}</p>
        <small>{item.subtitle}</small>
      </div>
      <time>{item.when}</time>
    </article>
  );
}

function asRecord(value: unknown): Record<string, unknown> {
  if (typeof value === "object" && value !== null) {
    return value as Record<string, unknown>;
  }
  return { value };
}

