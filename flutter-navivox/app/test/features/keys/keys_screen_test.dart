import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/features/keys/providers/key_store_providers.dart';
import 'package:navivox/features/keys/screens/keys_screen.dart';
import 'package:navivox/features/keys/services/key_store.dart';

void main() {
  testWidgets(
      'shows an empty-state hint when no identities or servers are imported',
      (tester) async {
    final identityStore = InMemoryIdentityStore();
    final serverStore = InMemoryServerStore();

    await tester.pumpWidget(ProviderScope(
      overrides: [
        identityStoreProvider.overrideWithValue(identityStore),
        serverStoreProvider.overrideWithValue(serverStore),
      ],
      child: const MaterialApp(home: KeysScreen()),
    ));
    await tester.pump();

    expect(find.text('No keys imported yet'), findsOneWidget);
  });

  testWidgets('lists imported servers with hostname and username',
      (tester) async {
    final identityStore = InMemoryIdentityStore();
    final serverStore = InMemoryServerStore();
    await serverStore.upsert(const StoredServer(
      termiusId: 's-1',
      label: 'prod-1',
      hostname: 'box.example.com',
      port: 22,
      username: 'ops',
      identityTermiusId: 'id-1',
      group: 'prod',
      tags: ['critical'],
    ));

    await tester.pumpWidget(ProviderScope(
      overrides: [
        identityStoreProvider.overrideWithValue(identityStore),
        serverStoreProvider.overrideWithValue(serverStore),
      ],
      child: const MaterialApp(home: KeysScreen()),
    ));
    await tester.pump();

    expect(find.text('prod-1'), findsOneWidget);
    expect(find.text('ops@box.example.com:22'), findsOneWidget);
  });

  testWidgets('lists imported identities by label', (tester) async {
    final identityStore = InMemoryIdentityStore();
    await identityStore.upsert(const StoredIdentity(
      termiusId: 'id-1',
      label: 'work-laptop-key',
      publicKey: null,
      privateKeyBlob: null,
      isEncrypted: false,
    ));
    final serverStore = InMemoryServerStore();

    await tester.pumpWidget(ProviderScope(
      overrides: [
        identityStoreProvider.overrideWithValue(identityStore),
        serverStoreProvider.overrideWithValue(serverStore),
      ],
      child: const MaterialApp(home: KeysScreen()),
    ));
    await tester.pump();

    expect(find.text('work-laptop-key'), findsOneWidget);
  });
}
