import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/features/keys/services/key_store.dart';
import 'package:navivox/features/keys/services/termius_importer.dart';

const _exportJson = '''
{
  "hosts": [
    {"id": "h1", "label": "prod", "hostname": "prod.example.com", "port": 22,
     "username": "deploy", "identity": "i1"},
    {"id": "h2", "label": "dev", "hostname": "dev.example.com", "port": 2222,
     "username": "dev", "identity": "i1"}
  ],
  "identities": [
    {"id": "i1", "label": "shared key",
     "private_key": "-----BEGIN OPENSSH PRIVATE KEY-----\\nstub\\n-----END OPENSSH PRIVATE KEY-----",
     "public_key": "ssh-ed25519 AAA...prod", "passphrase": null}
  ]
}
''';

void main() {
  group('InMemoryIdentityStore', () {
    test('upsert by termiusId is idempotent', () async {
      final store = InMemoryIdentityStore();

      await store.upsert(const StoredIdentity(
        termiusId: 'i1',
        label: 'A',
        publicKey: 'pub',
        privateKeyBlob: 'priv',
        isEncrypted: false,
      ));
      await store.upsert(const StoredIdentity(
        termiusId: 'i1',
        label: 'A renamed',
        publicKey: 'pub',
        privateKeyBlob: 'priv',
        isEncrypted: false,
      ));

      final all = await store.getAll();
      expect(all.length, 1);
      expect(all.single.label, 'A renamed');
    });

    test('watch() emits on every upsert and delete', () async {
      final store = InMemoryIdentityStore();
      final snapshots = <List<StoredIdentity>>[];
      final sub = store.watch().listen(snapshots.add);

      await store.upsert(const StoredIdentity(
        termiusId: 'i1',
        label: 'A',
        publicKey: 'pub',
        privateKeyBlob: 'priv',
        isEncrypted: false,
      ));
      await store.delete('i1');
      await Future<void>.delayed(const Duration(milliseconds: 5));
      await sub.cancel();

      expect(snapshots.map((s) => s.length).toList(), [0, 1, 0]);
    });
  });

  group('InMemoryServerStore', () {
    test('upsert dedupes by hostname+port+username+identityId', () async {
      final store = InMemoryServerStore();

      await store.upsert(const StoredServer(
        termiusId: 'h1',
        label: 'prod',
        hostname: 'prod.example.com',
        port: 22,
        username: 'deploy',
        identityTermiusId: 'i1',
      ));
      await store.upsert(const StoredServer(
        termiusId: 'h1-renamed',
        label: 'prod renamed',
        hostname: 'prod.example.com',
        port: 22,
        username: 'deploy',
        identityTermiusId: 'i1',
      ));

      final all = await store.getAll();
      expect(all.length, 1);
      expect(all.single.label, 'prod renamed');
    });

    test('different ports are not deduped together', () async {
      final store = InMemoryServerStore();
      await store.upsert(const StoredServer(
        termiusId: 'h1',
        label: 'prod',
        hostname: 'host',
        port: 22,
        username: 'me',
      ));
      await store.upsert(const StoredServer(
        termiusId: 'h2',
        label: 'prod-alt',
        hostname: 'host',
        port: 2222,
        username: 'me',
      ));

      expect((await store.getAll()).length, 2);
    });
  });

  group('TermiusImportService', () {
    test('imports identities then servers, returning counts', () async {
      final identityStore = InMemoryIdentityStore();
      final serverStore = InMemoryServerStore();
      final service = TermiusImportService(
        identityStore: identityStore,
        serverStore: serverStore,
      );

      final result = service.importParsed(TermiusImporter.parse(jsonDecode(_exportJson)));

      expect(result.identitiesImported, 1);
      expect(result.serversImported, 2);
      expect((await identityStore.getAll()).single.termiusId, 'i1');
      expect((await serverStore.getAll()).map((s) => s.label).toList(),
          ['prod', 'dev']);
    });

    test('rerunning the import is idempotent', () async {
      final identityStore = InMemoryIdentityStore();
      final serverStore = InMemoryServerStore();
      final service = TermiusImportService(
        identityStore: identityStore,
        serverStore: serverStore,
      );

      service.importParsed(TermiusImporter.parse(jsonDecode(_exportJson)));
      service.importParsed(TermiusImporter.parse(jsonDecode(_exportJson)));

      expect((await identityStore.getAll()).length, 1);
      expect((await serverStore.getAll()).length, 2);
    });
  });
}
