import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../providers/key_store_providers.dart';
import '../services/key_store.dart';

class KeysScreen extends ConsumerWidget {
  const KeysScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final identityStore = ref.watch(identityStoreProvider);
    final serverStore = ref.watch(serverStoreProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Keys')),
      body: StreamBuilder<List<StoredIdentity>>(
        stream: identityStore.watch(),
        builder: (context, identitiesSnapshot) {
          final identities = identitiesSnapshot.data ?? const <StoredIdentity>[];
          return StreamBuilder<List<StoredServer>>(
            stream: serverStore.watch(),
            builder: (context, serversSnapshot) {
              final servers = serversSnapshot.data ?? const <StoredServer>[];
              if (identities.isEmpty && servers.isEmpty) {
                return const Center(child: Text('No keys imported yet'));
              }
              return ListView(
                padding: const EdgeInsets.symmetric(vertical: 8),
                children: [
                  if (servers.isNotEmpty) ...[
                    const Padding(
                      padding: EdgeInsets.fromLTRB(16, 8, 16, 4),
                      child: Text('Servers',
                          style: TextStyle(fontWeight: FontWeight.bold)),
                    ),
                    for (final s in servers)
                      ListTile(
                        leading: const Icon(Icons.dns),
                        title: Text(s.label),
                        subtitle: Text('${s.username}@${s.hostname}:${s.port}'),
                      ),
                  ],
                  if (identities.isNotEmpty) ...[
                    const Padding(
                      padding: EdgeInsets.fromLTRB(16, 16, 16, 4),
                      child: Text('Identities',
                          style: TextStyle(fontWeight: FontWeight.bold)),
                    ),
                    for (final id in identities)
                      ListTile(
                        leading: const Icon(Icons.key),
                        title: Text(id.label),
                      ),
                  ],
                ],
              );
            },
          );
        },
      ),
    );
  }
}
