abstract class VpnEngine {
  Future<void> connect({
    required String token,
    required String apiBaseUrl,
    required String profileId,
    String? nodeId,
  });

  Future<void> disconnect();
  Future<String> diagnostics();
}

class StubVpnEngine implements VpnEngine {
  bool _connected = false;
  String _lastNode = 'auto';
  String _lastProfile = 'profile-default-vless';

  @override
  Future<void> connect({
    required String token,
    required String apiBaseUrl,
    required String profileId,
    String? nodeId,
  }) async {
    if (token.trim().isEmpty) {
      throw Exception('auth_failed: empty token');
    }
    if (token.contains('auth-fail')) {
      throw Exception('auth_failed: invalid credentials');
    }
    if (token.contains('quota-denied')) {
      throw Exception('quota_or_policy_denied: user exceeded quota');
    }
    if (token.contains('timeout')) {
      throw Exception('blocked_or_timeout: transport timeout');
    }

    _connected = true;
    _lastNode = nodeId?.isNotEmpty == true ? nodeId! : 'auto';
    _lastProfile = profileId;
  }

  @override
  Future<void> disconnect() async {
    _connected = false;
  }

  @override
  Future<String> diagnostics() async {
    return 'engine=stub connected=$_connected profile=$_lastProfile node=$_lastNode';
  }
}
