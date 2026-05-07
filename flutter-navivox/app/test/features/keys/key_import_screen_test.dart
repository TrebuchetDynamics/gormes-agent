import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/features/keys/screens/key_import_screen.dart';
import 'package:navivox/features/keys/services/key_store.dart';

const _termiusExport = '''
{
  "hosts": [
    {"id": "h1", "label": "prod", "hostname": "prod.example.com",
     "port": 22, "username": "deploy", "identity": "i1"}
  ],
  "identities": [
    {"id": "i1", "label": "shared",
     "private_key": "-----BEGIN OPENSSH PRIVATE KEY-----\\nstub\\n-----END OPENSSH PRIVATE KEY-----",
     "public_key": "ssh-ed25519 AAA...prod", "passphrase": null}
  ]
}
''';

void main() {
  testWidgets('KeyImportScreen imports a Termius JSON paste and lists hosts',
      (tester) async {
    final identityStore = InMemoryIdentityStore();
    final serverStore = InMemoryServerStore();

    await tester.pumpWidget(MaterialApp(
      home: KeyImportScreen(
        identityStore: identityStore,
        serverStore: serverStore,
      ),
    ));

    expect(find.text('Paste a Termius JSON export'), findsOneWidget);
    expect(find.byKey(const Key('imported-hosts-list')), findsNothing);

    await tester.enterText(find.byType(TextField), _termiusExport);
    await tester.tap(find.text('Import'));
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('imported-hosts-list')), findsOneWidget);
    expect(find.text('prod'), findsOneWidget);
    expect(find.text('deploy@prod.example.com:22'), findsOneWidget);
    expect(find.textContaining('Imported'), findsOneWidget);
  });

  testWidgets('shows an error banner when the JSON is malformed',
      (tester) async {
    await tester.pumpWidget(MaterialApp(
      home: KeyImportScreen(
        identityStore: InMemoryIdentityStore(),
        serverStore: InMemoryServerStore(),
      ),
    ));

    await tester.enterText(find.byType(TextField), 'not-json');
    await tester.tap(find.text('Import'));
    await tester.pump();

    expect(find.textContaining('Could not parse'), findsOneWidget);
  });
}
