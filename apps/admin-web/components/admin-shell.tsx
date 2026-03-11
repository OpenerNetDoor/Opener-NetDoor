"use client";

import { AppShell } from "./layout";

export function AdminShell({ children }: { children: React.ReactNode }) {
  return <AppShell>{children}</AppShell>;
}