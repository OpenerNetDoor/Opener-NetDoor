import 'dart:convert';

import 'package:flutter/foundation.dart';

import '../models/connection_state.dart';
import '../models/profile_models.dart';
import '../services/secure_storage.dart';
import '../services/vpn_engine.dart';

class AppState extends ChangeNotifier {
  AppState({required this.storage, required this.vpnEngine});

  final SecureStorage storage;
  final VpnEngine vpnEngine;

  String? apiBaseUrl;
  String? token;
  String? selectedNode;
  String selectedProfileId = defaultProfiles.first.id;
  bool autoSelectNode = true;
  final Set<String> favoriteNodes = <String>{};

  ConnectionState connectionState = ConnectionState.idle;
  String diagnostics = 'no diagnostics yet';

  List<NodeLocation> get availableNodes => defaultNodeLocations;
  List<ProtocolProfile> get availableProfiles => defaultProfiles;

  bool get hasProfile => token != null && token!.isNotEmpty;

  String get connectionStateText => connectionStateLabel(connectionState);

  Future<void> loadSession() async {
    apiBaseUrl = await storage.read('api_base_url');
    token = await storage.read('token');
    selectedNode = await storage.read('selected_node');
    selectedProfileId = (await storage.read('selected_profile')) ?? defaultProfiles.first.id;
    autoSelectNode = (await storage.read('auto_select_node')) != 'false';

    final favoritesRaw = await storage.read('favorite_nodes');
    if (favoritesRaw != null && favoritesRaw.isNotEmpty) {
      favoriteNodes
        ..clear()
        ..addAll(favoritesRaw.split(',').where((item) => item.trim().isNotEmpty));
    }

    notifyListeners();
  }

  Future<void> saveSession({
    required String baseUrl,
    required String authToken,
    String? profileId,
  }) async {
    apiBaseUrl = baseUrl.trim();
    token = authToken.trim();
    if (profileId != null && profileId.isNotEmpty) {
      selectedProfileId = profileId;
      await storage.write('selected_profile', selectedProfileId);
    }

    await storage.write('api_base_url', apiBaseUrl!);
    await storage.write('token', token!);
    connectionState = ConnectionState.idle;
    diagnostics = 'profile saved';
    notifyListeners();
  }

  Future<void> importFromManualToken({
    required String baseUrl,
    required String authToken,
    required String profileId,
  }) async {
    await saveSession(baseUrl: baseUrl, authToken: authToken, profileId: profileId);
  }

  Future<void> importFromLink(String link) async {
    final uri = Uri.tryParse(link.trim());
    if (uri == null) {
      diagnostics = 'invalid import link';
      notifyListeners();
      return;
    }

    final nextBase = uri.queryParameters['base_url'];
    final nextToken = uri.queryParameters['token'];
    final profile = uri.queryParameters['profile'];

    if (nextBase == null || nextToken == null) {
      diagnostics = 'import link must contain base_url and token';
      notifyListeners();
      return;
    }

    await saveSession(baseUrl: nextBase, authToken: nextToken, profileId: profile);
    diagnostics = 'profile imported from link';
    notifyListeners();
  }

  Future<void> importFromQrPayload(String payload) async {
    await importFromLink(payload);
  }

  Future<void> connect() async {
    if (!hasProfile) {
      diagnostics = 'profile is not configured';
      connectionState = ConnectionState.authFailed;
      notifyListeners();
      return;
    }

    connectionState = ConnectionState.resolvingProfile;
    notifyListeners();

    final resolvedProfile = availableProfiles.firstWhere(
      (profile) => profile.id == selectedProfileId,
      orElse: () => availableProfiles.first,
    );

    if (resolvedProfile.supportStatus != 'supported') {
      diagnostics = 'selected profile is unsupported';
      connectionState = ConnectionState.blockedOrTimeout;
      notifyListeners();
      return;
    }

    connectionState = ConnectionState.connecting;
    notifyListeners();

    try {
      await vpnEngine.connect(
        token: token!,
        apiBaseUrl: apiBaseUrl ?? 'http://127.0.0.1:8080',
        profileId: resolvedProfile.id,
        nodeId: autoSelectNode ? null : selectedNode,
      );
      connectionState = ConnectionState.connected;
      diagnostics = 'connected via ${resolvedProfile.protocol} (${resolvedProfile.transport})';
    } catch (error) {
      final text = '$error';
      if (text.contains('quota_or_policy_denied')) {
        connectionState = ConnectionState.quotaOrPolicyDenied;
      } else if (text.contains('auth_failed')) {
        connectionState = ConnectionState.authFailed;
      } else if (text.contains('blocked_or_timeout')) {
        connectionState = ConnectionState.blockedOrTimeout;
      } else {
        connectionState = ConnectionState.degraded;
      }
      diagnostics = 'connect failed: $error';
    } finally {
      notifyListeners();
    }
  }

  Future<void> reconnect() async {
    connectionState = ConnectionState.reconnecting;
    notifyListeners();
    await connect();
  }

  Future<void> disconnect() async {
    await vpnEngine.disconnect();
    connectionState = ConnectionState.idle;
    diagnostics = 'disconnected';
    notifyListeners();
  }

  Future<void> setNode(String node) async {
    selectedNode = node.isEmpty ? null : node;
    await storage.write('selected_node', selectedNode ?? '');
    notifyListeners();
  }

  Future<void> toggleFavoriteNode(String nodeId) async {
    if (favoriteNodes.contains(nodeId)) {
      favoriteNodes.remove(nodeId);
    } else {
      favoriteNodes.add(nodeId);
    }
    await storage.write('favorite_nodes', favoriteNodes.join(','));
    notifyListeners();
  }

  Future<void> setAutoSelectNode(bool value) async {
    autoSelectNode = value;
    await storage.write('auto_select_node', value ? 'true' : 'false');
    notifyListeners();
  }

  Future<void> setProfile(String profileId) async {
    selectedProfileId = profileId;
    await storage.write('selected_profile', profileId);
    notifyListeners();
  }

  Future<void> runDiagnostics() async {
    final storageMeta = await storage.metadata();
    final engineDetails = await vpnEngine.diagnostics();
    diagnostics = '$engineDetails\nstorage=${jsonEncode(storageMeta)}';
    notifyListeners();
  }

  Future<String> exportDiagnosticsBundle() async {
    final storageMeta = await storage.metadata();
    final payload = <String, dynamic>{
      'timestamp': DateTime.now().toUtc().toIso8601String(),
      'connection_state': connectionStateText,
      'api_base_url': apiBaseUrl,
      'selected_node': selectedNode,
      'selected_profile': selectedProfileId,
      'auto_select_node': autoSelectNode,
      'favorite_nodes': favoriteNodes.toList(),
      'diagnostics': diagnostics,
      'storage': storageMeta,
    };
    return const JsonEncoder.withIndent('  ').convert(payload);
  }

  Future<void> clearProfile() async {
    await storage.delete('api_base_url');
    await storage.delete('token');
    await storage.delete('selected_node');
    await storage.delete('selected_profile');
    await storage.delete('auto_select_node');
    await storage.delete('favorite_nodes');

    apiBaseUrl = null;
    token = null;
    selectedNode = null;
    selectedProfileId = defaultProfiles.first.id;
    autoSelectNode = true;
    favoriteNodes.clear();
    connectionState = ConnectionState.idle;
    diagnostics = 'profile cleared';
    notifyListeners();
  }
}
