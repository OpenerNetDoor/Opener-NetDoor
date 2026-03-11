import 'package:flutter/material.dart';

import '../models/profile_models.dart';
import '../state/app_state.dart';

class LoginImportScreen extends StatefulWidget {
  const LoginImportScreen({super.key, required this.state});

  final AppState state;

  @override
  State<LoginImportScreen> createState() => _LoginImportScreenState();
}

class _LoginImportScreenState extends State<LoginImportScreen> {
  final TextEditingController _apiBaseController = TextEditingController(text: 'http://127.0.0.1:8080');
  final TextEditingController _tokenController = TextEditingController();
  final TextEditingController _linkController = TextEditingController();
  final TextEditingController _qrController = TextEditingController();

  String _selectedProfile = defaultProfiles.first.id;
  int _mode = 0;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Import Profile')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            const Text('Login or import subscription profile (manual/link/QR payload).'),
            const SizedBox(height: 12),
            SegmentedButton<int>(
              segments: const <ButtonSegment<int>>[
                ButtonSegment<int>(value: 0, label: Text('Manual')),
                ButtonSegment<int>(value: 1, label: Text('Link')),
                ButtonSegment<int>(value: 2, label: Text('QR Payload')),
              ],
              selected: <int>{_mode},
              onSelectionChanged: (Set<int> value) => setState(() => _mode = value.first),
            ),
            const SizedBox(height: 12),
            Expanded(
              child: SingleChildScrollView(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    if (_mode == 0) ...<Widget>[
                      TextField(
                        controller: _apiBaseController,
                        decoration: const InputDecoration(labelText: 'API base URL'),
                      ),
                      const SizedBox(height: 8),
                      DropdownButtonFormField<String>(
                        value: _selectedProfile,
                        decoration: const InputDecoration(labelText: 'Protocol profile'),
                        items: defaultProfiles
                            .map(
                              (ProtocolProfile profile) => DropdownMenuItem<String>(
                                value: profile.id,
                                child: Text('${profile.name} (${profile.supportStatus})'),
                              ),
                            )
                            .toList(),
                        onChanged: (String? value) => setState(() => _selectedProfile = value ?? defaultProfiles.first.id),
                      ),
                      const SizedBox(height: 8),
                      TextField(
                        controller: _tokenController,
                        minLines: 3,
                        maxLines: 5,
                        decoration: const InputDecoration(labelText: 'JWT token or subscription secret'),
                      ),
                      const SizedBox(height: 12),
                      FilledButton(
                        onPressed: () async {
                          await widget.state.importFromManualToken(
                            baseUrl: _apiBaseController.text,
                            authToken: _tokenController.text,
                            profileId: _selectedProfile,
                          );
                        },
                        child: const Text('Save profile'),
                      ),
                    ],
                    if (_mode == 1) ...<Widget>[
                      TextField(
                        controller: _linkController,
                        minLines: 3,
                        maxLines: 5,
                        decoration: const InputDecoration(
                          labelText: 'Import link',
                          hintText: 'openernetdoor://import?base_url=http://...&token=...&profile=profile-default-vless',
                        ),
                      ),
                      const SizedBox(height: 12),
                      FilledButton(
                        onPressed: () async {
                          await widget.state.importFromLink(_linkController.text);
                        },
                        child: const Text('Import from link'),
                      ),
                    ],
                    if (_mode == 2) ...<Widget>[
                      TextField(
                        controller: _qrController,
                        minLines: 3,
                        maxLines: 5,
                        decoration: const InputDecoration(
                          labelText: 'QR payload',
                          hintText: 'Paste decoded payload from QR scanner seam',
                        ),
                      ),
                      const SizedBox(height: 12),
                      FilledButton(
                        onPressed: () async {
                          await widget.state.importFromQrPayload(_qrController.text);
                        },
                        child: const Text('Import QR payload'),
                      ),
                    ],
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
