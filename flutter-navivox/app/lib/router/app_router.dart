import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../core/channel/navivox_channel_provider.dart';
import '../features/agents/screens/agents_screen.dart';
import '../features/chat/screens/chat_screen.dart';
import '../features/config/screens/config_screen.dart';
import '../features/keys/screens/keys_screen.dart';
import '../features/servers/screens/servers_screen.dart';
import '../features/servers/screens/setup_screen.dart';
import '../shared/widgets/app_shell.dart';
import 'app_routes.dart';

final routerProvider = Provider<GoRouter>((ref) {
  final channel = ref.watch(navivoxChannelProvider);

  return GoRouter(
    initialLocation: AppRoutes.chats,
    refreshListenable: channel,
    redirect: (context, state) {
      final hasServers = channel.state.servers.isNotEmpty;
      final isSetup = state.matchedLocation.startsWith(AppRoutes.setup);

      if (!hasServers && !isSetup) {
        return AppRoutes.setup;
      }
      if (hasServers && isSetup) {
        return AppRoutes.chats;
      }
      return null;
    },
    routes: [
      GoRoute(
        path: AppRoutes.setup,
        builder: (context, state) => const SetupScreen(),
      ),
      ShellRoute(
        builder: (context, state, child) =>
            AppShell(location: state.matchedLocation, child: child),
        routes: [
          GoRoute(
            path: AppRoutes.chats,
            builder: (context, state) => const ChatScreen(),
          ),
          GoRoute(
            path: AppRoutes.servers,
            builder: (context, state) => const ServersScreen(),
          ),
          GoRoute(
            path: AppRoutes.agents,
            builder: (context, state) => const AgentsScreen(),
          ),
          GoRoute(
            path: AppRoutes.config,
            builder: (context, state) => const ConfigScreen(),
          ),
          GoRoute(
            path: AppRoutes.keys,
            builder: (context, state) => const KeysScreen(),
          ),
        ],
      ),
    ],
    errorBuilder: (context, state) => Scaffold(
      appBar: AppBar(title: const Text('Navivox')),
      body: Center(child: Text('Route not found: ${state.uri.path}')),
    ),
  );
});
