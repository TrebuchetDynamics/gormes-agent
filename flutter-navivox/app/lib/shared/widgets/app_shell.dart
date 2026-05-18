import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../router/app_routes.dart';

class AppShell extends StatelessWidget {
  const AppShell({required this.location, required this.child, super.key});

  final String location;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final destinations = [
      _Destination(AppRoutes.chats, Icons.chat_bubble, 'Chats'),
      _Destination(AppRoutes.servers, Icons.dns, 'Servers'),
      _Destination(AppRoutes.agents, Icons.smart_toy, 'Agents'),
      _Destination(AppRoutes.config, Icons.settings, 'Config'),
      _Destination(AppRoutes.settings, Icons.keyboard_voice, 'Settings'),
    ];
    final selectedIndex = destinations.indexWhere(
      (destination) => location.startsWith(destination.path),
    );

    return Scaffold(
      body: child,
      bottomNavigationBar: NavigationBar(
        selectedIndex: selectedIndex < 0 ? 0 : selectedIndex,
        onDestinationSelected: (index) {
          context.go(destinations[index].path);
        },
        destinations: [
          for (final destination in destinations)
            NavigationDestination(
              icon: Icon(destination.icon),
              label: destination.label,
            ),
        ],
      ),
    );
  }
}

class _Destination {
  const _Destination(this.path, this.icon, this.label);

  final String path;
  final IconData icon;
  final String label;
}
