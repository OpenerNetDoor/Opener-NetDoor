import type { SupportState } from "../permissions";

export interface PlannedAction {
  id: string;
  label: string;
  support: SupportState;
  note: string;
}

export const SUBSCRIPTION_PLANNED_ACTIONS: PlannedAction[] = [
  {
    id: "subscription-create",
    label: "Create subscription plans",
    support: "planned",
    note: "Backend billing endpoints are not part of current Stage 1-12 runtime.",
  },
  {
    id: "subscription-link",
    label: "Issue subscription links",
    support: "frontend_seam",
    note: "UI seam exists; end-user provisioning contract not wired yet.",
  },
  {
    id: "subscription-abuse",
    label: "Abuse throttling",
    support: "planned",
    note: "Will align with policy + quota service in next product slices.",
  },
];