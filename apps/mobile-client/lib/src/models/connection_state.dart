enum ConnectionState {
  idle,
  resolvingProfile,
  connecting,
  connected,
  degraded,
  reconnecting,
  blockedOrTimeout,
  authFailed,
  quotaOrPolicyDenied,
}

String connectionStateLabel(ConnectionState value) {
  switch (value) {
    case ConnectionState.idle:
      return 'idle';
    case ConnectionState.resolvingProfile:
      return 'resolving_profile';
    case ConnectionState.connecting:
      return 'connecting';
    case ConnectionState.connected:
      return 'connected';
    case ConnectionState.degraded:
      return 'degraded';
    case ConnectionState.reconnecting:
      return 'reconnecting';
    case ConnectionState.blockedOrTimeout:
      return 'blocked_or_timeout';
    case ConnectionState.authFailed:
      return 'auth_failed';
    case ConnectionState.quotaOrPolicyDenied:
      return 'quota_or_policy_denied';
  }
}
