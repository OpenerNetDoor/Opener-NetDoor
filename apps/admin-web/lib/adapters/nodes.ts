import type { Node } from "@opener-netdoor/shared-types";
import { formatRelativeTime } from "../format";

export interface NodeRowVM {
  id: string;
  tenantId: string;
  serverName: string;
  hostname: string;
  region: string;
  countryCode: string;
  status: string;
  capabilities: string[];
  lastSeen: string;
  heartbeat: string;
  advanced: {
    nodeKeyId: string;
    contractVersion: string;
    agentVersion: string;
    identityFingerprint: string;
  };
}

const REGION_CODE: Record<string, string> = {
  us: "US",
  de: "DE",
  nl: "NL",
  sg: "SG",
  jp: "JP",
  gb: "GB",
  fr: "FR",
  ca: "CA",
};

function deriveCountryCode(region: string): string {
  const key = region.split(/[-_]/)[0]?.toLowerCase() ?? "";
  return REGION_CODE[key] ?? (key.slice(0, 2).toUpperCase() || "--");
}

function normalizeServerName(hostname: string): string {
  const base = hostname.split(".")[0] ?? hostname;
  return base.trim() || hostname;
}

export function toNodeRowVM(node: Node): NodeRowVM {
  return {
    id: node.id,
    tenantId: node.tenant_id,
    serverName: normalizeServerName(node.hostname),
    hostname: node.hostname,
    region: node.region,
    countryCode: deriveCountryCode(node.region),
    status: node.status,
    capabilities: node.capabilities || [],
    lastSeen: formatRelativeTime(node.last_seen_at),
    heartbeat: formatRelativeTime(node.last_heartbeat_at),
    advanced: {
      nodeKeyId: node.node_key_id,
      contractVersion: node.contract_version || "n/a",
      agentVersion: node.agent_version || "n/a",
      identityFingerprint: node.identity_fingerprint || "",
    },
  };
}

export function toNodeRows(nodes: Node[]): NodeRowVM[] {
  return nodes.map(toNodeRowVM);
}
