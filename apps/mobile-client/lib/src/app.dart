import 'package:flutter/material.dart';

import 'screens/diagnostics_screen.dart';
import 'screens/home_screen.dart';
import 'screens/login_import_screen.dart';
import 'screens/nodes_screen.dart';
import 'screens/settings_screen.dart';
import 'services/secure_storage.dart';
import 'services/vpn_engine.dart';
import 'state/app_state.dart';

class OpenerNetDoorMobileApp extends StatefulWidget {
  const OpenerNetDoorMobileApp({super.key});

  @override
  State<OpenerNetDoorMobileApp> createState() => _OpenerNetDoorMobileAppState();
}

class _OpenerNetDoorMobileAppState extends State<OpenerNetDoorMobileApp> {
  late final AppState _state;

  @override
  void initState() {
    super.initState();
    _state = AppState(
      storage: MemorySecureStorage(),
      vpnEngine: StubVpnEngine(),
    )..loadSession();
  }

  @override
  Widget build(BuildContext context) {
    final baseTheme = ThemeData(
      useMaterial3: true,
      colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF2563EB), brightness: Brightness.dark),
      scaffoldBackgroundColor: const Color(0xFF0B1220),
      cardTheme: const CardThemeData(color: Color(0xFF111A2F)),
    );

    return MaterialApp(
      title: 'Opener NetDoor Mobile',
      theme: baseTheme,
      home: AnimatedBuilder(
        animation: _state,
        builder: (context, _) {
          if (!_state.hasProfile) {
            return LoginImportScreen(state: _state);
          }
          return MainTabs(state: _state);
        },
      ),
    );
  }
}

class MainTabs extends StatefulWidget {
  const MainTabs({super.key, required this.state});

  final AppState state;

  @override
  State<MainTabs> createState() => _MainTabsState();
}

class _MainTabsState extends State<MainTabs> {
  int _index = 0;

  @override
  Widget build(BuildContext context) {
    final pages = <Widget>[
      HomeScreen(state: widget.state),
      NodesScreen(state: widget.state),
      DiagnosticsScreen(state: widget.state),
      SettingsScreen(state: widget.state),
    ];

    return Scaffold(
      body: pages[_index],
      bottomNavigationBar: NavigationBar(
        selectedIndex: _index,
        onDestinationSelected: (value) => setState(() => _index = value),
        destinations: const <NavigationDestination>[
          NavigationDestination(icon: Icon(Icons.power_settings_new), label: 'Home'),
          NavigationDestination(icon: Icon(Icons.public), label: 'Nodes'),
          NavigationDestination(icon: Icon(Icons.stethoscope), label: 'Diagnostics'),
          NavigationDestination(icon: Icon(Icons.settings), label: 'Settings'),
        ],
      ),
    );
  }
}
