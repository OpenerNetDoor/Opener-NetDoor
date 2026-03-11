import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../state/app_state.dart';

class DiagnosticsScreen extends StatelessWidget {
  const DiagnosticsScreen({super.key, required this.state});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Diagnostics')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: AnimatedBuilder(
          animation: state,
          builder: (BuildContext context, _) {
            return Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Row(
                  children: <Widget>[
                    FilledButton(
                      onPressed: state.runDiagnostics,
                      child: const Text('Run diagnostics'),
                    ),
                    const SizedBox(width: 8),
                    OutlinedButton(
                      onPressed: () async {
                        final String payload = await state.exportDiagnosticsBundle();
                        await Clipboard.setData(ClipboardData(text: payload));
                        if (context.mounted) {
                          ScaffoldMessenger.of(context).showSnackBar(
                            const SnackBar(content: Text('Diagnostics bundle copied to clipboard')),
                          );
                        }
                      },
                      child: const Text('Export bundle'),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                Expanded(
                  child: SingleChildScrollView(
                    child: SelectableText(state.diagnostics),
                  ),
                ),
              ],
            );
          },
        ),
      ),
    );
  }
}
