import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/channel/fake_navivox_channel.dart';

class ServersScreen extends ConsumerWidget {
  const ServersScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final servers = ref.watch(fakeNavivoxChannelProvider).state.servers;

    return Scaffold(
      appBar: AppBar(title: const Text('Servers')),
      body: ListView(
        children: [
          for (final server in servers)
            ListTile(
              leading: const Icon(Icons.dns),
              title: Text(server.name),
              subtitle: Text(server.status),
            ),
        ],
      ),
    );
  }
}
