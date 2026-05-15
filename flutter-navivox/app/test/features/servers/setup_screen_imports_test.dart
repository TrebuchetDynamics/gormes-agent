import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/features/keys/providers/key_store_providers.dart';
import 'package:navivox/features/keys/services/key_store.dart';
import 'package:navivox/features/servers/screens/setup_screen.dart';

void main() {
  testWidgets('setup screen omits the imported-servers section when store is empty',
      (tester) async {
    final identityStore = InMemoryIdentityStore();
    final serverStore = InMemoryServerStore();

    await tester.pumpWidget(ProviderScope(
      overrides: [
        identityStoreProvider.overrideWithValue(identityStore),
        serverStoreProvider.overrideWithValue(serverStore),
      ],
      child: const MaterialApp(home: SetupScreen()),
    ));
    await tester.pump();

    expect(find.text('Recently imported'), findsNothing);
    expect(find.text('Connect to Gormes gateway'), findsOneWidget);
  });

  testWidgets('setup screen lists imported servers when present',
      (tester) async {
    final identityStore = InMemoryIdentityStore();
    final serverStore = InMemoryServerStore();
    await serverStore.upsert(const StoredServer(
      termiusId: 's-1',
      label: 'staging',
      hostname: 'stg.example.com',
      port: 22,
      username: 'deploy',
    ));
    await serverStore.upsert(const StoredServer(
      termiusId: 's-2',
      label: 'prod',
      hostname: 'prod.example.com',
      port: 2222,
      username: 'ops',
    ));

    await tester.pumpWidget(ProviderScope(
      overrides: [
        identityStoreProvider.overrideWithValue(identityStore),
        serverStoreProvider.overrideWithValue(serverStore),
      ],
      child: const MaterialApp(home: SetupScreen()),
    ));
    await tester.pump();

    expect(find.text('Recently imported'), findsOneWidget);
    expect(find.text('staging'), findsOneWidget);
    expect(find.text('deploy@stg.example.com:22'), findsOneWidget);
    expect(find.text('prod'), findsOneWidget);
    expect(find.text('ops@prod.example.com:2222'), findsOneWidget);
  });
}
