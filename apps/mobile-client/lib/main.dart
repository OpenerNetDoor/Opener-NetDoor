import 'package:flutter/material.dart';

void main() {
  runApp(const OpenerNetDoorApp());
}

class OpenerNetDoorApp extends StatelessWidget {
  const OpenerNetDoorApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Opener NetDoor Mobile',
      home: const HomePage(),
      theme: ThemeData(colorSchemeSeed: Colors.blue, useMaterial3: true),
    );
  }
}

class HomePage extends StatelessWidget {
  const HomePage({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Opener NetDoor Mobile')),
      body: const Center(
        child: Text('TODO: connect flow, QR import, diagnostics, failover.'),
      ),
    );
  }
}

