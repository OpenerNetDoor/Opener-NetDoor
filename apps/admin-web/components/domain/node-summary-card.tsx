"use client";

import { Card, ProgressBar, StatusBadge } from "../ui";
import type { NodeRowVM } from "../../lib/adapters/nodes";

export function NodeSummaryCard({ node }: { node: NodeRowVM }) {
  return (
    <Card title={node.serverName} subtitle={`${node.region} · ${node.healthLabel}`}>
      <div className="row">
        <StatusBadge value={node.status} />
      </div>
      <ProgressBar value={node.healthScore} label="Health" hint={`${node.healthScore}%`} />
      <p>Hostname: {node.hostname}</p>
      <p>Capabilities: {node.capabilities.length}</p>
      <p>Last seen: {node.lastSeen}</p>
    </Card>
  );
}
