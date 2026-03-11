import type {
  ServerFilter,
  ServerInventoryItem,
  ServerOperationJob,
  ServerOperationPlan,
  ServerOperationType,
} from "@opener-netdoor/shared-types";

export class ServerOpsUnsupportedError extends Error {
  constructor(message = "server operations backend endpoints are not available in current control-plane runtime") {
    super(message);
    this.name = "ServerOpsUnsupportedError";
  }
}

export interface CreateServerOperationInput {
  serverId: string;
  operation: ServerOperationType;
}

export interface ServerOperationsAdapter {
  readonly source: "live_api" | "local_planner";
  listServers(filter?: ServerFilter): Promise<ServerInventoryItem[]>;
  createPlan(input: CreateServerOperationInput): Promise<ServerOperationPlan>;
  startOperation(input: CreateServerOperationInput): Promise<ServerOperationJob>;
  listJobs(serverId?: string): Promise<ServerOperationJob[]>;
}

export function createUnsupportedServerOperationsAdapter(): ServerOperationsAdapter {
  return {
    source: "live_api",
    async listServers() {
      throw new ServerOpsUnsupportedError();
    },
    async createPlan() {
      throw new ServerOpsUnsupportedError();
    },
    async startOperation() {
      throw new ServerOpsUnsupportedError();
    },
    async listJobs() {
      throw new ServerOpsUnsupportedError();
    },
  };
}

export function createLocalPlannerServerOperationsAdapter(seedServers: ServerInventoryItem[] = []): ServerOperationsAdapter {
  const now = new Date().toISOString();
  const servers: ServerInventoryItem[] =
    seedServers.length > 0
      ? [...seedServers]
      : [
          {
            id: "srv-eu-01",
            tenant_id: "local",
            region: "eu-central-1",
            ip: "10.10.10.21",
            domain: "edge-eu.example.net",
            status: "active",
            provider: "vds",
            created_at: now,
            updated_at: now,
          },
          {
            id: "srv-us-01",
            tenant_id: "local",
            region: "us-east-1",
            ip: "10.10.11.21",
            domain: "edge-us.example.net",
            status: "needs_attention",
            provider: "vds",
            created_at: now,
            updated_at: now,
          },
        ];

  const jobs: ServerOperationJob[] = [];

  return {
    source: "local_planner",
    async listServers(filter?: ServerFilter) {
      return servers.filter((server) => {
        if (filter?.region && server.region !== filter.region) {
          return false;
        }
        if (filter?.status && server.status !== filter.status) {
          return false;
        }
        if (filter?.query) {
          const q = filter.query.toLowerCase();
          return (
            server.id.toLowerCase().includes(q) ||
            server.region.toLowerCase().includes(q) ||
            server.ip.toLowerCase().includes(q) ||
            server.domain?.toLowerCase().includes(q)
          );
        }
        return true;
      });
    },

    async createPlan(input: CreateServerOperationInput) {
      return {
        server_id: input.serverId,
        operation: input.operation,
        commands: buildCommandPlan(input.operation, input.serverId),
        warnings: buildWarnings(input.operation),
        requires_maintenance_window: input.operation === "restore" || input.operation === "rollback",
        generated_at: new Date().toISOString(),
        source: "local_planner",
      };
    },

    async startOperation(input: CreateServerOperationInput) {
      const id = `job-${Date.now()}-${Math.floor(Math.random() * 1000)}`;
      const correlationId = `ond-${Date.now()}-${input.serverId}`;
      const startedAt = new Date().toISOString();
      const job: ServerOperationJob = {
        id,
        server_id: input.serverId,
        operation: input.operation,
        status: "running",
        started_at: startedAt,
        correlation_id: correlationId,
        logs: [
          {
            level: "info",
            ts: startedAt,
            message: `local planner started ${input.operation} for ${input.serverId}`,
          },
          {
            level: "warn",
            ts: startedAt,
            message: "backend runtime endpoint is unavailable; this is a local planning run",
          },
        ],
        source: "local_planner",
      };
      jobs.unshift(job);
      return job;
    },

    async listJobs(serverId?: string) {
      if (!serverId) {
        return jobs;
      }
      return jobs.filter((job) => job.server_id === serverId);
    },
  };
}

function buildCommandPlan(operation: ServerOperationType, serverId: string): string[] {
  switch (operation) {
    case "install":
      return [
        `opener-netdoor installer install --server ${serverId}`,
        "opener-netdoor installer doctor --strict",
      ];
    case "upgrade":
      return [
        `opener-netdoor installer backup --server ${serverId}`,
        `opener-netdoor installer upgrade --server ${serverId}`,
      ];
    case "rollback":
      return [`opener-netdoor installer rollback --server ${serverId}`];
    case "backup":
      return [`opener-netdoor installer backup --server ${serverId}`];
    case "restore":
      return [`opener-netdoor installer restore --server ${serverId}`];
    case "cert_rotate":
      return [`opener-netdoor installer rotate-keys --server ${serverId}`];
    case "firewall_apply":
      return [`opener-netdoor installer apply-firewall --server ${serverId}`];
    case "cdn_apply":
      return [`opener-netdoor installer apply-edge --server ${serverId}`];
    default:
      return [`echo unsupported operation ${operation}`];
  }
}

function buildWarnings(operation: ServerOperationType): string[] {
  if (operation === "restore") {
    return ["restore requires maintenance window and traffic reroute"]; 
  }
  if (operation === "rollback") {
    return ["rollback can revoke recently issued node certificates"]; 
  }
  if (operation === "cdn_apply") {
    return ["CDN mode should be used only with compatible WS/gRPC TLS profiles"]; 
  }
  return [];
}
