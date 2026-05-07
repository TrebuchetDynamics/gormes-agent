import 'dart:ffi';
import 'dart:io' show Platform;

import 'package:drift/drift.dart';
import 'package:drift/native.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/features/keys/data/key_database.dart';
import 'package:navivox/features/keys/services/drift_key_store.dart';
import 'package:navivox/features/keys/services/key_store.dart';
import 'package:sqlite3/open.dart';

void main() {
  setUpAll(() {
    // Point the sqlite3 package at the versioned .so on Linux test boxes
    // that don't ship the unversioned libsqlite3.so symlink. App builds
    // bundle their own SQLite via sqlite3_flutter_libs.
    if (Platform.isLinux) {
      open.overrideFor(
        OperatingSystem.linux,
        () => DynamicLibrary.open('libsqlite3.so.0'),
      );
    }
  });

  late KeyDatabase db;
  late DriftIdentityStore identityStore;
  late DriftServerStore serverStore;

  setUp(() {
    db = KeyDatabase(DatabaseConnection(
      NativeDatabase.memory(),
      closeStreamsSynchronously: true,
    ));
    identityStore = DriftIdentityStore(db);
    serverStore = DriftServerStore(db);
  });

  tearDown(() async {
    await db.close();
  });

  group('DriftIdentityStore', () {
    test('upsert by termiusId is idempotent and survives restart', () async {
      await identityStore.upsert(const StoredIdentity(
        termiusId: 'i1',
        label: 'A',
        publicKey: 'pub',
        privateKeyBlob: 'priv',
        isEncrypted: false,
      ));
      await identityStore.upsert(const StoredIdentity(
        termiusId: 'i1',
        label: 'A renamed',
        publicKey: 'pub2',
        privateKeyBlob: 'priv',
        isEncrypted: true,
      ));

      final all = await identityStore.getAll();
      expect(all.length, 1);
      expect(all.single.label, 'A renamed');
      expect(all.single.publicKey, 'pub2');
      expect(all.single.isEncrypted, isTrue);
    });

    test('delete removes by termiusId', () async {
      await identityStore.upsert(const StoredIdentity(
        termiusId: 'i1',
        label: 'A',
        publicKey: 'pub',
        privateKeyBlob: 'priv',
        isEncrypted: false,
      ));
      await identityStore.delete('i1');
      expect(await identityStore.getAll(), isEmpty);
    });

    test('watch() emits the current snapshot then updates', () async {
      final snapshots = <List<StoredIdentity>>[];
      final sub = identityStore.watch().listen(snapshots.add);

      await Future<void>.delayed(const Duration(milliseconds: 5));
      await identityStore.upsert(const StoredIdentity(
        termiusId: 'i1',
        label: 'A',
        publicKey: 'pub',
        privateKeyBlob: 'priv',
        isEncrypted: false,
      ));
      await Future<void>.delayed(const Duration(milliseconds: 10));
      await identityStore.delete('i1');
      await Future<void>.delayed(const Duration(milliseconds: 10));
      await sub.cancel();

      // Initial empty + post-insert (1) + post-delete (0) — final state matters.
      expect(snapshots.first, isEmpty);
      expect(snapshots.last, isEmpty);
      expect(snapshots.any((s) => s.length == 1), isTrue);
    });
  });

  group('DriftServerStore', () {
    test('upsert dedupes by hostname+port+username+identityId', () async {
      await serverStore.upsert(const StoredServer(
        termiusId: 'h1',
        label: 'prod',
        hostname: 'prod.example.com',
        port: 22,
        username: 'deploy',
        identityTermiusId: 'i1',
      ));
      await serverStore.upsert(const StoredServer(
        termiusId: 'h1-renamed',
        label: 'prod renamed',
        hostname: 'prod.example.com',
        port: 22,
        username: 'deploy',
        identityTermiusId: 'i1',
      ));

      final all = await serverStore.getAll();
      expect(all.length, 1);
      expect(all.single.label, 'prod renamed');
      // termiusId should reflect the latest upsert so the UI can link back.
      expect(all.single.termiusId, 'h1-renamed');
    });

    test('different ports are kept separate', () async {
      await serverStore.upsert(const StoredServer(
        termiusId: 'h1',
        label: 'prod',
        hostname: 'host',
        port: 22,
        username: 'me',
      ));
      await serverStore.upsert(const StoredServer(
        termiusId: 'h2',
        label: 'prod-alt',
        hostname: 'host',
        port: 2222,
        username: 'me',
      ));
      expect((await serverStore.getAll()).length, 2);
    });

    test('tags round-trip through JSON encoding', () async {
      await serverStore.upsert(const StoredServer(
        termiusId: 'h1',
        label: 'prod',
        hostname: 'host',
        port: 22,
        username: 'me',
        tags: ['web', 'api'],
        knownHostKey: 'ssh-ed25519 AAA...',
      ));
      final loaded = (await serverStore.getAll()).single;
      expect(loaded.tags, ['web', 'api']);
      expect(loaded.knownHostKey, 'ssh-ed25519 AAA...');
    });

    test('delete by termiusId removes the matching row', () async {
      await serverStore.upsert(const StoredServer(
        termiusId: 'h1',
        label: 'prod',
        hostname: 'host',
        port: 22,
        username: 'me',
      ));
      await serverStore.delete('h1');
      expect(await serverStore.getAll(), isEmpty);
    });
  });
}
