class ProtocolProfile {
  const ProtocolProfile({
    required this.id,
    required this.name,
    required this.protocol,
    required this.transport,
    required this.cdnFriendly,
    required this.fallbackOrder,
    required this.supportStatus,
  });

  final String id;
  final String name;
  final String protocol;
  final String transport;
  final bool cdnFriendly;
  final List<String> fallbackOrder;
  final String supportStatus;
}

class NodeLocation {
  const NodeLocation({
    required this.id,
    required this.label,
    required this.region,
    required this.latencyMs,
  });

  final String id;
  final String label;
  final String region;
  final int latencyMs;
}

const List<ProtocolProfile> defaultProfiles = <ProtocolProfile>[
  ProtocolProfile(
    id: 'profile-default-vless',
    name: 'Default VLESS WS TLS',
    protocol: 'vless',
    transport: 'ws+tls',
    cdnFriendly: true,
    fallbackOrder: <String>['profile-fallback-trojan', 'profile-wireguard'],
    supportStatus: 'supported',
  ),
  ProtocolProfile(
    id: 'profile-fallback-trojan',
    name: 'Fallback Trojan TLS',
    protocol: 'trojan',
    transport: 'tls',
    cdnFriendly: true,
    fallbackOrder: <String>['profile-wireguard'],
    supportStatus: 'supported',
  ),
  ProtocolProfile(
    id: 'profile-wireguard',
    name: 'WireGuard Primary',
    protocol: 'wireguard',
    transport: 'udp-native',
    cdnFriendly: false,
    fallbackOrder: <String>[],
    supportStatus: 'supported',
  ),
  ProtocolProfile(
    id: 'profile-mieru-advanced',
    name: 'mieru Advanced',
    protocol: 'mieru',
    transport: 'quic',
    cdnFriendly: false,
    fallbackOrder: <String>['profile-default-vless'],
    supportStatus: 'supported',
  ),
  ProtocolProfile(
    id: 'profile-nieva-placeholder',
    name: 'Nieva (unverified)',
    protocol: 'nieva',
    transport: 'n/a',
    cdnFriendly: false,
    fallbackOrder: <String>[],
    supportStatus: 'unsupported_unverified',
  ),
];

const List<NodeLocation> defaultNodeLocations = <NodeLocation>[
  NodeLocation(id: 'auto', label: 'Auto-select', region: 'auto', latencyMs: 0),
  NodeLocation(id: 'eu-frankfurt-1', label: 'Frankfurt', region: 'eu-central-1', latencyMs: 42),
  NodeLocation(id: 'us-virginia-1', label: 'Virginia', region: 'us-east-1', latencyMs: 95),
  NodeLocation(id: 'sg-singapore-1', label: 'Singapore', region: 'ap-southeast-1', latencyMs: 132),
];
