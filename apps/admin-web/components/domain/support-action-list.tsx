"use client";

import Link from "next/link";
import { ArrowRight, Clock3 } from "lucide-react";
import { StatusBadge } from "../ui";

export interface SupportActionItem {
  id: string;
  label: string;
  href: string;
  support: "supported" | "frontend_seam" | "planned" | "unsupported";
}

export function SupportActionList({ items }: { items: SupportActionItem[] }) {
  return (
    <div className="nd-quick-actions">
      {items.map((item) => (
        <Link key={item.id} href={item.href}>
          <button className="nd-quick-action" type="button">
            <span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}>
              {item.support === "supported" ? <ArrowRight size={16} /> : <Clock3 size={16} />}
              <span>{item.label}</span>
            </span>
            <StatusBadge value={item.support.replaceAll("_", " ")} />
          </button>
        </Link>
      ))}
    </div>
  );
}
