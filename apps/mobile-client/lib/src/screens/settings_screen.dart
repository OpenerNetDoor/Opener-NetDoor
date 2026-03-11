import 'package:flutter/material.dart';

import '../state/app_state.dart';

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key, required this.state});

  final AppState state;

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  String _storageMeta = 'loading';

  @override
  void initState() {
    super.initState();
    _loadStorageMeta();
  }

  Future<void> _loadStorageMeta() async {
    final Map<String, String> meta = await widget.state.storage.metadata();
    setState(() {
      _storageMeta = meta.entries.map((MapEntry<String, String> e) => '${e.key}=${e.value}').join(' | ');
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: AnimatedBuilder(
        animation: widget.state,
        builder: (BuildContext context, _) {
          return Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Text('API base URL: ${widget.state.apiBaseUrl ?? 'not set'}'),
                const SizedBox(height: 8),
                Text('Token present: ${widget.state.token?.isNotEmpty == true ? 'yes' : 'no'}'),
                const SizedBox(height: 8),
                Text('Selected profile: ${widget.state.selectedProfileId}'),
                const SizedBox(height: 8),
                Text('Storage backend: $_storageMeta'),
                const SizedBox(height: 16),
                FilledButton.tonal(
                  onPressed: widget.state.clearProfile,
                  child: const Text('Clear profile'),
                ),
              ],
            ),
          );
        },
      ),
    );
  }
}
