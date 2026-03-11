abstract class SecureStorage {
  Future<void> write(String key, String value);
  Future<String?> read(String key);
  Future<void> delete(String key);
  Future<Map<String, String>> metadata();
}

class MemorySecureStorage implements SecureStorage {
  final Map<String, String> _cache = <String, String>{};

  @override
  Future<void> delete(String key) async {
    _cache.remove(key);
  }

  @override
  Future<Map<String, String>> metadata() async {
    return <String, String>{
      'backend': 'memory',
      'encryption': 'none',
      'note': 'replace with flutter_secure_storage + hardware-backed keystore in production build',
    };
  }

  @override
  Future<String?> read(String key) async {
    return _cache[key];
  }

  @override
  Future<void> write(String key, String value) async {
    _cache[key] = value;
  }
}
