"use client";

import { Card, StatusBadge } from "../ui";
import type { NodeRowVM } from "../../lib/adapters/nodes";

export function NodeSummaryCard({ node }: { node: NodeRowVM }) {
  return (
    <Card title={node.serverName} subtitle={node.region}>
      <div className="row">
        <StatusBadge value={node.status} />
      </div>
      <p>Load telemetry: No data</p>
      <p>Hostname: {node.hostname}</p>
      <p>Capabilities: {node.capabilities.length}</p>
      <p>Last seen: {node.lastSeen}</p>
    </Card>
  );
}
