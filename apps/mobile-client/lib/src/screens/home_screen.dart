import 'package:flutter/material.dart';

import '../models/profile_models.dart';
import '../state/app_state.dart';

class HomeScreen extends StatelessWidget {
  const HomeScreen({super.key, required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final ProtocolProfile selectedProfile = state.availableProfiles.firstWhere(
      (ProtocolProfile profile) => profile.id == state.selectedProfileId,
      orElse: () => state.availableProfiles.first,
    );

    return Scaffold(
      appBar: AppBar(title: const Text('Home')),
      body: AnimatedBuilder(
        animation: state,
        builder: (BuildContext context, _) {
          return Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(12),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: <Widget>[
                        Text('State: ${state.connectionStateText}'),
                        const SizedBox(height: 6),
                        Text('Profile: ${selectedProfile.name} (${selectedProfile.transport})'),
                        const SizedBox(height: 6),
                        Text('Node: ${state.autoSelectNode ? 'auto-select' : (state.selectedNode ?? 'not selected')}'),
                        const SizedBox(height: 6),
                        Text('Fallback chain: ${selectedProfile.fallbackOrder.join(' -> ')}'),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  value: state.selectedProfileId,
                  decoration: const InputDecoration(labelText: 'Active profile'),
                  items: state.availableProfiles
                      .map(
                        (ProtocolProfile profile) => DropdownMenuItem<String>(
                          value: profile.id,
                          child: Text('${profile.name} [${profile.supportStatus}]'),
                        ),
                      )
                      .toList(),
                  onChanged: (String? value) {
                    if (value != null) {
                      state.setProfile(value);
                    }
                  },
                ),
                const SizedBox(height: 12),
                Row(
                  children: <Widget>[
                    FilledButton(
                      onPressed: state.connect,
                      child: const Text('Connect'),
                    ),
                    const SizedBox(width: 8),
                    OutlinedButton(
                      onPressed: state.reconnect,
                      child: const Text('Reconnect'),
                    ),
                    const SizedBox(width: 8),
                    OutlinedButton(
                      onPressed: state.disconnect,
                      child: const Text('Disconnect'),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                Text('Diagnostics: ${state.diagnostics}'),
              ],
            ),
          );
        },
      ),
    );
  }
}
